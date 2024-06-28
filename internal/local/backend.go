package local

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/git-lfs-transfer/transfer"
)

var _ transfer.Backend = &LocalBackend{}

func oidExpectedPath(root, oid string) string {
	p := transfer.Pointer{Oid: oid}
	rp := p.RelativePath()
	rp = strings.ReplaceAll(rp, "/", string(filepath.Separator))
	return filepath.Join(root, "objects", rp)
}

// LocalBackend is a local Git LFS backend.
type LocalBackend struct { // nolint: revive
	timestamp *time.Time
	lfsPath   string
	umask     fs.FileMode
}

// New creates a new local backend. lfsPath should be a `.git/lfs` directory.
func New(lfsPath string, umask os.FileMode, timestamp *time.Time) *LocalBackend {
	return &LocalBackend{
		lfsPath:   lfsPath,
		umask:     umask,
		timestamp: timestamp,
	}
}

// Batch implements main.Backend.
func (l *LocalBackend) Batch(_ string, pointers []transfer.BatchItem, _ transfer.Args) ([]transfer.BatchItem, error) {
	for i := range pointers {
		present := false
		stat, err := os.Stat(oidExpectedPath(l.lfsPath, pointers[i].Oid))
		if err == nil {
			pointers[i].Size = stat.Size()
			present = true
		}
		pointers[i].Present = present
	}
	return pointers, nil
}

// Download implements main.Backend. The returned reader must be closed by the
// caller.
func (l *LocalBackend) Download(oid string, _ transfer.Args) (io.ReadCloser, int64, error) {
	f, err := os.Open(oidExpectedPath(l.lfsPath, oid))
	if err != nil {
		return nil, 0, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	return f, info.Size(), nil
}

// LockBackend implements main.Backend.
func (l *LocalBackend) LockBackend(_ transfer.Args) transfer.LockBackend {
	path := filepath.Join(l.lfsPath, "locks")
	return NewLockBackend(l, path)
}

// Upload implements main.Backend.
func (l *LocalBackend) Upload(oid string, size int64, r io.Reader, _ transfer.Args) error {
	if r == nil {
		return fmt.Errorf("%w: received null data", transfer.ErrMissingData)
	}
	tempDir := filepath.Join(l.lfsPath, "incomplete")
	randBytes := make([]byte, 12)
	if _, err := rand.Read(randBytes); err != nil {
		return err
	}
	tempName := fmt.Sprintf("%s%x", oid, randBytes)
	tempFile := filepath.Join(tempDir, tempName)
	f, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.Remove(tempFile)
	}()
	if _, err := io.Copy(f, r); err != nil {
		return err
	}
	f.Close() // double-close is fine
	destPath := oidExpectedPath(l.lfsPath, oid)
	parent := filepath.Dir(destPath)
	if err := os.MkdirAll(parent, 0777); err != nil {
		return err
	}
	if err := os.Link(tempFile, destPath); err != nil {
		return err
	}
	if _, err := l.FixPermissions(destPath); err != nil {
		return err
	}
	return nil
}

// Verify implements main.Backend.
func (l *LocalBackend) Verify(oid string, size int64, args transfer.Args) (transfer.Status, error) {
	if size == 0 {
		return nil, fmt.Errorf("missing size argument")
	}
	stat, err := os.Stat(oidExpectedPath(l.lfsPath, oid))
	if errors.Is(err, fs.ErrNotExist) {
		return transfer.NewStatus(transfer.StatusNotFound, "not found"), nil
	}
	if err != nil {
		return nil, err
	}
	if stat.Size() != size {
		return transfer.NewStatus(transfer.StatusConflict, "size mismatch"), nil
	}
	return transfer.SuccessStatus(), nil
}

var _ transfer.LockBackend = &localLockBackend{}

type localLockBackend struct {
	backend  transfer.Backend
	lockPath string
}

// NewLockBackend creates a new local lock backend.
func NewLockBackend(backend transfer.Backend, lockPath string) transfer.LockBackend {
	return &localLockBackend{
		backend:  backend,
		lockPath: lockPath,
	}
}

// Timestamp returns the timestamp for the lock backend.
func (l *localLockBackend) Timestamp() *time.Time {
	return l.backend.(*LocalBackend).timestamp
}

// Create implements main.LockBackend.
func (l *localLockBackend) Create(path, _ string) (transfer.Lock, error) {
	id := localBackendLock{}.HashFor(path)
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("%s:%d:", LocalBackendLockVersion, l.Timestamp().Unix()))
	b.WriteString(path)
	fileName := filepath.Join(l.lockPath, id)
	f, err := NewLockFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("error creating local lock file: %w", err)
	}
	defer func() {
		f.Close()  // nolint: errcheck
		f.Remove() // nolint: errcheck
	}()
	if _, err := f.Write(b.Bytes()); err != nil {
		return nil, err
	}
	if err := f.Persist(); err != nil {
		return nil, err
	}
	user, err := l.UserForFile(fileName)
	if err != nil {
		return nil, err
	}
	return NewLocalBackendLock(l.lockPath, path, l.Timestamp(), user), nil
}

// FromID implements main.LockBackend.
func (l *localLockBackend) FromID(id string) (transfer.Lock, error) {
	fileName := filepath.Join(l.lockPath, id)
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("error opening local lock file: %w", err)
	}
	defer f.Close() // nolint: errcheck
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("error reading local lock file: %w", err)
	}
	time, btsPath, err := localBackendLock{}.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("error parsing local lock file: %w", err)
	}
	user, err := l.UserForFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("error getting user for local lock file: %w", err)
	}
	return NewLocalBackendLock(l.lockPath, string(btsPath), time, user), nil
}

// FromPath implements main.LockBackend.
func (l *localLockBackend) FromPath(path string) (transfer.Lock, error) {
	id := localBackendLock{}.HashFor(path)
	lock, err := l.FromID(id)
	if err != nil {
		return nil, err
	}
	if lock.Path() != path {
		return nil, fmt.Errorf("%w: unexpected file name", transfer.ErrCorruptData)
	}
	return lock, nil
}

// Unlock implements main.LockBackend.
func (localLockBackend) Unlock(lock transfer.Lock) error {
	return lock.Unlock()
}

// Range implements main.LockBackend. Iterate over all locks. Returning an error will break and return.
func (l *localLockBackend) Range(_ string, _ int, f func(l transfer.Lock) error) (string, error) {
	var err error
	data, err := os.ReadDir(l.lockPath)
	if err != nil {
		return "", err
	}
	sort.Slice(data, func(i, j int) bool {
		return data[i].Name() < data[j].Name()
	})
	for _, lf := range data {
		var lock transfer.Lock
		lock, err = l.FromID(lf.Name())
		if err != nil {
			err = errors.Join(err, fmt.Errorf("error reading lock %s: %w", lf.Name(), err))
		}
		if err := f(lock); err != nil {
			return "", err
		}
	}
	return "", err
}
