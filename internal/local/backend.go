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
		stat, err := oid.Stat(l.lfsPath)
		if err == nil {
			size = stat.Size()
		}
		items = append(items, transfer.BatchItem{
			Oid:     oid,
			Size:    size,
			Present: size > 0,
		})
	}
	return items, nil
}

// Download implements main.Backend. The returned reader must be closed by the
// caller.
func (l *LocalBackend) Download(oid transfer.Oid, args ...string) (*transfer.File, error) {
	path := oid.ExpectedPath(l.lfsPath)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	return &transfer.File{Reader: f, Size: info.Size()}, nil
}

// FinishUpload implements main.Backend.
func (l *LocalBackend) FinishUpload(state interface{}, args ...string) error {
	switch state := state.(type) {
	case *UploadState:
		destPath := state.Oid.ExpectedPath(l.lfsPath)
		parent := filepath.Dir(destPath)
		if err := os.MkdirAll(parent, 0777^l.umask); err != nil {
			return err
		}
		_, err := l.FixPermissions(state.TempFile)
		if err != nil {
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
	TempFile string
}

// StartUpload implements main.Backend.
func (l *LocalBackend) StartUpload(oid transfer.Oid, r io.Reader, args ...string) (interface{}, error) {
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
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return nil, err
	}
	return &UploadState{
		Oid:      oid,
		TempFile: tempFile,
	}, nil
}

// Verify implements main.Backend.
func (l *LocalBackend) Verify(oid transfer.Oid, args ...string) (transfer.Status, error) {
	var expectedSize int
	for i := 0; i < len(args); i += 2 {
		if args[i] == "size" {
			expectedSize, _ = strconv.Atoi(args[i+1])
			break
		}
	}
	if expectedSize == 0 {
		return nil, fmt.Errorf("missing size argument")
	}
	stat, err := oid.Stat(l.lfsPath)
	if err != nil {
		return nil, err
	}
	if stat.Size() != int64(expectedSize) {
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
func NewLockBackend(backend transfer.Backend, lockPath string) *localLockBackend {
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
		return nil, fmt.Errorf("error creating local lock file: %v", err)
	}
	defer f.Close()
	f.Write(b.Bytes())
	f.Persist()
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
		return nil, fmt.Errorf("error opening local lock file: %v", err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("error reading local lock file: %v", err)
	}
	time, btsPath, err := localBackendLock{}.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("error parsing local lock file: %v", err)
	}
	user, err := l.UserForFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("error getting user for local lock file: %v", err)
	}
	return NewLocalBackendLock(l.lockPath, string(btsPath), time, user), nil
}

// FromPath implements main.LockBackend.
func (l *localLockBackend) FromPath(path string) (transfer.Lock, error) {
	id := localBackendLock{}.HashFor(path)
	return l.FromID(id)
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
	sort.Slice(data, func(i, j int) bool {
		return data[i].Name() < data[j].Name()
	})
	for _, lf := range data {
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
