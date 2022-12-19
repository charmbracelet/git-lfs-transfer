package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/git-lfs-transfer/cmd"
	"github.com/charmbracelet/git-lfs-transfer/transfer"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, cmd.Usage())
		fmt.Fprintf(os.Stderr, "expected 2 arguments, got %d\n", len(args)-1)
		os.Exit(1)
	}
	if err := cmd.Command(os.Stdin, os.Stdout, os.Stderr, args[1:]...); err != nil {
		fmt.Fprintf(os.Stderr, cmd.Usage())
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
