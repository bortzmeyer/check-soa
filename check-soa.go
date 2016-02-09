// A simple program to have rapidly an idea of the health of a DNS
// zone. It queries each name server of the zone for the SOA record and
// displays the value of the serial number for each server.
//
// Stephane Bortzmeyer <bortzmeyer@nic.fr>
package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/miekg/dns"
	"net"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	TIMEOUT         float64 = float64(1.5)
	MAXTRIALS       uint    = 3
	MAX_NAMESERVERS uint    = 20
	MAX_ADDRESSES   uint    = 10
	EDNSBUFFERSIZE  uint16  = 4096
)

type DNSreply struct {
	qname      string
	qtype      uint16
	r          *dns.Msg
	err        error
	nameserver string
	rtt        time.Duration
}

type SOAreply struct {
	name      string
	address   string
	serial    uint32
	retrieved bool
	msg       string
	rtt       time.Duration
}

type nameServer struct {
	name         string
	ips          []string
	globalErrMsg string
	success      []bool
	errMsg       []string
	serial       []uint32
	rtts         []time.Duration
}

type Results map[string]nameServer

var (
	Version = "No Version Provided at compile time"
	/* TODO: make it per-thread? It does not seem necessary, the goroutines
	do not modify it */
	conf           *dns.ClientConfig
	v4only         *bool
	v6only         *bool
	debug          *bool
	version        *bool
	quiet          *bool
	noedns         *bool
	nodnssec       *bool
	recursion      *bool
	noauthrequired *bool
	times          *bool
	timeout        time.Duration
	maxTrials      *int
	nslist         map[string]nameServer
	useZoneNS      bool
)

func localQuery(mychan chan DNSreply, qname string, qtype uint16) {
	var result DNSreply
	var trials uint
	result.qname = qname
	result.qtype = qtype
	result.r = nil
	result.err = errors.New("No name server to answer the question")
	localm := new(dns.Msg)
	localm.Id = dns.Id()
	localm.RecursionDesired = true
	localm.Question = make([]dns.Question, 1)
	localc := new(dns.Client)
	localc.ReadTimeout = timeout
	localm.Question[0] = dns.Question{qname, qtype, dns.ClassINET}
Tests:
	for trials = 0; trials < uint(*maxTrials); trials++ {
	Resolvers:
		for serverIndex := range conf.Servers {
			server := conf.Servers[serverIndex]
			result.nameserver = server
			// Brackets around the server address are necessary for IPv6 name servers
			r, rtt, err := localc.Exchange(localm, "["+server+"]:"+conf.Port) // Do not use net.JoinHostPort, see https://github.com/bortzmeyer/check-soa/commit/3e4edb13855d8c4016768796b2892aa83eda1933#commitcomment-2355543
			if r == nil {
				result.r = nil
				result.err = err
				if strings.Contains(err.Error(), "timeout") {
					// Try another resolver
					break Resolvers
				} else { // We give in
					break Tests
				}
			} else {
				result.rtt = rtt
				if r.Rcode == dns.RcodeSuccess {
					// TODO: as a result, NODATA (NOERROR/ANSWER=0) are silently ignored (try "foo", for instance, the name exists but no IP address)
					// TODO: for rcodes like SERVFAIL, trying another resolver could make sense
					result.r = r
					result.err = nil
					break Tests
				} else {
					// All the other codes are errors. Yes, it may
					// happens that one resolver returns REFUSED
					// and the others work but we do not handle
					// this case. TODO: delete the resolver from
					// the list and try another one
					result.r = r
					result.err = errors.New(dns.RcodeToString[r.Rcode])
					break Tests
				}
			}
		}
	}
	if *debug {
		fmt.Printf("DEBUG: end of DNS request \"%s\" / %d\n", qname, qtype)
	}
	mychan <- result
}

