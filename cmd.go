package main

import (
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
		if err := os.MkdirAll(filepath.Join(path, dir), os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func run(r io.Reader, w io.Writer, args ...string) error {
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
	handler := transfer.NewPktline(r, w, logger)
	for _, cap := range transfer.Capabilities {
		if err := handler.WritePacketText(cap); err != nil {
			logger.Log("error sending capability", "cap", cap, "err", err)
		}
	}
	if err := handler.WriteFlush(); err != nil {
		logger.Log("error flushing capabilities", "err", err)
	}
	now := time.Now()
	logger.Log("umask", "umask", umask)
	backend := local.New(lfsPath, umask, &now)
	p := transfer.NewProcessor(handler, backend, logger)
	defer logger.Log("done processing commands")
	switch op {
	case "upload":
		return p.ProcessCommands(transfer.UploadOperation)
	case "download":
		return p.ProcessCommands(transfer.DownloadOperation)
	default:
		return fmt.Errorf("unknown operation %q", op)
	}
}

// Usage returns the command usage.
func Usage() string {
	return `Git LFS SSH transfer agent

Usage:
  git-lfs-transfer PATH OPERATION
`
}

func init() {
	tracerx.DefaultKey = "GIT"
	tracerx.Prefix = "trace git-lfs-transfer: "
}

// Command is the main git-lfs-transfer entry.
func Command(stdin io.Reader, stdout io.Writer, stderr io.Writer, args ...string) error {
	done := make(chan os.Signal, 1)
	errc := make(chan error, 1)

	setup(done)
	logger.Log("git-lfs-transfer", "version", "v1")
	defer logger.Log("git-lfs-transfer completed")
	go func() {
		errc <- run(stdin, stdout, args...)
	}()

	select {
	case s := <-done:
		logger.Log("signal received", "signal", s)
	case err := <-errc:
		logger.Log("done running")
		fmt.Fprintln(stderr, Usage())
		fmt.Fprintln(stderr, err)
		if err != nil {
			return err
		}
	}
	return nil
}
