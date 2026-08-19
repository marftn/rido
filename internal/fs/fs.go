package fs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"github.com/marftn/rido/internal/log"
	"time"
)

const (
	FileModeDefault  os.FileMode = 0o700
	FileModeReadOnly os.FileMode = 0o600
	FileModeShared   os.FileMode = 0o644
)

const day = 24 * time.Hour

func CopyFile(dst, src string) (err error) {
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
	defer func() {
		err = errors.Join(err, out.Close())
	}()

	if e := out.Chmod(info.Mode().Perm()); e != nil {
		return fmt.Errorf("could not chmod %q: %w", dst, e)
	}

	if _, e := io.Copy(out, in); e != nil {
		return fmt.Errorf("could not copy %q -> %q: %w", src, dst, e)
	}

	return nil
}

// CopyDir copies a directory tree, preserving modes and symlinks, so that
// `revert` hands the tree back as it was found.
// NOTE: irregular files (sockets, devices, pipes) are skipped rather than
// recreated.
func CopyDir(dst, src string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("could not stat %q: %w", path, err)
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		switch {
		case info.IsDir():
			if e := os.MkdirAll(target, info.Mode().Perm()); e != nil {
				return fmt.Errorf("could not create %q: %w", target, e)
			}

			return os.Chmod(target, info.Mode().Perm())
		case info.Mode().Type() == os.ModeSymlink:
			link, e := os.Readlink(path)
			if e != nil {
				return fmt.Errorf("could not read link %q: %w", path, e)
			}

			return os.Symlink(link, target)
		case info.Mode().IsRegular():
			return CopyFile(target, path)
		default:
			log.Warnf("Irregular file '%s' was skipped.", entry.Name())
			return nil
		}
	})
}

// Exists reports whether the path exists, without following symlinks: a dangling
// symlink occupies its path and is considered present.
func Exists(filename string) bool {
	_, err := os.Lstat(filename)

	return !os.IsNotExist(err)
}

// Describe reports what sits at a path, e.g. "regular file, 412 B".
func Describe(filename string) string {
	info, err := os.Lstat(filename)
	if err != nil {
		return "unknown"
	}

	switch mode := info.Mode(); {
	case mode.IsDir():
		return "directory"
	case mode.Type() == os.ModeSymlink:
		if _, e := os.Stat(filename); e != nil {
			return "dangling symlink"
		}

		return "symlink"
	case mode.IsRegular():
		return "regular file, " + humanSize(info.Size())
	default:
		return mode.Type().String()
	}
}

// ModifiedAgo reports how long ago a path was last modified, e.g. "12m ago".
func ModifiedAgo(filename string) string {
	info, err := os.Lstat(filename)
	if err != nil {
		return "unknown"
	}

	since := time.Since(info.ModTime())

	switch {
	case since < time.Minute:
		return fmt.Sprintf("%ds ago", int(since.Seconds()))
	case since < time.Hour:
		return fmt.Sprintf("%dm ago", int(since.Minutes()))
	case since < day:
		return fmt.Sprintf("%dh ago", int(since.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(since/day))
	}
}

// humanSize formats a byte count, e.g. "412 B", "2.1 KB".
func humanSize(size int64) string {
	const unit = 1024

	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGT"[exp])
}