func soaQuery(mychan chan SOAreply, zone string, name string, server string) {
	var result SOAreply
	var trials uint
	result.retrieved = false
	result.name = name
	result.address = server
	result.msg = "UNKNOWN"
	m := new(dns.Msg)
	if !*noedns {
		m.SetEdns0(EDNSBUFFERSIZE, !*nodnssec)
	}
	m.Id = dns.Id()
	if *recursion {
		m.RecursionDesired = true
	} else {
		m.RecursionDesired = false
	}
	m.Question = make([]dns.Question, 1)
	c := new(dns.Client)
	c.ReadTimeout = timeout
	m.Question[0] = dns.Question{zone, dns.TypeSOA, dns.ClassINET}
	nsAddressPort := ""
	nsAddressPort = net.JoinHostPort(server, "53")
	if *debug {
		fmt.Printf("DEBUG Querying SOA from %s\n", nsAddressPort)
	}
	for trials = 0; trials < uint(*maxTrials); trials++ {
		soa, rtt, err := c.Exchange(m, nsAddressPort)
		if soa == nil {
			result.rtt = 0
			result.msg = fmt.Sprintf("%s", err.Error())
		} else {
			result.rtt = rtt
			if soa.Rcode != dns.RcodeSuccess {
				result.msg = dns.RcodeToString[soa.Rcode]
				break
			} else {
				if len(soa.Answer) == 0 { /* May happen if the server is a recursor, not authoritative, since we query with RD=0 */
					result.msg = "0 answer"
					break
				} else {
					gotSoa := false
					for _, rsoa := range soa.Answer {
						switch rsoa.(type) {
						case *dns.SOA:
							if *noauthrequired || soa.MsgHdr.Authoritative {
								result.retrieved = true
								result.serial = rsoa.(*dns.SOA).Serial
								result.msg = "OK"
							} else {
								result.msg = "Not authoritative"
							}
							gotSoa = true
						case *dns.CNAME: /* Bad practice but common */
							fmt.Printf("Apparently not a zone but an alias\n")
							os.Exit(1)
						case *dns.RRSIG:
							/* Ignore them. See bug #8 */
						default:
							// TODO: a name server can send us other RR types.
							fmt.Printf("Internal error when processing %s, unexpected record type\n", rsoa)
							os.Exit(1)
						}
						if !gotSoa {
							result.msg = "No SOA record in reply"
						}
						break
					}
				}
			}
		}
	}
	mychan <- result
}

