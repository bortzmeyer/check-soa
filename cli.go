package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	v4only         bool
	v6only         bool
	fDebug         bool
	version        bool
	quiet          bool
	noedns         bool
	nsid           bool
	bufsize        int
	tcp            bool
	nodnssec       bool
	recursion      bool
	noauthrequired bool
	times          bool
	help           bool
	timeoutI       float64
	maxTrials      int
	nslist         map[string]nameServer
	nslists        string
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s [options] ZONE\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.BoolVar(&v4only, "4", false, "Use only IPv4")
	flag.BoolVar(&v6only, "6", false, "Use only IPv6")
	flag.BoolVar(&help, "h", false, "Print help")
	flag.BoolVar(&fDebug, "d", false, "Debugging")
	flag.BoolVar(&version, "v", false, "Displays version of the code")
	flag.BoolVar(&quiet, "q", false, "Quiet mode, display only errors")
	flag.BoolVar(&noedns, "r", false, "Disable EDNS format")
	flag.BoolVar(&nsid, "nsid", false, "Enable NSID option")
	flag.IntVar(&bufsize, "b", int(EDNSBUFFERSIZE), "EDNS buffer size")
	flag.BoolVar(&tcp, "tcp", false, "Use TCP")
	// DNSSEC DO is on by default, to detect firewall or
	// fragmentation problems.
	flag.BoolVar(&nodnssec, "s", false, "Disable DNSSEC (DO bit)")
	flag.BoolVar(&recursion, "e", false, "Set recursion on")
	flag.BoolVar(&noauthrequired, "a", false, "Do not require an authoritative answer")
	flag.BoolVar(&times, "i", false, "Display the response time of servers")
	flag.Float64Var(&timeoutI, "t", float64(TIMEOUT), "Timeout in seconds")
	flag.IntVar(&maxTrials, "n", int(MAXTRIALS), "Number of trials before giving in")
	flag.StringVar(&nslists, "ns", "", "Name servers to query")
	flag.Parse()
}
