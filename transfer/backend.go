package transfer

import "io"

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

// Backend is a Git LFS backend.
type Backend interface {
	Batch(op Operation, oids []struct {
		Oid
		int64
	}) ([]*BatchItem, error)
	StartUpload(oid Oid, r io.Reader, args ...string) (interface{}, error)
	FinishUpload(state interface{}, args ...string) error
	Verify(oid Oid, args ...string) (Status, error)
	Download(oid Oid, args ...string) (*struct {
		io.Reader
		int64
	}, error)
	LockBackend() LockBackend
}

// Lock is a Git LFS lock.
type Lock interface {
	Unlock() error
	ID() string
	Path() string
	FormattedTimestamp() string
	OwnerName() string
	AsLockSpec(ownerID bool) (string, error)
	AsArguments() string
}

// Iterator is an iterator interface.
type Iterator[T any] interface {
	Next() bool
	Value() T
	Err() error
}

// LockBackend is a Git LFS lock backend.
type LockBackend interface {
	Iterator[Lock]
	Create(path string) (Lock, error)
	Unlock(lock Lock) error
	FromPath(path string) (Lock, error)
	FromID(id string) (Lock, error)
}
