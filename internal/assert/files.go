package assert

import (
	"fmt"
	"os"
	"path/filepath"
)

func AssertFilesExist(files []string) error {
	for _, file := range files {
		if err := AssertFileExists(file); err != nil {
			return err
		}
	}

	return nil
}

func AssertFilesDoNotExist(files []string) error {
	for _, file := range files {
		_, err := os.Stat(file)

		if err == nil {
			return fmt.Errorf("file already exists: %s", file)
		}

		if !os.IsNotExist(err) {
			return fmt.Errorf("checking file %s: %w", file, err)
		}
	}

	return nil
}

func AssertFileExists(file string) error {
	_, err := os.Stat(file)

	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", file)
		}

		return fmt.Errorf("checking file %s: %w", file, err)
	}

	return nil
}

func AssertNoDuplicate(items []string) error {
	seen := make(map[string]struct{})

	for _, item := range items {
		if _, exists := seen[item]; exists {
			return fmt.Errorf("item '%s' is present more than once", item)
		}

		seen[item] = struct{}{}
	}

	return nil
}

func AssertNoNestedPath(files []string) error {
	normalized := make([]string, len(files))

	for i, file := range files {
		path, err := filepath.Abs(file)
		if err != nil {
			return fmt.Errorf("resolving path %s: %w", file, err)
		}

		normalized[i] = filepath.Clean(path)
	}

	for i, parent := range normalized {
		for j, child := range normalized {
			if i == j {
				continue
			}

			rel, err := filepath.Rel(parent, child)
			if err != nil {
				continue
			}

			// child is inside parent
			if rel != "." && rel != ".." && !startsWithParent(rel) {
				return fmt.Errorf("cannot add both %s and %s: nested paths", files[i], files[j])
			}
		}
	}

	return nil
}

func startsWithParent(rel string) bool {
	return len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator)
}
