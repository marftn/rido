package fs

import (
	"fmt"
	"io"
	"os"
)

const (
	DefaultPermissions os.FileMode = 0700
)

func CopyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("could not open %q: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("could not create %q: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("could not copy %q -> %q: %w", src, dst, err)
	}

	return nil
}
