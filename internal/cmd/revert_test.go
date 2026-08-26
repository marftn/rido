package cmd

import (
	"path/filepath"
	"testing"

	"github.com/marftn/rido/internal/fs"
	"github.com/marftn/rido/internal/store"

	"github.com/stretchr/testify/require"
)

func TestRevertPutsTheFileBack(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, *store.StoreItem)
		args  []string
	}{
		{name: "linked", setup: func(*testing.T, *store.StoreItem) {}, args: []string{envName}},
		{name: "missing", setup: makeMissing, args: []string{envName}},
		{
			name:  "occupied with force",
			setup: func(t *testing.T, i *store.StoreItem) { t.Helper(); makeOccupied(t, i) },
			args:  []string{"-f", envName},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sb := newSandbox(t)
			item := sb.addEnv(t)

			tc.setup(t, item)

			require.NoError(t, RevertCmd(sb.cfg, tc.args))

			requireRegularFile(t, item.Meta.Origin)
			requireContent(t, item.Meta.Origin, envBody)
			requireMode(t, item.Meta.Origin, fs.FileModeReadOnly)
			require.False(t, fs.Exists(item.Path()), "entry %s survived", item.ID)
			requireEntryCount(t, sb, 0)
			requireNoTempEntries(t, sb)
		})
	}
}

func TestRevertAcceptsAnID(t *testing.T) {
	sb := newSandbox(t)
	item := sb.addEnv(t)

	require.NoError(t, RevertCmd(sb.cfg, []string{item.ID.String()}))

	requireRegularFile(t, item.Meta.Origin)
	requireEntryCount(t, sb, 0)
}

// The origin directory is recreated, so no entry is left stranded in the store.
func TestRevertRecreatesAMissingOriginDirectory(t *testing.T) {
	sb := newSandbox(t)
	name := filepath.Join("sub", envName)
	sb.write(t, name, envBody, fs.FileModeReadOnly)
	require.NoError(t, AddCmd(sb.cfg, []string{name}))

	item := sb.only(t)
	makeStale(t, item)

	require.NoError(t, RevertCmd(sb.cfg, []string{item.ID.String()}))

	requireRegularFile(t, item.Meta.Origin)
	requireContent(t, item.Meta.Origin, envBody)
	requireEntryCount(t, sb, 0)
}

func TestRevertRestoresADirectoryTree(t *testing.T) {
	sb := newSandbox(t)
	want := makeTree(t, filepath.Join(sb.root, "want"))
	origin := makeTree(t, filepath.Join(sb.repo, treeName))

	require.NoError(t, AddCmd(sb.cfg, []string{treeName}))
	require.NoError(t, RevertCmd(sb.cfg, []string{treeName}))

	requireTreeEqual(t, want, origin)
	requireEntryCount(t, sb, 0)
	requireNoTempEntries(t, sb)
}

// Under `go test` stdin is empty, so the question is answered no.
func TestRevertSkipsAnOccupiedOriginWithoutForce(t *testing.T) {
	sb := newSandbox(t)
	item := sb.addEnv(t)

	makeOccupied(t, item)
	before := treeOf(t, sb.root)

	require.ErrorIs(t, revertOne(sb.load(t), envName, false), errSkipped)
	require.ErrorIs(t, RevertCmd(sb.cfg, []string{envName}), ErrReported)

	require.Equal(t, before, treeOf(t, sb.root))
	requireEntryCount(t, sb, 1)
}

func TestRevertRefusesABrokenEntry(t *testing.T) {
	sb := newSandbox(t)
	item := sb.addEnv(t)

	makeBroken(t, item)

	require.ErrorContains(
		t,
		revertOne(sb.load(t), envName, true),
		"payload is missing from store entry",
	)
	require.ErrorIs(t, RevertCmd(sb.cfg, []string{"-f", envName}), ErrReported)

	requireEntryCount(t, sb, 1)
	requireStatus(t, item, store.StatusBroken)
}

// The file comes back under a new ID, with nothing left from the old entry.
func TestRevertThenAddAgainMakesANewEntry(t *testing.T) {
	sb := newSandbox(t)
	first := sb.addEnv(t)

	require.NoError(t, RevertCmd(sb.cfg, []string{envName}))
	require.NoError(t, AddCmd(sb.cfg, []string{envName}))

	second := sb.only(t)
	require.NotEqual(t, first.ID, second.ID)
	requireLinked(t, second)
	requireContent(t, second.PayloadPath(), envBody)
	requireEntryCount(t, sb, 1)
}
