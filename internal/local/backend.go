package local

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/charmbracelet/git-lfs-transfer/transfer"
)

var _ transfer.Backend = &LocalBackend{}

// LocalBackend is a local Git LFS backend.
type LocalBackend struct {
	lfsPath   string
	umask     fs.FileMode
	timestamp *time.Time
}

// New creates a new local backend. lfsPath should be a `.git/lfs` directory.
func New(lfsPath string, umask os.FileMode, timestamp *time.Time) *LocalBackend {
	return &LocalBackend{
		lfsPath:   lfsPath,
		umask:     umask,
		timestamp: timestamp,
	}
}

// Batch implements main.Backend
func (l *LocalBackend) Batch(_ transfer.Operation, oids []transfer.OidWithSize) ([]transfer.BatchItem, error) {
	items := make([]transfer.BatchItem, 0)
	for _, o := range oids {
		oid := o.Oid
		size := o.Size
		present := false
		stat, err := oid.Stat(l.lfsPath)
		if err == nil {
			size = stat.Size()
			present = true
		}
		items = append(items, transfer.BatchItem{
			Oid:     oid,
			Size:    size,
			Present: present,
		})
	}
	return items, nil
}

// Download implements main.Backend. The returned reader must be closed by the
// caller.
func (l *LocalBackend) Download(oid transfer.Oid, args ...string) (fs.File, error) {
	path := oid.ExpectedPath(l.lfsPath)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// FinishUpload implements main.Backend.
func (l *LocalBackend) FinishUpload(state interface{}, args ...string) error {
	switch state := state.(type) {
	case *UploadState:
		destPath := state.Oid.ExpectedPath(l.lfsPath)
		parent := filepath.Dir(destPath)
		transfer.Logf("finishing upload of %s at %s", destPath, parent)
		if err := os.MkdirAll(parent, 0777); err != nil {
			return err
		}
		if err := os.Link(state.TempFile.Name(), destPath); err != nil {
			return err
		}
		defer state.TempFile.Close()
		if _, err := l.FixPermissions(destPath); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("invalid state type: %T", state)
	}
}

// LockBackend implements main.Backend.
func (l *LocalBackend) LockBackend() transfer.LockBackend {
	path := filepath.Join(l.lfsPath, "locks")
	return NewLockBackend(l, path)
}

// UploadState is a state for an upload.
type UploadState struct {
	Oid      transfer.Oid
	TempFile *os.File
}

// StartUpload implements main.Backend. The returned temp file should be closed.
func (l *LocalBackend) StartUpload(oid transfer.Oid, r io.Reader, args ...string) (interface{}, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: received null data", transfer.ErrMissingData)
	}
	transfer.Logf("start uploading %s", oid)
	tempDir := filepath.Join(l.lfsPath, "incomplete")
	randBytes := make([]byte, 12)
	if _, err := rand.Read(randBytes); err != nil {
		return nil, err
	}
	tempName := fmt.Sprintf("%s%x", oid, randBytes)
	tempFile := filepath.Join(tempDir, tempName)
	f, err := os.Create(tempFile)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(f, r); err != nil {
		return nil, err
	}
	return &UploadState{
		Oid:      oid,
		TempFile: f,
	}, nil
}

// Verify implements main.Backend.
func (l *LocalBackend) Verify(oid transfer.Oid, args map[string]string) (transfer.Status, error) {
	var expectedSize int
	size, ok := args[transfer.SizeKey]
	if ok {
		expectedSize, _ = strconv.Atoi(size)
	}
	if expectedSize == 0 {
		return nil, fmt.Errorf("missing size argument")
	}
	stat, err := oid.Stat(l.lfsPath)
	if err != nil {
		return nil, err
	}
	if stat.Size() != int64(expectedSize) {
		transfer.Logf("size mismatch, expected %d, got %d", expectedSize, stat.Size())
		return transfer.NewFailureStatus(transfer.StatusConflict, "size mismatch"), nil
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
func (l *localLockBackend) Create(path string) (transfer.Lock, error) {
	id := localBackendLock{}.HashFor(path)
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("%s:%d:", LocalBackendLockVersion, l.Timestamp().Unix()))
	b.WriteString(path)
	fileName := filepath.Join(l.lockPath, id)
	f, err := NewLockFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("error creating local lock file: %w", err)
	}
	defer f.Remove()
	defer f.Close()
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
	defer f.Close()
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
func (l *localLockBackend) Range(f func(l transfer.Lock) error) error {
	data, err := os.ReadDir(l.lockPath)
	if err != nil {
		return err
	}
	transfer.Logf("found %d locks", len(data))
	sort.Slice(data, func(i, j int) bool {
		return data[i].Name() < data[j].Name()
	})
	for _, lf := range data {
		transfer.Logf("found lock %s", lf.Name())
		lock, err := l.FromID(lf.Name())
		if err != nil {
			// TODO: handle error
			continue
		}
		if err := f(lock); err != nil {
			return err
		}
	}
	return nil
}
