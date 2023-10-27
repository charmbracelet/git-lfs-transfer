//go:build !(darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)
// +build !darwin,!dragonfly,!freebsd,!linux,!netbsd,!openbsd,!solaris

package main

import (
	"os"
)

func setPermissions(path string) os.FileMode {
	return 0077
}

func setup(c chan os.Signal) {}
