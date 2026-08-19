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

// TestResolution covers the three resolution paths: bare ID, readlink with a
// stale origin, and the origin scan.
func TestResolution(t *testing.T) {
	dir := t.TempDir()
	origin := filepath.Join(dir, "work", ".env")

	if err := os.MkdirAll(filepath.Dir(origin), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(origin, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	st, err := LoadStore(config.Config{StoreRoot: filepath.Join(dir, "store")})
	if err != nil {
		t.Fatal(err)
	}

	meta := NewMeta(origin)

	item := st.NewStoreItem(&meta)
	if err := WriteStoreItem(&item); err != nil {
		t.Fatal(err)
	}

	if got, e := st.FindStoreItem(item.ID.String()); e != nil || got.ID != item.ID {
		t.Fatalf("bare ID: %v %v", got, e)
	}

	if got, e := st.FindStoreItem(origin); e != nil || got.ID != item.ID {
		t.Fatalf("origin scan: %v %v", got, e)
	}

	// Move the symlink: the origin is now stale, so only readlink can resolve it.
	moved := filepath.Join(dir, "moved.env")
	if err := os.Rename(origin, moved); err != nil {
		t.Fatal(err)
	}

	if got, e := st.FindStoreItem(moved); e != nil || got.ID != item.ID {
		t.Fatalf("readlink: %v %v", got, e)
	}

	if _, e := st.FindStoreItem(filepath.Join(dir, "nope")); e == nil {
		t.Fatal("expected ErrNotFound")
	}
}
