package assert

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func FilesExist(files []string) error {
	for _, file := range files {
		if err := FileExists(file); err != nil {
			return err
		}
	}

	return nil
}

func FileExists(file string) error {
	_, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", file)
		}

		return fmt.Errorf("checking file %s: %w", file, err)
	}

	return nil
}

func NoDuplicate(items []string) error {
	seen := make(map[string]struct{}, len(items))

	for _, item := range items {
		if _, exists := seen[item]; exists {
			return fmt.Errorf("item '%s' is present more than once", item)
		}

		seen[item] = struct{}{}
	}

	return nil
}

// NoNestedPath rejects a set of paths where one contains another, or where two
// spellings resolve to the same path.
func NoNestedPath(files []string) error {
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

			if isEquivalent(parent, child) {
				return fmt.Errorf("cannot add both '%s' and '%s': same path", files[i], files[j])
			}

			if isInsideParent(parent, child) {
				return fmt.Errorf("cannot add both '%s' and '%s': nested paths", files[i], files[j])
			}
		}
	}

	return nil
}

func isEquivalent(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}

	return rel == "."
}

func isInsideParent(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}

	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
