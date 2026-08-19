package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/marftn/rido/internal/config"
	"github.com/marftn/rido/internal/fs"

	"github.com/stretchr/testify/require"
)

const (
	testEnvFilename = ".env"
)

func TestAddThenRelink(t *testing.T) {
	repo := t.TempDir()
	cfg := config.Config{StoreRoot: filepath.Join(t.TempDir(), "store")}

	origin := filepath.Join(repo, testEnvFilename)
	require.NoError(t, os.WriteFile(origin, []byte("SECRET=1"), fs.FileModeReadOnly))

	st, err := LoadStore(cfg)
	require.NoError(t, err)

	meta := NewMeta(origin)
	item := st.NewStoreItem(&meta)
	require.NoError(t, WriteStoreItem(&item))

	target, err := os.Readlink(origin)
	require.NoError(t, err)
	require.Equal(t, item.PayloadPath(), target)
	require.Equal(t, StatusLinked, item.Status())

	data, err := os.ReadFile(origin)
	require.NoError(t, err)
	require.Equal(t, "SECRET=1", string(data))

	info, err := os.Stat(item.PayloadPath())
	require.NoError(t, err)
	require.Equal(
		t,
		fs.FileModeReadOnly,
		info.Mode().Perm(),
		"payload mode must be preserved",
	)

	entries, err := os.ReadDir(repo)
	require.NoError(t, err)
	require.Len(t, entries, 1, "the parked file must be gone")

	// A removed symlink is MISSING and can be relinked.
	require.NoError(t, os.Remove(origin))
	require.Equal(t, StatusMissing, item.Status())
	require.NoError(t, ReplaceWithSymlink(item.Meta, item.PayloadPath()))
	require.Equal(t, StatusLinked, item.Status())

	// A dangling symlink occupies the origin, relinking must not fail on EEXIST.
	require.NoError(t, os.Remove(origin))
	require.NoError(t, os.Symlink(filepath.Join(repo, "gone"), origin))
	require.Equal(t, StatusOccupied, item.Status())
	require.NoError(t, ReplaceWithSymlink(item.Meta, item.PayloadPath()))
	require.Equal(t, StatusLinked, item.Status())

	// Entries survive a reload, a lost payload is BROKEN.
	reloaded, err := LoadStore(cfg)
	require.NoError(t, err)
	require.Len(t, reloaded.Items, 1)
	require.Equal(t, StatusLinked, reloaded.Items[0].Status())

	require.NoError(t, os.Remove(item.PayloadPath()))
	require.Equal(t, StatusBroken, item.Status())
}

func TestRevert(t *testing.T) {
	repo := t.TempDir()
	cfg := config.Config{StoreRoot: filepath.Join(t.TempDir(), "store")}

	origin := filepath.Join(repo, testEnvFilename)
	require.NoError(t, os.WriteFile(origin, []byte("SECRET=3"), fs.FileModeReadOnly))

	st, err := LoadStore(cfg)
	require.NoError(t, err)

	meta := NewMeta(origin)
	item := st.NewStoreItem(&meta)
	require.NoError(t, WriteStoreItem(&item))
	require.NoError(t, Revert(&item))

	info, err := os.Lstat(origin)
	require.NoError(t, err)
	require.Zero(t, info.Mode().Type(), "the origin must be a regular file again")
	require.Equal(t, fs.FileModeReadOnly, info.Mode().Perm(), "mode must be preserved")

	data, err := os.ReadFile(origin)
	require.NoError(t, err)
	require.Equal(t, "SECRET=3", string(data))

	require.False(t, fs.Exists(item.Path()), "the entry must be dropped")

	entries, err := os.ReadDir(repo)
	require.NoError(t, err)
	require.Len(t, entries, 1, "the parked file must be gone")
}

func TestRevertRecreatesGoneOriginDir(t *testing.T) {
	cfg := config.Config{StoreRoot: filepath.Join(t.TempDir(), "store")}

	st, err := LoadStore(cfg)
	require.NoError(t, err)

	gone := filepath.Join(t.TempDir(), "oldrepo")
	require.NoError(t, os.Mkdir(gone, fs.FileModeDefault))

	origin := filepath.Join(gone, testEnvFilename)
	require.NoError(t, os.WriteFile(origin, []byte("SECRET=4"), fs.FileModeReadOnly))

	meta := NewMeta(origin)
	item := st.NewStoreItem(&meta)
	require.NoError(t, WriteStoreItem(&item))
	require.NoError(t, os.RemoveAll(gone))

	require.Equal(t, StatusStale, item.Status())
	require.NoError(t, Revert(&item))

	data, err := os.ReadFile(origin)
	require.NoError(t, err)
	require.Equal(t, "SECRET=4", string(data))
}

func TestLoadMetaFileRejectsNewerVersion(t *testing.T) {
	filename := filepath.Join(t.TempDir(), MetaFilename)
	meta := NewMeta(filepath.Join(t.TempDir(), testEnvFilename))

	require.NoError(t, WriteMetaFile(filename, &meta))

	loaded, err := LoadMetaFile(filename)
	require.NoError(t, err)
	require.Equal(t, MetaVersion, loaded.Version)

	meta.Version = MetaVersion + 1
	require.NoError(t, WriteMetaFile(filename, &meta))

	_, err = LoadMetaFile(filename)
	require.Error(t, err)
}

