package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/marftn/rido/internal/fs"
	"github.com/marftn/rido/internal/store"

	"github.com/stretchr/testify/require"
)

// An agent replaced the symlink with a file of its own. restore -f wins it back.
func TestStoryAgentOverwritesTheLink(t *testing.T) {
	sb := newSandbox(t)
	item := sb.addEnv(t)

	require.NoError(t, os.Remove(item.Meta.Origin))
	require.NoError(t, os.WriteFile(item.Meta.Origin, []byte("SECRET=overwritten"), fs.FileModeShared))

	require.NoError(t, RestoreCmd(sb.cfg, []string{"-f", envName}))

	requireLinked(t, item)
	requireContent(t, item.Meta.Origin, envBody)
	requireNoTempEntries(t, sb)
}

func TestStoryStopManaging(t *testing.T) {
	sb := newSandbox(t)
	item := sb.addEnv(t)

	require.NoError(t, RevertCmd(sb.cfg, []string{envName}))

	requireRegularFile(t, item.Meta.Origin)
	requireContent(t, item.Meta.Origin, envBody)
	requireMode(t, item.Meta.Origin, fs.FileModeReadOnly)
	requireEntryCount(t, sb, 0)
	requireNoTempEntries(t, sb)
}

// A file written into the linked directory lands in the store, and comes back
// with the rest of the tree on revert.
func TestStoryWholeDirectory(t *testing.T) {
	sb := newSandbox(t)
	want := makeTree(t, filepath.Join(sb.root, "want"))
	origin := makeTree(t, filepath.Join(sb.repo, treeName))

	require.NoError(t, AddCmd(sb.cfg, []string{treeName}))

	item := sb.only(t)
	added := "added-while-linked"
	require.NoError(t, os.WriteFile(filepath.Join(origin, added), []byte("later"), fs.FileModeShared))
	require.NoError(t, os.WriteFile(filepath.Join(want, added), []byte("later"), fs.FileModeShared))

	requireRegularFile(t, filepath.Join(item.PayloadPath(), added))

	require.NoError(t, RevertCmd(sb.cfg, []string{treeName}))

	requireTreeEqual(t, want, origin)
	requireEntryCount(t, sb, 0)
}

func TestStoryChurn(t *testing.T) {
	sb := newSandbox(t)
	names := []string{"a.env", "b.env", "c.env"}

	for _, name := range names {
		sb.write(t, name, name, fs.FileModeReadOnly)
	}

	require.NoError(t, AddCmd(sb.cfg, names))
	requireEntryCount(t, sb, 3)

	require.NoError(t, RevertCmd(sb.cfg, []string{"a.env"}))
	requireEntryCount(t, sb, 2)
	requireRegularFile(t, filepath.Join(sb.repo, "a.env"))

	makeMissing(t, findByName(t, sb, "b.env"))
	require.NoError(t, RestoreCmd(sb.cfg, []string{"b.env"}))

	sb.write(t, "d.env", "d.env", fs.FileModeReadOnly)
	require.NoError(t, AddCmd(sb.cfg, []string{"d.env"}))

	st := sb.load(t)
	require.Len(t, st.Items, 3)

	ids := []string{}

	for i := range st.Items {
		item := &st.Items[i]
		requireStatus(t, item, store.StatusLinked)
		requireContent(t, item.PayloadPath(), item.Meta.Filename)

		ids = append(ids, item.ID.String())
	}

	// The store holds those entries and nothing else.
	entries, err := os.ReadDir(sb.cfg.StoreRoot)
	require.NoError(t, err)

	names = []string{}
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	require.ElementsMatch(t, ids, names)
	requireNoTempEntries(t, sb)
}

func findByName(t *testing.T, sb sandbox, name string) *store.StoreItem {
	t.Helper()

	st := sb.load(t)

	item, err := st.FindStoreItem(name)
	require.NoError(t, err)

	return item
}
