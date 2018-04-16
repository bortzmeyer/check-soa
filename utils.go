package main

import "log"

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
