package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHumanSize(t *testing.T) {
	for size, want := range map[int64]string{
		0:       "0 B",
		412:     "412 B",
		1023:    "1023 B",
		1024:    "1.0 KB",
		2202009: "2.1 MB",
	} {
		require.Equal(t, want, humanSize(size))
	}
}

func TestDescribe(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(file, make([]byte, 412), FileModeReadOnly))

	link := filepath.Join(dir, "link")
	require.NoError(t, os.Symlink(file, link))

	dangling := filepath.Join(dir, "dangling")
	require.NoError(t, os.Symlink(filepath.Join(dir, "gone"), dangling))

	require.Equal(t, "regular file, 412 B", Describe(file))
	require.Equal(t, "directory", Describe(dir))
	require.Equal(t, "symlink", Describe(link))
	require.Equal(t, "dangling symlink", Describe(dangling))
	require.Equal(t, "unknown", Describe(filepath.Join(dir, "gone")))

	require.Equal(t, "0s ago", ModifiedAgo(file))
	require.Equal(t, "unknown", ModifiedAgo(filepath.Join(dir, "gone")))
}

func TestCopyDir(t *testing.T) {
	src, dst := t.TempDir(), filepath.Join(t.TempDir(), "copy")

	var fmRWX os.FileMode = 0o777
	var fmRW os.FileMode = 0o666

	require.NoError(t, os.MkdirAll(filepath.Join(src, "sub"), fmRWX))
	require.NoError(t, os.Chmod(filepath.Join(src, "sub"), fmRWX))
	require.NoError(
		t,
		os.WriteFile(filepath.Join(src, "sub", "secret"), []byte("x"), FileModeReadOnly),
	)
	require.NoError(t, os.Chmod(filepath.Join(src, "sub", "secret"), fmRW))
	require.NoError(t, os.Symlink("secret", filepath.Join(src, "sub", "link")))

	require.NoError(t, CopyDir(dst, src))

	sub, err := os.Lstat(filepath.Join(dst, "sub"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(fmRWX), sub.Mode().Perm())

	file, err := os.Lstat(filepath.Join(dst, "sub", "secret"))
	require.NoError(t, err)
	require.Equal(t, fmRW, file.Mode().Perm())

	require.Equal(t, "symlink", Describe(filepath.Join(dst, "sub", "link")))
}
