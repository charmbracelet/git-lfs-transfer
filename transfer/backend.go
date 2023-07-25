package transfer

import (
	"io"
	"io/fs"
)

const (
	// UploadOperation is an upload operation.
	UploadOperation = "upload"
	// DownloadOperation is a download operation.
	DownloadOperation = "download"
)

// Backend is a Git LFS backend.
type Backend interface {
	Batch(op string, pointers []Pointer, args map[string]string) ([]BatchItem, error)
	StartUpload(oid string, r io.Reader, args map[string]string) (interface{}, error)
	FinishUpload(state interface{}, args map[string]string) error
	Verify(oid string, args map[string]string) (Status, error)
	Download(oid string, args map[string]string) (fs.File, error)
	LockBackend(args map[string]string) LockBackend
}

// Lock is a Git LFS lock.
type Lock interface {
	Unlock() error
	ID() string
	Path() string
	FormattedTimestamp() string
	OwnerName() string
	AsLockSpec(ownerID bool) ([]string, error)
	AsArguments() []string
}

// LockBackend is a Git LFS lock backend.
type LockBackend interface {
	// Create creates a lock for the given path and refname.
	// Refname can be empty.
	Create(path, refname string) (Lock, error)
	Unlock(lock Lock) error
	FromPath(path string) (Lock, error)
	FromID(id string) (Lock, error)
	Range(cursor string, limit int, iter func(Lock) error) (string, error)
}
