//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package main

import (
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/git-lfs/git-lfs/v3/git"
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
	var umask int = 0777
	if val != 0 {
		umask ^= val
	}
	return os.FileMode(syscall.Umask(umask))
}
