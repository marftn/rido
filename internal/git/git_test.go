package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/marftn/rido/internal/fs"

	"github.com/go-git/go-git/v6"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), fs.FileModeReadOnly))

	return path
}

func initRepo(t *testing.T) string {
	t.Helper()

	repo := t.TempDir()
	_, err := git.PlainInit(repo, false)
	require.NoError(t, err)

	return repo
}

func trackFile(t *testing.T, repo, name string) {
	t.Helper()

	repository, err := git.PlainOpen(repo)
	require.NoError(t, err)

	worktree, err := repository.Worktree()
	require.NoError(t, err)

	_, err = worktree.Add(name)
	require.NoError(t, err)
}

func requireContent(t *testing.T, filename, want string) {
	t.Helper()

	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	require.Equal(t, want, string(data))
}

func TestCheckPassesOutsideARepository(t *testing.T) {
	require.NoError(t, Check(writeFile(t, t.TempDir(), ".env", "SECRET=1")))
}

func TestCheckRefusesATrackedFile(t *testing.T) {
	repo := initRepo(t)
	origin := writeFile(t, repo, ".env", "SECRET=1")

	trackFile(t, repo, ".env")

	require.ErrorContains(t, Check(origin), "git rm -r --cached .env")
}

func TestCheckPassesAnIgnoredFile(t *testing.T) {
	repo := initRepo(t)
	ignore := writeFile(t, repo, ".gitignore", "/creds.json")

	require.NoError(t, Check(writeFile(t, repo, "creds.json", "{}")))

	requireContent(t, ignore, "/creds.json")
}

// Stdin is closed under `go test`, so Check returns an error if the file is
// not gitignored.
func TestCheckPassesWhenTheGitignoreOfferIsDeclined(t *testing.T) {
	repo := initRepo(t)
	ignore := writeFile(t, repo, ".gitignore", "/creds.json")

	require.ErrorIs(t, Check(writeFile(t, repo, "other.json", "{}")), errNoTTY)

	requireContent(t, ignore, "/creds.json")
}

func TestAppendLineTerminatesTheExistingContent(t *testing.T) {
	ignore := writeFile(t, t.TempDir(), ".gitignore", "/creds.json")

	require.NoError(t, appendLine(ignore, "/other.json"))
	require.NoError(t, appendLine(ignore, "/third.json"))

	requireContent(t, ignore, "/creds.json\n/other.json\n/third.json\n")
}

func TestAppendLineCreatesTheFile(t *testing.T) {
	ignore := filepath.Join(t.TempDir(), ".gitignore")

	require.NoError(t, appendLine(ignore, "/creds.json"))

	requireContent(t, ignore, "/creds.json\n")
}
