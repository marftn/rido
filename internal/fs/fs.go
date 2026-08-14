package fs

import (
	"fmt"
	"io"
	"os"
)

const (
	FileModeDefault  os.FileMode = 0o700
	FileModeReadOnly os.FileMode = 0o600
)

func CopyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("could not open %q: %w", src, err)
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return fmt.Errorf("could not stat %q: %w", src, err)
	}

	// Mode is preserved so that `revert` hands the file back as it was found.
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("could not create %q: %w", dst, err)
	}
	defer out.Close()

	if _, e := io.Copy(out, in); e != nil {
		return fmt.Errorf("could not copy %q -> %q: %w", src, dst, e)
	}

	return nil
}

// CopyDir copies a directory tree.
// FIXME: os.CopyFS does not preserve modes and rejects irregular files.
// We need to write a WalkDir copy to preserve original modes.
func CopyDir(dst, src string) error {
	return os.CopyFS(dst, os.DirFS(src))
}

// Exists reports whether the path exists, without following symlinks: a dangling
// symlink occupies its path and is considered present.
func Exists(filename string) bool {
	_, err := os.Lstat(filename)

	return !os.IsNotExist(err)
}
