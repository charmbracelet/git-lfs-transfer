//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package cmd

import (
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/git-lfs/git-lfs/v3/git"
	"golang.org/x/sys/unix"
)

func setPermissions(path string) os.FileMode {
	var config *git.Configuration
	if strings.HasSuffix(path, ".git") {
		config = git.NewReadOnlyConfig(path, path)
	} else {
		config = git.NewReadOnlyConfig(path, "")
	}
	var val int
	sr := config.Find("core.sharedrepository")
	switch sr {
	case "true", "group":
		val = 0660
	case "all", "world", "everybody":
		val = 0664
	case "false", "umask":
	default:
		v, _ := strconv.ParseUint(sr, 8, 32)
		val = int(v)
	}
	umask := unix.Umask(0)
	unix.Umask(umask)
	if val != 0 {
		umask = 0777 &^ val
	}
	return os.FileMode(umask)
}

func setup(c chan os.Signal) {
	signal.Notify(c, os.Interrupt, unix.SIGTERM, unix.SIGPIPE)
}
