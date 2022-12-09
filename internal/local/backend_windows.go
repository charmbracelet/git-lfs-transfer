//go:build windows
// +build windows

package local

// UserForFile returns the user that owns the file at the given path.
func (l *localLockBackend) UserForFile(path string) (string, error) {
	return "unknown", nil
}
