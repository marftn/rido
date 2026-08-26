package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/marftn/rido/internal/fs"
	"github.com/marftn/rido/internal/store"

	"github.com/stretchr/testify/require"
)

// addOutside adds a file that lives next to the working directory, not in it.
func addOutside(t *testing.T, sb sandbox) string {
	t.Helper()

	origin := filepath.Join(sb.root, "outside", envName)
	require.NoError(t, os.MkdirAll(filepath.Dir(origin), dirMode))
	require.NoError(t, os.WriteFile(origin, []byte("OUTSIDE=1"), fs.FileModeReadOnly))
	require.NoError(t, AddCmd(sb.cfg, []string{origin}))

	return origin
}

func TestAllLeavesEntriesOutsideTheWorkingDirectoryAlone(t *testing.T) {
	sb := newSandbox(t)
	inside := sb.addEnv(t)
	outside := addOutside(t, sb)

	st := sb.load(t)
	require.Len(t, st.Items, 2)

	for i := range st.Items {
		makeMissing(t, &st.Items[i])
	}

	require.NoError(t, RestoreCmd(sb.cfg, []string{"--all"}))

	st = sb.load(t)
	for i := range st.Items {
		item := &st.Items[i]
		if item.Meta.Origin == inside.Meta.Origin {
			requireStatus(t, item, store.StatusLinked)
		} else {
			require.Equal(t, outside, item.Meta.Origin)
			requireStatus(t, item, store.StatusMissing)
		}
	}
}

func TestStoreWideCoversEveryEntry(t *testing.T) {
	sb := newSandbox(t)
	sb.addEnv(t)
	addOutside(t, sb)

	st := sb.load(t)
	for i := range st.Items {
		makeMissing(t, &st.Items[i])
	}

	require.NoError(t, RestoreCmd(sb.cfg, []string{"--store-wide"}))

	st = sb.load(t)
	require.Len(t, st.Items, 2)

	for i := range st.Items {
		requireStatus(t, &st.Items[i], store.StatusLinked)
	}
}

// While the symlink is there, rido finds the entry through it. Once it is gone,
// rido finds it by reading the meta files.
func TestResolutionWorksWithAndWithoutTheSymlink(t *testing.T) {
	sb := newSandbox(t)
	item := sb.addEnv(t)

	require.NoError(t, restoreOne(sb.load(t), envName, false))
	requireLinked(t, item)

	makeMissing(t, item)

	require.NoError(t, restoreOne(sb.load(t), envName, false))
	requireLinked(t, item)
	requireEntryCount(t, sb, 1)
}

func TestASecondSymlinkToAnEntryIsRefused(t *testing.T) {
	sb := newSandbox(t)
	item := sb.addEnv(t)

	second := filepath.Join(sb.repo, "copy.env")
	require.NoError(t, os.Symlink(item.PayloadPath(), second))

	require.ErrorIs(t, restoreOne(sb.load(t), "copy.env", false), store.ErrNotOrigin)
	require.ErrorIs(t, RestoreCmd(sb.cfg, []string{"copy.env"}), ErrReported)

	requireLinked(t, sb.only(t))
}

// A symlink that was moved becomes the origin, whatever meta.json still says.
func TestAMovedSymlinkHealsAStaleOrigin(t *testing.T) {
	sb := newSandbox(t)
	item := sb.addEnv(t)

	moved := filepath.Join(sb.repo, "moved.env")
	require.NoError(t, os.Rename(item.Meta.Origin, moved))

	require.NoError(t, RevertCmd(sb.cfg, []string{"moved.env"}))

	requireRegularFile(t, moved)
	requireContent(t, moved, envBody)
	require.False(t, fs.Exists(item.Meta.Origin))
	requireEntryCount(t, sb, 0)
}

func TestEverySpellingOfAnOriginResolvesToOneEntry(t *testing.T) {
	sb := newSandbox(t)
	item := sb.addEnv(t)
	makeTree(t, filepath.Join(sb.repo, treeName))
	require.NoError(t, AddCmd(sb.cfg, []string{treeName}))

	tests := []struct {
		name     string
		spelling string
		wantID   string
	}{
		{name: "relative", spelling: envName, wantID: item.ID.String()},
		{name: "dot slash", spelling: "./" + envName, wantID: item.ID.String()},
		{name: "absolute", spelling: item.Meta.Origin, wantID: item.ID.String()},
		{name: "id", spelling: item.ID.String(), wantID: item.ID.String()},
	}

	st := sb.load(t)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			found, err := st.FindStoreItem(tc.spelling)
			require.NoError(t, err)
			require.Equal(t, tc.wantID, found.ID.String())
		})
	}

	t.Run("trailing slash on a directory", func(t *testing.T) {
		found, err := st.FindStoreItem(treeName + string(filepath.Separator))
		require.NoError(t, err)
		require.Equal(t, treeName, found.Meta.Filename)
	})

	// The same origin spelled several ways in one call still acts on one entry.
	makeMissing(t, item)
	require.NoError(t, RestoreCmd(sb.cfg, []string{envName, "./" + envName, item.Meta.Origin}))

	requireLinked(t, item)
	requireEntryCount(t, sb, 2)
	requireNoTempEntries(t, sb)
}
