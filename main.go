package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/git-lfs-transfer/internal/local"
	"github.com/charmbracelet/git-lfs-transfer/transfer"
	"github.com/spf13/cobra"
)

var (
	// RootCmd is the root command for the git-lfs-transfer command.
	rootCmd = &cobra.Command{
		Use:   "git-lfs-transfer PATH OPERATION",
		Short: "Git LFS SSH transfer agent",
		Args:  cobra.ExactArgs(2),
		RunE:  run,
	}
)

func ensureDirs(path string) error {
	for _, dir := range []string{
		"objects", "incomplete", "tmp", "locks",
	} {
		os.MkdirAll(filepath.Join(path, dir), 0755)
	}
	return nil
}

func run(cmd *cobra.Command, args []string) error {
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
	handler := transfer.NewPktline(cmd.InOrStdin(), cmd.OutOrStdout())
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
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
