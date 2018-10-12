package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

// fDebug displays only if fDebug is set
func debug(str string, a ...interface{}) {
	if fDebug {
		log.Printf(str, a...)
	}
}

// fDebug displays only if fVerbose is set
func verbose(str string, a ...interface{}) {
	if !quiet {
		log.Printf(str, a...)
	}
}

// Display an myerror on… Stderr
func myerror(str string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, str, a...)
}

// getPTR does a reverse lookup on IP
func getPTR(addr string) (ptrs []string, err error) {

	r, err := net.LookupAddr(addr)
	debug("ptrs=%v", r)
	return ptrs, nil
}
