package transfer

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

var (
	// ErrInvalidOid is returned when an invalid Oid is encountered.
	ErrInvalidOid = errors.New("invalid oid")
)

// OidWithSize is a Git LFS object ID with size.
type OidWithSize struct {
	Oid  Oid
	Size int64
}

// Oid is a Git LFS object ID.
type Oid string

var oid Oid = Oid("")

// NewOid creates a new Oid from bytes.
func NewOid(b []byte) (Oid, error) {
	if oid.Valid(b) {
		return Oid(b), nil
	}
	return Oid(""), ErrInvalidOid
}

// Valid returns true if the Oid is valid.
func (Oid) Valid(b []byte) bool {
	if len(b) != 64 {
		return false
	}
	for _, c := range b {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// String returns the string representation of the Oid.
func (o Oid) String() string {
	return string(o)
}

// Value returns the bytes representation of the Oid.
func (o Oid) Value() []byte {
	return []byte(o)
}

// ExpectedPath returns the expected path of the Oid. The path argument should
// be a `.git/lfs` directory.
func (o Oid) ExpectedPath(path string) string {
	b := o.Value()
	return filepath.Join(path, "objects", string(b[0:2]), string(b[2:4]), string(b))
}

// Stat returns the file info of the Oid. The path argument should be a `.git/lfs`
// directory.
func (o Oid) Stat(path string) (fs.FileInfo, error) {
	stat, err := os.Stat(o.ExpectedPath(path))
	if err != nil {
		return nil, err
	}
	return stat, nil
}
