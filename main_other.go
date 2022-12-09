//go:build !(darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)
// +build !darwin,!dragonfly,!freebsd,!linux,!netbsd,!openbsd,!solaris

package cmd

import (
	"os"
)

func setPermissions(path string) os.FileMode {
	return 0077
}
