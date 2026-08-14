package store

import (
	"os"
	"path/filepath"
	"rido/internal/config"
	"rido/internal/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddThenRelink(t *testing.T) {
	repo := t.TempDir()
	cfg := config.Config{StoreLocation: filepath.Join(t.TempDir(), "store")}

	origin := filepath.Join(repo, ".env")
	require.NoError(t, os.WriteFile(origin, []byte("SECRET=1"), fs.FileModeReadOnly))

	st, err := LoadStore(cfg)
	require.NoError(t, err)

	item := st.NewStoreItem(&Meta{Filename: ".env", Origin: origin})
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

func TestStatusStaleWhenOriginDirIsGone(t *testing.T) {
	cfg := config.Config{StoreLocation: filepath.Join(t.TempDir(), "store")}

	st, err := LoadStore(cfg)
	require.NoError(t, err)

	gone := filepath.Join(t.TempDir(), "oldrepo")
	require.NoError(t, os.Mkdir(gone, 0o700))

	origin := filepath.Join(gone, ".env")
	require.NoError(t, os.WriteFile(origin, []byte("SECRET=2"), 0o600))

	item := st.NewStoreItem(&Meta{Filename: ".env", Origin: origin})
	require.NoError(t, WriteStoreItem(&item))
	require.NoError(t, os.RemoveAll(gone))

	require.Equal(t, StatusStale, item.Status())
}