func masterTask(zone string, nameservers map[string]nameServer) (uint, uint, bool, Results) {
	var (
		numRequests uint
	)
	success := true
	addressChannel := make(chan DNSreply)
	soaChannel := make(chan SOAreply)
	numNS := uint(0)
	numAddrNS := uint(0)
	results := make(Results)
	for name := range nameservers {
		if !*v6only {
			go localQuery(addressChannel, name, dns.TypeA)
		}
		if !*v4only {
			go localQuery(addressChannel, name, dns.TypeAAAA)
		}
		numNS += 1
	}
	if *v6only || *v4only {
		numRequests = numNS
	} else {
		numRequests = numNS * 2
	}
	for i := uint(0); i < numRequests; i++ {
		addrResult := <-addressChannel
		addrFamily := "IPv6"
		if addrResult.qtype == dns.TypeA {
			addrFamily = "IPv4"
		}
		if addrResult.r == nil {
			// TODO We may have different globalErrMsg is it
			// works with IPv4 but not IPv6 (it should not happen but it does)
			nameservers[addrResult.qname] = nameServer{
				name:         addrResult.qname,
				ips:          nil,
				globalErrMsg: fmt.Sprintf("Cannot get the %s address: %s", addrFamily, addrResult.err)}
			success = false
		} else {
			if addrResult.r.Rcode != dns.RcodeSuccess {
				nameservers[addrResult.qname] = nameServer{
					name:         addrResult.qname,
					ips:          nil,
					globalErrMsg: fmt.Sprintf("Cannot get the %s address: %s", addrFamily, dns.RcodeToString[addrResult.r.Rcode])}
				success = false
			} else {
				for j := range addrResult.r.Answer {
					ansa := addrResult.r.Answer[j]
					var ns string
					switch ansa.(type) {
					case *dns.A:
						ns = ansa.(*dns.A).A.String()
						nameservers[addrResult.qname] = nameServer{name: addrResult.qname, ips: append(nameservers[addrResult.qname].ips, ns)}
						numAddrNS += 1
						go soaQuery(soaChannel, zone, addrResult.qname, ns)
					case *dns.AAAA:
						ns = ansa.(*dns.AAAA).AAAA.String()
						nameservers[addrResult.qname] = nameServer{name: addrResult.qname, ips: append(nameservers[addrResult.qname].ips, ns)}
						numAddrNS += 1
						go soaQuery(soaChannel, zone, addrResult.qname, ns)
					}
				}
			}
		}
	}
	for i := uint(0); i < numAddrNS; i++ {
		if *debug {
			fmt.Printf("DEBUG Getting result for ns #%d/%d\n", i+1, numAddrNS)
		}
		soaResult := <-soaChannel
		_, present := results[soaResult.name]
		if !present {
			results[soaResult.name] = nameServer{name: soaResult.name,
				ips:     make([]string, 0),
				success: make([]bool, 0),
				errMsg:  make([]string, 0),
				serial:  make([]uint32, 0),
				rtts:    make([]time.Duration, 0)}
		}
		if !soaResult.retrieved {
			results[soaResult.name] = nameServer{name: soaResult.name,
				ips:     append(results[soaResult.name].ips, soaResult.address),
				success: append(results[soaResult.name].success, false),
				errMsg:  append(results[soaResult.name].errMsg, fmt.Sprintf("%s", soaResult.msg)),
				serial:  append(results[soaResult.name].serial, 0),
				rtts:    append(results[soaResult.name].rtts, soaResult.rtt)}
			success = false
		} else {
			results[soaResult.name] = nameServer{name: soaResult.name,
				ips:     append(results[soaResult.name].ips, soaResult.address),
				success: append(results[soaResult.name].success, true),
				errMsg:  append(results[soaResult.name].errMsg, ""),
				serial:  append(results[soaResult.name].serial, soaResult.serial),
				rtts:    append(results[soaResult.name].rtts, soaResult.rtt)}
		}
	}
	for name := range nameservers {
		if nameservers[name].ips == nil {
			results[name] = nameservers[name]
		}
	}
	return numNS, numAddrNS, success, results
}

