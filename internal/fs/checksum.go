package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Checksum hashes a file or a whole tree by feeding every relative path and its
// content into one digest. Symlinks are hashed by their target rather than followed, and
// irregular files are left out because they are not copied (at least for now).
func Checksum(root string) (string, error) {
	digest := sha256.New()

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// The paths are hashed to make sure a tree preserves its structure. A folder that contains
		// the right bytes but the wrong filenames or folders must fail the checksum test.
		switch {
		case entry.IsDir():
			_, _ = digest.Write([]byte(rel + "\n"))

			return nil
		case entry.Type() == os.ModeSymlink:
			link, e := os.Readlink(path)
			if e != nil {
				return fmt.Errorf("could not read link %q: %w", path, e)
			}

			_, _ = digest.Write([]byte(rel + " -> " + link + "\n"))

			return nil
		case entry.Type().IsRegular():
			_, _ = digest.Write([]byte(rel + "\n"))

			return hashFile(digest, path)
		default:
			return nil
		}
	})
	if err != nil {
		return "", fmt.Errorf("could not checksum %q: %w", root, err)
	}

	return hex.EncodeToString(digest.Sum(nil)), nil
}

func hashFile(digest io.Writer, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open %q: %w", path, err)
	}
	defer file.Close()

	if _, e := io.Copy(digest, file); e != nil {
		return fmt.Errorf("could not read %q: %w", path, e)
	}

	return nil
}
