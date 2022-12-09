//go:build windows
// +build windows

package local

// CurrentUser returns the current user name.
func (l *localBackendLock) CurrentUser() (string, error) {
	return "unknown", nil
}
