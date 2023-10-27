package local

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/charmbracelet/git-lfs-transfer/transfer"
)

// LockFile is a local backend lock file.
type LockFile struct {
	path     string
	temp     string
	tempFile *os.File
}

var _ io.Writer = &LockFile{}
var _ io.Closer = &LockFile{}

// NewLockFile creates a new lock file.
func NewLockFile(path string) (*LockFile, error) {
	temp := path + ".lock"
	lf := &LockFile{
		path: path,
		temp: temp,
	}
	// If the lock file already exists, return an error.
	if _, err := os.Stat(temp); err == nil {
		f, err := os.Open(temp)
		if err != nil {
			return nil, transfer.ErrConflict
		}
		lf.tempFile = f
		return lf, transfer.ErrConflict
	}
	f, err := os.Create(temp)
	if err != nil {
		return nil, err
	}
	lf.tempFile = f
	return lf, nil
}

// Write writes the given data to the lock file.
func (l *LockFile) Write(data []byte) (int, error) {
	return l.tempFile.Write(data)
}

// Close closes the lock file.
func (l *LockFile) Close() error {
	return l.tempFile.Close()
}

// Persist persists the lock file.
func (l *LockFile) Persist() error {
	err := os.Link(l.temp, l.path)
	if errors.Is(err, os.ErrExist) {
		return transfer.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("error persisting lock file: %w", err)
	}
	return nil
}

// Remove removes the lock file.
func (l *LockFile) Remove() error {
	return os.Remove(l.temp)
}

const (
	// LocalBackendLockVersion is the version of the local backend.
	LocalBackendLockVersion = "v1"
)

var _ transfer.Lock = &localBackendLock{}

// localBackendLock is a local backend lock.
type localBackendLock struct {
	root      string
	pathName  string
	time      *time.Time
	ownerName string
}

// NewLocalBackendLock creates a new local backend lock.
func NewLocalBackendLock(root, pathName string, time *time.Time, ownerName string) transfer.Lock {
	return &localBackendLock{
		root:      root,
		pathName:  pathName,
		time:      time,
		ownerName: ownerName,
	}
}

// HashFor returns the hash for the given path.
func (localBackendLock) HashFor(path string) string {
	hash := sha256.New()
	hash.Write([]byte(LocalBackendLockVersion))
	hash.Write([]byte(":"))
	hash.Write([]byte(path))
	return hex.EncodeToString(hash.Sum(nil))
}

// Parse parses the given data.
func (localBackendLock) Parse(data []byte) (*time.Time, []byte, error) {
	v := bytes.SplitN(data, []byte(":"), 3)
	if len(v) != 3 {
		return nil, nil, fmt.Errorf("invalid lock data: %q", data)
	}
	unixTime, err := strconv.Atoi(string(v[1]))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse time: %q", data)
	}
	time := time.Unix(int64(unixTime), 0)
	return &time, v[2], nil
}

// AsArguments implements main.Lock.
func (l *localBackendLock) AsArguments() []string {
	return []string{
		fmt.Sprintf("id=%s", l.ID()),
		fmt.Sprintf("path=%s", l.Path()),
		fmt.Sprintf("locked-at=%s", l.FormattedTimestamp()),
		fmt.Sprintf("ownername=%s", l.OwnerName()),
	}
}

// AsLockSpec implements main.Lock.
func (l *localBackendLock) AsLockSpec(ownerID bool) ([]string, error) {
	id := l.ID()
	msgs := []string{
		fmt.Sprintf("lock %s", id),
		fmt.Sprintf("path %s %s", id, l.Path()),
		fmt.Sprintf("locked-at %s %s", id, l.FormattedTimestamp()),
		fmt.Sprintf("ownername %s %s", id, l.OwnerName()),
	}
	if ownerID {
		user, err := l.CurrentUser()
		if err != nil {
			return nil, fmt.Errorf("error getting current user: %w", err)
		}
		who := "theirs"
		if user == l.OwnerName() {
			who = "ours"
		}
		msgs = append(msgs, fmt.Sprintf("owner %s %s", id, who))
	}
	return msgs, nil
}

// FormattedTimestamp implements main.Lock.
func (l *localBackendLock) FormattedTimestamp() string {
	return l.time.UTC().Format(time.RFC3339)
}

// ID implements main.Lock.
func (l *localBackendLock) ID() string {
	return l.HashFor(l.pathName)
}

// OwnerName implements main.Lock.
func (l *localBackendLock) OwnerName() string {
	return l.ownerName
}

// Path implements main.Lock.
func (l *localBackendLock) Path() string {
	return l.pathName
}

// Unlock implements main.Lock.
func (l *localBackendLock) Unlock() error {
	id := l.HashFor(l.pathName)
	fileName := filepath.Join(l.root, id)
	return os.Remove(fileName)
}
