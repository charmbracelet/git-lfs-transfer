//go:build !(darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)
// +build !darwin,!dragonfly,!freebsd,!linux,!netbsd,!openbsd,!solaris

package local

import "github.com/charmbracelet/git-lfs-transfer/transfer"

// FixPermissions fixes the permissions of the file at the given path.
func (l *LocalBackend) FixPermissions(path string) (transfer.Status, error) {
	return transfer.SuccessStatus(), nil
}
