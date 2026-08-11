package guard

import (
	"io/fs"
	"os"
	"rido/internal/log"
)

// IsEmptyOrSymlink returns true if the file does not exist or is a symlink.
// If the function fails to call `lstat`, the error is logged and the function
// silently returns `false`.
func IsEmptyOrSymlink(filename string) bool {
	info, err := os.Lstat(filename)
	if os.IsNotExist(err) {
		return true
	} else if err != nil {
		log.Errorf("failed to lstat file: %v", err)

		return false
	}

	return info.Mode().Type() == fs.ModeSymlink
}
