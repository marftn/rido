package cmd

import (
	"path/filepath"
	"github.com/marftn/rido/internal/config"
	"github.com/marftn/rido/internal/store"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	envFile = ".env"
	allFlag = "--all"
)

func TestParseFlagsAndTargets(t *testing.T) {
	repo := t.TempDir()
	outside := t.TempDir()

	st := store.NewStore(config.Config{StoreRoot: filepath.Join(t.TempDir(), "store")})
	for _, origin := range []string{
		filepath.Join(repo, envFile),
		filepath.Join(repo, "config", "creds.json"),
		filepath.Join(outside, envFile),
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
		{
			name:  "paths",
			args:  []string{envFile},
			force: false,
			want:  []string{envFile},
			fails: false,
		},
		{
			name:  "force short",
			args:  []string{"-f", envFile},
			force: true,
			want:  []string{envFile},
			fails: false,
		},
		{
			name:  "force long",
			args:  []string{"--force", envFile},
			force: true,
			want:  []string{envFile},
			fails: false,
		},
		{
			name:  "no target",
			force: false,
			args:  nil,
			want:  []string{envFile},
			fails: true,
		},
		{
			name:  "all is cwd only",
			args:  []string{allFlag},
			force: false,
			want: []string{
				filepath.Join(repo, envFile),
				filepath.Join(repo, "config", "creds.json"),
			},
			fails: false,
		},
		{
			name:  "store wide",
			args:  []string{"--store-wide"},
			force: false,
			want: []string{
				filepath.Join(repo, envFile),
				filepath.Join(repo, "config", "creds.json"),
				filepath.Join(outside, envFile),
			},
			fails: false,
		},
		{
			name:  "all with a path",
			args:  []string{allFlag, envFile},
			force: false,
			want:  []string{},
			fails: true,
		},
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
	meta := store.NewMeta(filepath.Join(t.TempDir(), envFile))
	st.NewStoreItem(&meta)

	t.Chdir(t.TempDir())

	f, args := parseFlags("restore", []string{allFlag})

	_, err := targets(&st, f, args)
	require.ErrorContains(t, err, "use --store-wide")
}
