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
)

func ensureDirs(path string) error {
	for _, dir := range []string{
		"objects", "incomplete", "tmp", "locks",
	} {
		os.MkdirAll(filepath.Join(path, dir), 0755)
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
	umask := setPermissions(gitdir)
	lfsPath := filepath.Join(gitdir, "lfs")
	if err := ensureDirs(path); err != nil {
		return err
	}
	handler := transfer.NewPktline(r, w)
	if err := handler.WritePacketText("version=1"); err != nil {
		return err
	}
	if err := handler.WriteFlush(); err != nil {
		return err
	}
	now := time.Now()
	backend := local.New(lfsPath, umask, &now)
	p := transfer.NewProcessor(handler, backend)
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

func main() {
	if debug := os.Getenv("GIT_LFS_TRANSFER_DEBUG"); debug == "true" {
		transfer.Debug = true
	}

	transfer.Logf("git-lfs-transfer %s", "v1")
	defer transfer.Log("git-lfs-transfer completed")
	if err := run(os.Stdin, os.Stdout, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, usage())
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
