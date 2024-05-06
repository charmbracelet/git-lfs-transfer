package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/git-lfs-transfer/transfer"
	"github.com/rubyist/tracerx"
)

type tracerxLogger struct{}

// Log logs the given arguments if Debug is true.
func (*tracerxLogger) Log(msg string, kv ...interface{}) {
	format := msg
	for i := 0; i < len(kv); i += 2 {
		format += " %s=%v"
	}
	tracerx.Printf(format, kv...)
}

var logger = new(tracerxLogger)

func init() {
	tracerx.DefaultKey = "GIT"
	tracerx.Prefix = "trace git-lfs-transfer: "
}

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, Usage())
		fmt.Fprintf(os.Stderr, "expected 2 arguments, got %d\n", len(args)-1)
		os.Exit(1)
	}
	if err := Command(os.Stdin, os.Stdout, os.Stderr, args[1:]...); err != nil {
		fmt.Fprint(os.Stderr, Usage())
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, err)
		switch {
		case errors.Is(err, transfer.ErrConflict):
			os.Exit(1)
		default:
			os.Exit(2)
		}
	}
}
