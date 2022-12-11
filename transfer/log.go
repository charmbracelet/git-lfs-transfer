package transfer

import "log"

var (
	// Debug is the debug flag.
	Debug = false
)

// Log logs the given arguments if Debug is true.
func Log(v ...interface{}) {
	if Debug {
		log.Print(v...)
	}
}

// Logf logs the given arguments if Debug is true.
func Logf(format string, v ...interface{}) {
	if Debug {
		log.Printf(format, v...)
	}
}
