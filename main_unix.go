//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/git-lfs/git-lfs/v3/git"
)

func setPermissions(path string) os.FileMode {
	var config *git.Configuration
	if strings.HasSuffix(path, ".git") {
		config = git.NewReadOnlyConfig(path, path)
	} else {
		config = git.NewReadOnlyConfig(path, "")
	}
	var umask os.FileMode
	sr := config.Find("core.sharedrepository")
	switch sr {
	case "true", "group":
		umask = 0660
	case "all", "world", "everybody":
		umask = 0664
	case "false", "umask":
	default:
		val, _ := strconv.ParseUint(sr, 8, 32)
		umask = os.FileMode(val)
	}
	return 0777 ^ umask
}
