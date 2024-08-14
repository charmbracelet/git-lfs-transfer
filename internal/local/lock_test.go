package local_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/charmbracelet/git-lfs-transfer/internal/local"
	"github.com/charmbracelet/git-lfs-transfer/transfer"
)

func TestTakeLock(t *testing.T) {
	tb := testing.TB(t)
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "test_subject")
	testfile, err := local.NewLockFile(path)
	defer func() { _ = testfile.Close() }()
	assert.Equal(t, err, nil)
	assert.NotEqual(t, testfile, nil)
}

func TestMissLock(t *testing.T) {
	tb := testing.TB(t)
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "test_subject")
	testfile, err := local.NewLockFile(path)
	defer testfile.Close()
	assert.Equal(t, err, nil)
	assert.NotEqual(t, testfile, nil)
	t2, e2 := local.NewLockFile(path)
	if t2 != nil {
		defer t2.Close()
	}
	assert.Equal(t, e2, transfer.ErrConflict)
	assert.NotEqual(t, t2, nil)
}
