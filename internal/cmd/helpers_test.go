package cmd

import (
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/marftn/rido/internal/config"
	"github.com/marftn/rido/internal/fs"
	"github.com/marftn/rido/internal/store"

	"github.com/stretchr/testify/require"
)

const (
	dirMode  = 0o755
	envName  = ".env"
	envBody  = "SECRET=1"
	treeName = "secrets"
)

// TestMain fixes the umask, so the mode checks test rido and not the shell the
// tests were started from.
func TestMain(m *testing.M) {
	previous := syscall.Umask(0o022)
	code := m.Run()
	syscall.Umask(previous)

	os.Exit(code)
}

// sandbox is a store and a working directory, both thrown away with the test.
type sandbox struct {
	cfg  config.Config
	root string
	repo string
}

func newSandbox(t *testing.T) sandbox {
	t.Helper()

	// On macOS t.TempDir sits under /var, which is a symlink to /private/var.
	// Reading a symlink back gives the resolved path, so resolve the root once
	// here and build every expected path from it.
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)

	sb := sandbox{
		cfg:  config.Config{StoreRoot: filepath.Join(root, "store")},
		root: root,
		repo: filepath.Join(root, "repo"),
	}

	require.NoError(t, os.MkdirAll(sb.repo, dirMode))
	t.Chdir(sb.repo)

	return sb
}

func (sb sandbox) write(t *testing.T, name, content string, mode os.FileMode) string {
	t.Helper()

	path := filepath.Join(sb.repo, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), dirMode))
	require.NoError(t, os.WriteFile(path, []byte(content), mode))

	return path
}

// load re-reads the store from disk, like a fresh run of rido would.
func (sb sandbox) load(t *testing.T) *store.Store {
	t.Helper()

	st, err := store.LoadStore(sb.cfg)
	require.NoError(t, err)

	return st
}

func (sb sandbox) only(t *testing.T) *store.StoreItem {
	t.Helper()

	st := sb.load(t)
	require.Len(t, st.Items, 1)

	return &st.Items[0]
}

func (sb sandbox) addEnv(t *testing.T) *store.StoreItem {
	t.Helper()

	sb.write(t, envName, envBody, fs.FileModeReadOnly)
	require.NoError(t, AddCmd(sb.cfg, []string{envName}))

	return sb.only(t)
}

func requireEntryCount(t *testing.T, sb sandbox, want int) {
	t.Helper()

	require.Len(t, sb.load(t).Items, want)
}

func requireStatus(t *testing.T, item *store.StoreItem, want store.Status) {
	t.Helper()

	require.Equal(t, want, item.Status(), "origin %s", item.Meta.Origin)
}

func requireLinked(t *testing.T, item *store.StoreItem) {
	t.Helper()

	target, err := os.Readlink(item.Meta.Origin)
	require.NoError(t, err)
	require.Equal(t, item.PayloadPath(), target)
	requireStatus(t, item, store.StatusLinked)
}

func requireContent(t *testing.T, path, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, want, string(data))
}

func requireMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Lstat(path)
	require.NoError(t, err)
	require.Equal(t, want, info.Mode().Perm(), "mode of %s", path)
}

func requireRegularFile(t *testing.T, path string) {
	t.Helper()

	info, err := os.Lstat(path)
	require.NoError(t, err)
	require.True(t, info.Mode().IsRegular(), "%s is %s", path, info.Mode())
}

// A `.rido-tmp` file is a copy parked while a file is moved. None may survive.
func requireNoTempEntries(t *testing.T, sb sandbox) {
	t.Helper()

	err := filepath.WalkDir(sb.root, func(path string, _ os.DirEntry, err error) error {
		require.NoError(t, err)
		require.False(t, strings.HasSuffix(path, ".rido-tmp"), "leftover %s", path)

		return nil
	})
	require.NoError(t, err)
}

type node struct {
	Kind string
	Mode os.FileMode
	Body string
}

// treeOf lists a tree as a map, so two trees compare in one assertion. The tree
// is opened as a root, so every read stays inside it whatever the symlinks say.
func treeOf(t *testing.T, dir string) map[string]node {
	t.Helper()

	root, err := os.OpenRoot(dir)
	require.NoError(t, err)

	defer root.Close()

	tree := root.FS()
	nodes := map[string]node{}

	err = iofs.WalkDir(tree, ".", func(path string, entry iofs.DirEntry, walkErr error) error {
		require.NoError(t, walkErr)

		info, infoErr := entry.Info()
		require.NoError(t, infoErr)

		n := node{Kind: "file", Mode: info.Mode().Perm(), Body: ""}

		switch {
		case info.IsDir():
			n.Kind = "dir"
		case info.Mode().Type() == os.ModeSymlink:
			target, linkErr := iofs.ReadLink(tree, path)
			require.NoError(t, linkErr)

			// A symlink has no mode of its own worth comparing.
			n = node{Kind: "symlink", Mode: 0, Body: target}
		default:
			data, readErr := iofs.ReadFile(tree, path)
			require.NoError(t, readErr)

			n.Body = string(data)
		}

		nodes[path] = n

		return nil
	})
	require.NoError(t, err)

	return nodes
}

func requireTreeEqual(t *testing.T, want, got string) {
	t.Helper()

	require.Equal(t, treeOf(t, want), treeOf(t, got))
}

// makeTree writes the directory used by the tree tests.
func makeTree(t *testing.T, root string) string {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Join(root, "nested"), dirMode))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "empty"), dirMode))
	require.NoError(t, os.WriteFile(filepath.Join(root, "db.json"), []byte(`{"pw":"x"}`), 0o400))
	require.NoError(t, os.WriteFile(filepath.Join(root, "nested", "token"), []byte("t0k3n"), 0o600))
	require.NoError(t, os.Symlink("../db.json", filepath.Join(root, "nested", "link")))

	return root
}

// Each fixture below checks the status it claims to create. A fixture that quietly
// stops working then fails, instead of making its tests pass for the wrong reason.

func makeMissing(t *testing.T, item *store.StoreItem) {
	t.Helper()

	require.NoError(t, os.Remove(item.Meta.Origin))
	requireStatus(t, item, store.StatusMissing)
}

func makeOccupied(t *testing.T, item *store.StoreItem) {
	t.Helper()

	content := "Some agent wrote this."

	require.NoError(t, os.Remove(item.Meta.Origin))
	require.NoError(t, os.WriteFile(item.Meta.Origin, []byte(content), fs.FileModeShared))
	requireStatus(t, item, store.StatusOccupied)
}

func makeDangling(t *testing.T, item *store.StoreItem) {
	t.Helper()

	require.NoError(t, os.Remove(item.Meta.Origin))
	require.NoError(
		t,
		os.Symlink(filepath.Join(item.Store.Config.StoreRoot, "gone"), item.Meta.Origin),
	)
	requireStatus(t, item, store.StatusOccupied)
}

func makeStale(t *testing.T, item *store.StoreItem) {
	t.Helper()

	require.NoError(t, os.RemoveAll(filepath.Dir(item.Meta.Origin)))
	requireStatus(t, item, store.StatusStale)
}

func makeBroken(t *testing.T, item *store.StoreItem) {
	t.Helper()

	require.NoError(t, os.RemoveAll(item.PayloadPath()))
	requireStatus(t, item, store.StatusBroken)
}
