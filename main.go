package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/git-lfs-transfer/cmd"
	"github.com/charmbracelet/git-lfs-transfer/transfer"
)

func main() {
	if err := cmd.Command(os.Stdin, os.Stdout, os.Stderr); err != nil {
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