func main() {
	var (
		err     error
		nslists *string
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s [options] ZONE\n", os.Args[0])
		flag.PrintDefaults()
	}
	v4only = flag.Bool("4", false, "Use only IPv4")
	v6only = flag.Bool("6", false, "Use only IPv6")
	help := flag.Bool("h", false, "Print help")
	debug = flag.Bool("d", false, "Debugging")
	version = flag.Bool("v", false, "Displays version of the code")
	quiet = flag.Bool("q", false, "Quiet mode, display only errors")
	noedns = flag.Bool("r", false, "Disable EDNS format")
	// DNSSEC DO is on by default, to detect firewall or
	// fragmentation problems.
	nodnssec = flag.Bool("s", false, "Disable DNSSEC (DO bit)")
	recursion = flag.Bool("e", false, "Set recursion on")
	noauthrequired = flag.Bool("a", false, "Do not require an authoritative answer")
	times = flag.Bool("i", false, "Display the response time of servers")
	timeoutI := flag.Float64("t", float64(TIMEOUT), "Timeout in seconds")
	maxTrials = flag.Int("n", int(MAXTRIALS), "Number of trials before giving in")
	nslists = flag.String("ns", "", "Name servers to query")
	flag.Parse()
	if *version {
		fmt.Fprintf(os.Stdout, "%s\n", Version)
		os.Exit(0)
	}
	if *debug && *quiet {
		fmt.Fprintf(os.Stderr, "debug or quiet but not both\n")
		flag.Usage()
		os.Exit(1)
	}
	if *v4only && *v6only {
		fmt.Fprintf(os.Stderr, "v4-only or v6-only but not both\n")
		flag.Usage()
		os.Exit(1)
	}
	if len(flag.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "Only one argument expected, %d arguments received\n", len(flag.Args()))
		flag.Usage()
		os.Exit(1)
	}
	if *timeoutI <= 0 {
		fmt.Fprintf(os.Stderr, "Timeout must be positive, not %d\n", *timeoutI)
		flag.Usage()
		os.Exit(1)
	}
	timeout = time.Duration(*timeoutI * float64(time.Second))
	if *maxTrials <= 0 {
		fmt.Fprintf(os.Stderr, "Number of trials must be positive, not %d\n", *maxTrials)
		flag.Usage()
		os.Exit(1)
	}
	if *help {
		flag.Usage()
		os.Exit(0)
	}
	if *debug {
		fmt.Fprintf(os.Stdout, Version)
	}
	separators, _ := regexp.Compile("\\s+")
	nslista := separators.Split(*nslists, -1)
	// If no nameservers option, Split returns the original (empty) string unmolested
	useZoneNS = len(nslista) == 0 || (len(nslista) == 1 && nslista[0] == "")
	nslist = make(map[string]nameServer)

	zone := dns.Fqdn(flag.Arg(0))
	conf, err = dns.ClientConfigFromFile("/etc/resolv.conf")
	if conf == nil {
		fmt.Printf("Cannot initialize the local resolver: %s\n", err)
		os.Exit(1)
	}

	if useZoneNS {
		nsChan := make(chan DNSreply)
		go localQuery(nsChan, zone, dns.TypeNS)
		nsResult := <-nsChan
		if nsResult.r == nil {
			fmt.Printf("Cannot retrieve the list of name servers for %s: %s\n", zone, nsResult.err)
			os.Exit(1)
		}
		if nsResult.r.Rcode == dns.RcodeNameError {
			fmt.Printf("No such domain %s\n", zone)
			os.Exit(1)
		}
		for i := range nsResult.r.Answer {
			ans := nsResult.r.Answer[i]
			switch ans.(type) {
			case *dns.NS:
				name := ans.(*dns.NS).Ns
				nslist[name] = nameServer{name: name, ips: make([]string, MAX_ADDRESSES)}
			}
		}
	} else {
		for i := range nslista {
			nslist[dns.Fqdn(nslista[i])] = nameServer{name: dns.Fqdn(nslista[i]), ips: make([]string, MAX_ADDRESSES)}
		}
	}
	numNS, numNSaddr, success, results := masterTask(zone, nslist)
	if numNS == 0 {
		fmt.Printf("No NS records for \"%s\". It is probably a domain but not a zone\n", zone)
		os.Exit(1)
	}
	if numNSaddr == 0 {
		fmt.Printf("No IP addresses for name servers of %s\n", zone)
		if *v4only {
			fmt.Printf("May be retry without -4?\n")
		}
		if *v6only {
			fmt.Printf("May be retry without -6?\n")
		}
		os.Exit(1)
	}
	/* TODO: test if all name servers have the same serial ? */
	keys := make([]string, len(results))
	i := 0
	for k, _ := range results {
		keys[i] = k
		i++
	}
	// TODO: allow to sort by response time?
	sort.Strings(keys)
	for k := range keys {
		serverOK := true
		result := results[keys[k]]
		for i := 0; i < len(result.ips); i++ {
			if !result.success[i] {
				serverOK = false
				break
			}
			if result.ips == nil {
				serverOK = false
				break
			}
		}
		if !*quiet || !serverOK {
			fmt.Printf("%s\n", keys[k])
		}
		for i := 0; i < len(result.ips); i++ {
			code := "ERROR"
			msg := ""
			if result.success[i] {
				code = "OK"
				msg = fmt.Sprintf("%d", result.serial[i])
			} else {
				msg = result.errMsg[i]
			}
			if *times && result.rtts[i] != 0 {
				msg = msg + fmt.Sprintf(" (%d ms)", int(float64(result.rtts[i])/1e6))
			}
			if !*quiet || !result.success[i] {
				fmt.Printf("\t%s: %s: %s\n", result.ips[i], code, msg)
			}
		}
		if len(result.ips) == 0 {
			success = false
			fmt.Printf("\t%s\n", result.globalErrMsg)
		}
	}
	if success {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
