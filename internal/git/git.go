package git

import (
	"fmt"
	"os"
	"path/filepath"
	"rido/internal/fs"
	"rido/internal/tty"
	"strings"

	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/osfs"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/format/gitignore"
)

// Check refuses a path tracked by git, since hiding it would leave the payload
// in history, and offers to gitignore one that is neither tracked nor ignored.
// A path outside a git repository is left alone.
func Check(origin string) error {
	repo, err := git.PlainOpenWithOptions(
		filepath.Dir(origin),
		&git.PlainOpenOptions{DetectDotGit: true},
	)
	if err != nil {
		//nolint:nilerr // A non-nil err means we're outside a git repo. We can return nil.
		return nil
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("could not read the git worktree: %w", err)
	}

	root := worktree.Filesystem().Root()

	rel, err := filepath.Rel(root, origin)
	if err != nil {
		return fmt.Errorf("could not relate %q to %q: %w", origin, root, err)
	}

	rel = filepath.ToSlash(rel)

	tracked, err := isTracked(repo, rel)
	if err != nil {
		return err
	}

	if tracked {
		return fmt.Errorf("tracked by git, untrack it first: git rm -r --cached %s", rel)
	}

	ignored, err := isIgnored(worktree, rel)
	if err != nil {
		return err
	}

	if ignored {
		return nil
	}

	if !tty.IsTTY() {
		return errNoTTY
	}

	return offerGitignore(root, origin, rel)
}

// isTracked reports whether the index holds the path, or anything under it.
func isTracked(repo *git.Repository, rel string) (bool, error) {
	index, err := repo.Storer.Index()
	if err != nil {
		return false, fmt.Errorf("could not read the git index: %w", err)
	}

	for _, entry := range index.Entries {
		if entry.Name == rel || strings.HasPrefix(entry.Name, rel+"/") {
			return true, nil
		}
	}

	return false, nil
}

func isIgnored(worktree *git.Worktree, rel string) (bool, error) {
	worktreeFs := worktree.Filesystem()

	patterns, err := ignorePatterns(worktreeFs)
	if err != nil {
		return false, err
	}

	info, err := worktreeFs.Lstat(rel)
	if err != nil {
		return false, fmt.Errorf("could not stat %q: %w", rel, err)
	}

	matcher := gitignore.NewMatcher(append(patterns, worktree.Excludes...))

	return matcher.Match(strings.Split(rel, "/"), info.IsDir()), nil
}

// ignorePatterns gathers the system, global and repository patterns, in ascending
// order of priority.
func ignorePatterns(worktreeFs billy.Filesystem) ([]gitignore.Pattern, error) {
	rootFs := osfs.New("/")

	system, err := gitignore.LoadSystemPatterns(rootFs)
	if err != nil {
		return nil, fmt.Errorf("could not read the system gitignore patterns: %w", err)
	}

	global, err := gitignore.LoadGlobalPatterns(rootFs)
	if err != nil {
		return nil, fmt.Errorf("could not read the global gitignore patterns: %w", err)
	}

	repo, err := gitignore.ReadPatterns(worktreeFs, nil)
	if err != nil {
		return nil, fmt.Errorf("could not read the gitignore patterns: %w", err)
	}

	return append(append(system, global...), repo...), nil
}

func offerGitignore(root, origin, rel string) error {
	// Anchored to the repository root, so the entry cannot match a namesake elsewhere.
	entry := "/" + rel

	isYes, err := tty.AskForConfirmation(
		os.Stdin,
		os.Stdout,
		"'%s' is not gitignored. Append '%s' to .gitignore?",
		origin,
		entry,
	)
	if err != nil {
		return fmt.Errorf("failed to get user confirmation: %w", err)
	}

	if !isYes {
		return nil
	}

	return appendLine(filepath.Join(root, ".gitignore"), entry)
}

func appendLine(filename, line string) error {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, fs.FileModeShared)
	if err != nil {
		return fmt.Errorf("could not open %q: %w", filename, err)
	}
	defer file.Close()

	sep, err := missingNewline(file)
	if err != nil {
		return err
	}

	if _, e := fmt.Fprintf(file, "%s%s\n", sep, line); e != nil {
		return fmt.Errorf("could not append to %q: %w", filename, e)
	}

	return file.Close()
}

// missingNewline returns a newline when the file does not already end with one.
func missingNewline(file *os.File) (string, error) {
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("could not stat %q: %w", file.Name(), err)
	}

	if info.Size() == 0 {
		return "", nil
	}

	last := make([]byte, 1)
	if _, e := file.ReadAt(last, info.Size()-1); e != nil {
		return "", fmt.Errorf("could not read %q: %w", file.Name(), e)
	}

	if last[0] == '\n' {
		return "", nil
	}

	return "\n", nil
}
