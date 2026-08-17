package cmd

import (
	"path/filepath"
	"rido/internal/config"
	"rido/internal/store"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFlagsAndTargets(t *testing.T) {
	repo := t.TempDir()
	outside := t.TempDir()

	st := store.NewStore(config.Config{StoreRoot: filepath.Join(t.TempDir(), "store")})
	for _, origin := range []string{
		filepath.Join(repo, ".env"),
		filepath.Join(repo, "config", "creds.json"),
		filepath.Join(outside, ".env"),
	} {
		meta := store.NewMeta(origin)
		st.NewStoreItem(&meta)
	}

	t.Chdir(repo)

	tests := []struct {
		name  string
		args  []string
		force bool
		want  []string
		fails bool
	}{
		{name: "paths", args: []string{".env"}, want: []string{".env"}},
		{name: "force short", args: []string{"-f", ".env"}, force: true, want: []string{".env"}},
		{
			name:  "force long",
			args:  []string{"--force", ".env"},
			force: true,
			want:  []string{".env"},
		},
		{name: "no target", args: nil, fails: true},
		{
			name: "all is cwd only",
			args: []string{"--all"},
			want: []string{
				filepath.Join(repo, ".env"),
				filepath.Join(repo, "config", "creds.json"),
			},
		},
		{
			name: "store wide",
			args: []string{"--store-wide"},
			want: []string{
				filepath.Join(repo, ".env"),
				filepath.Join(repo, "config", "creds.json"),
				filepath.Join(outside, ".env"),
			},
		},
		{name: "all with a path", args: []string{"--all", ".env"}, fails: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, args := parseFlags("restore", tc.args)
			require.Equal(t, tc.force, f.force)

			got, err := targets(&st, f, args)
			if tc.fails {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestTargetsNoEntryUnderCwd(t *testing.T) {
	st := store.NewStore(config.Config{StoreRoot: filepath.Join(t.TempDir(), "store")})
	meta := store.NewMeta(filepath.Join(t.TempDir(), ".env"))
	st.NewStoreItem(&meta)

	t.Chdir(t.TempDir())

	f, args := parseFlags("restore", []string{"--all"})

	_, err := targets(&st, f, args)
	require.ErrorContains(t, err, "use --store-wide")
}
