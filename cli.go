package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

const (
	version = "No version provided at compile time"
)

var (
	v4only         bool
	v6only         bool
	fDebug         bool
	fVersion       bool
	lVersion       string = version
	quiet          bool
	noedns         bool
	nsid           bool
	bufsize        int
	tcp            bool
	nodnssec       bool
	recursion      bool
	noauthrequired bool
	times          bool
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
	flag.BoolVar(&fDebug, "d", false, "Debugging")
	flag.BoolVar(&fVersion, "v", false, "Displays version of the code")
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
	flag.Float64Var(&timeoutI, "t", TIMEOUT, "Timeout in seconds (for one trial)")
	flag.IntVar(&maxTrials, "n", int(MAXTRIALS), "Number of trials before giving in")
	flag.StringVar(&nslists, "ns", "", "Name servers to query (space-separated)")
	flag.Parse()
}

func checkCliFlags() error {
	if fVersion {
		myerror("%s\n", lVersion)
		return ErrMustExit
	}

	if fDebug && quiet {
		myerror("fDebug or quiet but not both\n")
		return ErrMustExitUsage
	}
	if noedns && nsid {
		myerror("NSID requires EDNS\n")
		return ErrMustExitUsage
	}
	if v4only && v6only {
		myerror("v4-only or v6-only but not both\n")
		return ErrMustExitUsage
	}
	if len(flag.Args()) != 1 {
		myerror("Only one argument expected, %d arguments received\n", len(flag.Args()))
		return ErrMustExitUsage
	}
	if timeoutI <= 0 {
		myerror("Timeout must be positive, not %d\n", timeoutI)
		return ErrMustExitUsage
	}
	timeout = time.Duration(timeoutI * float64(time.Second))
	if maxTrials <= 0 {
		myerror("Number of trials must be positive, not %d\n", maxTrials)
		return ErrMustExitUsage
	}
	return nil
}
