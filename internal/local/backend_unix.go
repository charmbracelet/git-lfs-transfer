//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package local

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"syscall"

	transfer "github.com/aymanbagabas/git-lfs-transfer"
)

// FixPermissions fixes the permissions of the file at the given path.
func (l *LocalBackend) FixPermissions(path string) (transfer.Status, error) {
	_, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0777^l.umask); err != nil {
		return nil, err
	}
	return transfer.SuccessStatus(), nil
}

// UserForFile returns the user that owns the file at the given path.
func (l *localLockBackend) UserForFile(path string) (string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	info, ok := stat.Sys().(*syscall.Stat_t)
	if !ok {
		return "", fmt.Errorf("cannot get user for file %q", path)
	}
	user, err := user.LookupId(strconv.Itoa(int(info.Uid)))
	if err != nil {
		return "", err
	}
	return user.Username, nil
}
