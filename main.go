package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/git-lfs-transfer/internal/local"
	"github.com/charmbracelet/git-lfs-transfer/transfer"
	"github.com/rubyist/tracerx"
)

func ensureDirs(path string) error {
	for _, dir := range []string{
		"objects", "incomplete", "tmp", "locks",
	} {
		os.MkdirAll(filepath.Join(path, dir), 0777)
	}
	return nil
}

func run(r io.Reader, w io.Writer, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("expected 2 arguments, got %d", len(args))
	}
	path := args[0]
	op := args[1]
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	gitdir := path
	if !strings.HasSuffix(path, ".git") {
		gitdir = filepath.Join(path, ".git")
	}
	lfsPath := filepath.Join(gitdir, "lfs")
	if err := ensureDirs(lfsPath); err != nil {
		return err
	}
	umask := setPermissions(gitdir)
	handler := transfer.NewPktline(r, w)
	if err := handler.WritePacketText("version=1"); err != nil {
		transfer.Logf("error sending capabilites: %v", err)
	}
	if err := handler.WriteFlush(); err != nil {
		transfer.Logf("error flushing capabilites: %v", err)
	}
	now := time.Now()
	transfer.Logf("umask %o", umask)
	backend := local.New(lfsPath, umask, &now)
	p := transfer.NewProcessor(handler, backend)
	defer transfer.Log("done processing commands")
	switch op {
	case "upload":
		return p.ProcessCommands(transfer.UploadOperation)
	case "download":
		return p.ProcessCommands(transfer.DownloadOperation)
	default:
		return fmt.Errorf("unknown operation %q", op)
	}
}

func usage() string {
	return `Git LFS SSH transfer agent

Usage:
  git-lfs-transfer PATH OPERATION
`
}

func init() {
	tracerx.DefaultKey = "GIT"
	tracerx.Prefix = "trace git-lfs-transfer: "
}

func main() {
	done := make(chan os.Signal, 1)
	errc := make(chan error, 1)

	setup(done)
	transfer.Logf("git-lfs-transfer %s", "v1")
	defer transfer.Log("git-lfs-transfer completed")
	go func() {
		errc <- run(os.Stdin, os.Stdout, os.Args[1:])
	}()

	select {
	case s := <-done:
		transfer.Logf("signal %q received", s)
	case err := <-errc:
		transfer.Log("done running")
		if err != nil {
			fmt.Fprintf(os.Stderr, usage())
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
}
