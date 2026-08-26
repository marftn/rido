package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/marftn/rido/internal/fs"
	"github.com/marftn/rido/internal/store"

	"github.com/stretchr/testify/require"
)

func TestRestoreRelinksAMissingSymlink(t *testing.T) {
	sb := newSandbox(t)
	item := sb.addEnv(t)

	makeMissing(t, item)

	require.NoError(t, RestoreCmd(sb.cfg, []string{envName}))

	requireLinked(t, item)
	requireContent(t, item.PayloadPath(), envBody)
	requireNoTempEntries(t, sb)
}

func TestRestoreAcceptsAnID(t *testing.T) {
	sb := newSandbox(t)
	item := sb.addEnv(t)

	makeMissing(t, item)

	require.NoError(t, RestoreCmd(sb.cfg, []string{item.ID.String()}))

	requireLinked(t, item)
}

func TestRestoreIsANoOpOnALinkedEntry(t *testing.T) {
	sb := newSandbox(t)
	item := sb.addEnv(t)

	require.NoError(t, RestoreCmd(sb.cfg, []string{envName}))

	requireLinked(t, item)
	requireContent(t, item.Meta.Origin, envBody)
}

func TestRestoreIsIdempotent(t *testing.T) {
	sb := newSandbox(t)
	item := sb.addEnv(t)

	makeMissing(t, item)

	require.NoError(t, RestoreCmd(sb.cfg, []string{envName}))

	before := treeOf(t, sb.root)

	require.NoError(t, RestoreCmd(sb.cfg, []string{envName}))

	require.Equal(t, before, treeOf(t, sb.root))
}

func TestRestoreReplacesAnOccupiedOriginWithForce(t *testing.T) {
	tests := []struct {
		name    string
		occupy  func(*testing.T, *store.StoreItem)
		wantWas string
	}{
		{
			name:    fs.FileDescRegularFile,
			occupy:  func(t *testing.T, i *store.StoreItem) { t.Helper(); makeOccupied(t, i) },
			wantWas: fs.FileDescRegularFile,
		},
		{
			name:    fs.FileDescDanglingSymlink,
			occupy:  makeDangling,
			wantWas: fs.FileDescDanglingSymlink,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sb := newSandbox(t)
			item := sb.addEnv(t)

			tc.occupy(t, item)
			require.Contains(t, fs.Describe(item.Meta.Origin), tc.wantWas)

			require.NoError(t, RestoreCmd(sb.cfg, []string{"-f", envName}))

			requireLinked(t, item)
			requireContent(t, item.PayloadPath(), envBody)
			requireNoTempEntries(t, sb)
		})
	}
}

// Under `go test` stdin is empty, so the question is answered no.
func TestRestoreSkipsAnOccupiedOriginWithoutForce(t *testing.T) {
	tests := []struct {
		name   string
		occupy func(*testing.T, *store.StoreItem)
	}{
		{
			name:   fs.FileDescRegularFile,
			occupy: func(t *testing.T, i *store.StoreItem) { t.Helper(); makeOccupied(t, i) },
		},
		{name: fs.FileDescDanglingSymlink, occupy: makeDangling},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sb := newSandbox(t)
			item := sb.addEnv(t)

			tc.occupy(t, item)
			before := treeOf(t, sb.root)

			require.ErrorIs(t, restoreOne(sb.load(t), envName, false), errSkipped)
			require.ErrorIs(t, RestoreCmd(sb.cfg, []string{envName}), ErrReported)

			require.Equal(t, before, treeOf(t, sb.root))
			requireEntryCount(t, sb, 1)
		})
	}
}

func TestRestoreRefusesAnEntryItCannotRelink(t *testing.T) {
	tests := []struct {
		name       string
		breakIt    func(*testing.T, *store.StoreItem)
		want       string
		wantStatus store.Status
	}{
		{
			name:       "stale",
			breakIt:    makeStale,
			want:       "origin directory of store item",
			wantStatus: store.StatusStale,
		},
		{
			name:       "broken",
			breakIt:    makeBroken,
			want:       "payload is missing from store entry",
			wantStatus: store.StatusBroken,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sb := newSandbox(t)
			name := filepath.Join("sub", envName)
			sb.write(t, name, envBody, fs.FileModeReadOnly)
			require.NoError(t, AddCmd(sb.cfg, []string{name}))

			item := sb.only(t)
			tc.breakIt(t, item)

			require.ErrorContains(t, restoreOne(sb.load(t), item.ID.String(), true), tc.want)
			require.ErrorIs(t, RestoreCmd(sb.cfg, []string{"-f", item.ID.String()}), ErrReported)

			requireEntryCount(t, sb, 1)
			requireStatus(t, item, tc.wantStatus)
		})
	}
}

func TestRestoreRefusesAnUnknownTarget(t *testing.T) {
	sb := newSandbox(t)
	sb.addEnv(t)

	_, err := os.Create(filepath.Join(sb.repo, "other"))
	require.NoError(t, err)

	require.ErrorIs(t, restoreOne(sb.load(t), "other", false), store.ErrNotFound)
	require.ErrorIs(t, RestoreCmd(sb.cfg, []string{"other"}), ErrReported)
}