func TestStatusStaleWhenOriginDirIsGone(t *testing.T) {
	cfg := config.Config{StoreRoot: filepath.Join(t.TempDir(), "store")}

	st, err := LoadStore(cfg)
	require.NoError(t, err)

	gone := filepath.Join(t.TempDir(), "oldrepo")
	require.NoError(t, os.Mkdir(gone, fs.FileModeDefault))

	origin := filepath.Join(gone, testEnvFilename)
	require.NoError(t, os.WriteFile(origin, []byte("SECRET=2"), 0o600))

	meta := NewMeta(origin)
	item := st.NewStoreItem(&meta)
	require.NoError(t, WriteStoreItem(&item))
	require.NoError(t, os.RemoveAll(gone))

	require.Equal(t, StatusStale, item.Status())
}

// addedEntry puts a file under a fresh store and returns the store and its entry.
// The origin sits in its own directory so tests can move that directory away.
func addedEntry(t *testing.T) (*Store, *StoreItem) {
	t.Helper()

	dir := t.TempDir()
	origin := filepath.Join(dir, "work", testEnvFilename)

	require.NoError(t, os.MkdirAll(filepath.Dir(origin), fs.FileModeDefault))
	require.NoError(t, os.WriteFile(origin, []byte("SECRET=5"), fs.FileModeReadOnly))

	st, err := LoadStore(config.Config{StoreRoot: filepath.Join(dir, "store")})
	require.NoError(t, err)

	meta := NewMeta(origin)
	item := st.NewStoreItem(&meta)
	require.NoError(t, WriteStoreItem(&item))

	return st, &item
}

func TestResolveByID(t *testing.T) {
	st, item := addedEntry(t)

	found, err := st.FindStoreItem(item.ID.String())
	require.NoError(t, err)
	require.Equal(t, item.ID, found.ID)
}

func TestResolveByOrigin(t *testing.T) {
	st, item := addedEntry(t)

	found, err := st.FindStoreItem(item.Meta.Origin)
	require.NoError(t, err)
	require.Equal(t, item.ID, found.ID)
}

func TestResolveWithoutMatch(t *testing.T) {
	st, _ := addedEntry(t)

	_, err := st.FindStoreItem(filepath.Join(t.TempDir(), "nope"))
	require.ErrorIs(t, err, ErrNotFound)
}

// TestResolveMovedLink resolves a symlink that no longer sits at its origin. The
// entry is found through the link and the gone origin is corrected.
func TestResolveMovedLink(t *testing.T) {
	st, item := addedEntry(t)

	moved := filepath.Join(t.TempDir(), "moved.env")
	require.NoError(t, os.Rename(item.Meta.Origin, moved))

	found, err := st.FindStoreItem(moved)
	require.NoError(t, err)
	require.Equal(t, item.ID, found.ID)
	require.Equal(t, moved, found.Meta.Origin, "a gone origin must be corrected")
}

// TestRevertAfterOriginMoved is the spec's re-point recipe: the directory moved,
// so the symlink resolves the entry while meta.json's origin is gone. The payload
// must be restored where the symlink was, not at the old origin.
func TestRevertAfterOriginMoved(t *testing.T) {
	st, item := addedEntry(t)

	oldDir := filepath.Dir(item.Meta.Origin)
	newDir := filepath.Join(filepath.Dir(oldDir), "moved")
	require.NoError(t, os.Rename(oldDir, newDir))

	moved := filepath.Join(newDir, testEnvFilename)

	found, err := st.FindStoreItem(moved)
	require.NoError(t, err)
	require.Equal(t, StatusLinked, found.Status())
	require.NoError(t, Revert(found))

	info, err := os.Lstat(moved)
	require.NoError(t, err)
	require.Zero(t, info.Mode().Type(), "the payload must replace the symlink")

	require.NoDirExists(t, oldDir, "the vanished origin directory must not be recreated")
}

func TestResolveCopiedLink(t *testing.T) {
	st, item := addedEntry(t)

	copied := filepath.Join(t.TempDir(), "copy.env")
	require.NoError(t, os.Symlink(item.PayloadPath(), copied))

	_, err := st.FindStoreItem(copied)
	require.ErrorIs(t, err, ErrNotOrigin)
	require.ErrorContains(t, err, item.Meta.Origin, "the error must name the real origin")
	require.True(t, fs.Exists(item.Meta.Origin), "the origin must be left alone")
}

func TestVerifyCopyRejectsMismatch(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")

	require.NoError(t, os.WriteFile(src, []byte("SECRET=1"), fs.FileModeReadOnly))
	require.NoError(t, os.WriteFile(dst, []byte("SECRET=truncated"), fs.FileModeReadOnly))

	require.ErrorIs(t, verifyCopy(dst, src), ErrChecksumMismatch)

	require.NoError(t, os.WriteFile(dst, []byte("SECRET=1"), fs.FileModeReadOnly))
	require.NoError(t, verifyCopy(dst, src))
}
