package transfer

import (
	"io"
)

// Operation is a Git LFS operation.
type Operation int

const (
	// UploadOperation is an upload operation.
	UploadOperation Operation = iota
	// DownloadOperation is a download operation.
	DownloadOperation
)

// String returns the string representation of the Operation.
func (o Operation) String() string {
	switch o {
	case UploadOperation:
		return "upload"
	case DownloadOperation:
		return "download"
	default:
		return "unknown"
	}
}

// File is a Git LFS file.
type File struct {
	io.Reader
	Size int64
}

// Backend is a Git LFS backend.
type Backend interface {
	Batch(op Operation, oids []OidWithSize) ([]BatchItem, error)
	StartUpload(oid Oid, r io.Reader, args ...string) (interface{}, error)
	FinishUpload(state interface{}, args ...string) error
	Verify(oid Oid, args map[string]string) (Status, error)
	Download(oid Oid, args ...string) (*File, error)
	LockBackend() LockBackend
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
	Create(path string) (Lock, error)
	Unlock(lock Lock) error
	FromPath(path string) (Lock, error)
	FromID(id string) (Lock, error)
	Range(func(Lock) error) error
}
