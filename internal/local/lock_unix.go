//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package local

import (
	"fmt"
	"os/user"
	"strconv"
	"syscall"
)

// CurrentUser returns the current user name.
func (l *localBackendLock) CurrentUser() (string, error) {
	uid := syscall.Getuid()
	user, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		return fmt.Sprintf("uid %d", uid), nil
	}
	return user.Username, nil
}
