package transfer

import (
	"github.com/rubyist/tracerx"
)

var (
	// Debug is the debug flag.
	Debug = false
)

// Log logs the given arguments if Debug is true.
func Log(v ...interface{}) {
	tracerx.Printf("%v", v...)
}

// Logf logs the given arguments if Debug is true.
func Logf(format string, v ...interface{}) {
	tracerx.Printf(format, v...)
}
