package fs

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChecksumMatchesCopy(t *testing.T) {
	src := filepath.Join(t.TempDir(), "secrets")
	require.NoError(t, os.MkdirAll(filepath.Join(src, "prod"), FileModeDefault))
	require.NoError(t, os.Mkdir(filepath.Join(src, "empty"), FileModeDefault))
	require.NoError(
		t,
		os.WriteFile(filepath.Join(src, "prod", "api.key"), []byte("k"), FileModeReadOnly),
	)
	require.NoError(t, os.Symlink("prod/api.key", filepath.Join(src, "current.key")))
	require.NoError(t, syscall.Mkfifo(filepath.Join(src, "pipe"), uint32(FileModeReadOnly)))

	before, err := Checksum(src)
	require.NoError(t, err)

	dst := filepath.Join(t.TempDir(), "copy")
	require.NoError(t, CopyDir(dst, src))

	after, err := Checksum(dst)
	require.NoError(t, err)
	require.Equal(t, before, after, "an unchanged copy must hash the same")
}

func TestChecksumDetectsChanges(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "secret")
	require.NoError(t, os.WriteFile(file, []byte("SECRET=1"), FileModeReadOnly))

	sum, err := Checksum(file)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(file, []byte("SECRET=2"), FileModeReadOnly))
	changed, err := Checksum(file)
	require.NoError(t, err)
	require.NotEqual(t, sum, changed, "different content must hash differently")

	// Same bytes, wrong place.
	tree := filepath.Join(t.TempDir(), "tree")
	require.NoError(t, os.MkdirAll(filepath.Join(tree, "a"), FileModeDefault))
	require.NoError(t, os.WriteFile(filepath.Join(tree, "a", "k"), []byte("k"), FileModeReadOnly))

	moved := filepath.Join(t.TempDir(), "tree")
	require.NoError(t, os.MkdirAll(filepath.Join(moved, "b"), FileModeDefault))
	require.NoError(t, os.WriteFile(filepath.Join(moved, "b", "k"), []byte("k"), FileModeReadOnly))

	treeSum, err := Checksum(tree)
	require.NoError(t, err)
	movedSum, err := Checksum(moved)
	require.NoError(t, err)
	require.NotEqual(t, treeSum, movedSum, "a moved file must hash differently")
}
