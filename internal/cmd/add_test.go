package cmd

import (
	"context"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/marftn/rido/internal/fs"
	"github.com/marftn/rido/internal/store"

	"github.com/go-git/go-git/v6"
	"github.com/stretchr/testify/require"
)

func TestAddLinksAFile(t *testing.T) {
	sb := newSandbox(t)
	origin := sb.write(t, envName, envBody, fs.FileModeReadOnly)

	require.NoError(t, AddCmd(sb.cfg, []string{envName}))

	item := sb.only(t)
	require.Equal(t, origin, item.Meta.Origin)
	require.Equal(t, envName, item.Meta.Filename)
	require.Equal(t, store.MetaVersion, item.Meta.Version)
	require.Equal(t, filepath.Join(sb.cfg.StoreRoot, item.ID.String(), envName), item.PayloadPath())

	requireLinked(t, item)
	requireContent(t, item.PayloadPath(), envBody)
	requireMode(t, item.PayloadPath(), fs.FileModeReadOnly)
	requireNoTempEntries(t, sb)
}

func TestAddLinksEveryFileOfOneCall(t *testing.T) {
	sb := newSandbox(t)
	names := []string{envName, "creds.json", filepath.Join("config", "token")}

	for _, name := range names {
		sb.write(t, name, filepath.Base(name), fs.FileModeReadOnly)
	}

	require.NoError(t, AddCmd(sb.cfg, names))

	st := sb.load(t)
	require.Len(t, st.Items, len(names))

	for i := range st.Items {
		item := &st.Items[i]
		requireLinked(t, item)
		requireContent(t, item.PayloadPath(), item.Meta.Filename)
	}
}

func TestAddMovesADirectoryTreeIntact(t *testing.T) {
	sb := newSandbox(t)
	want := makeTree(t, filepath.Join(sb.root, "want"))
	origin := makeTree(t, filepath.Join(sb.repo, treeName))

	require.NoError(t, AddCmd(sb.cfg, []string{treeName}))

	item := sb.only(t)
	require.Equal(t, origin, item.Meta.Origin)

	requireLinked(t, item)
	requireTreeEqual(t, want, item.PayloadPath())
	requireNoTempEntries(t, sb)
}

func TestAddPreservesModes(t *testing.T) {
	tests := []struct {
		name string
		mode os.FileMode
	}{
		{name: "private", mode: fs.FileModeReadOnly},
		{name: "shared", mode: fs.FileModeShared},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sb := newSandbox(t)
			sb.write(t, envName, envBody, tc.mode)

			require.NoError(t, AddCmd(sb.cfg, []string{envName}))

			requireMode(t, sb.only(t).PayloadPath(), tc.mode)
		})
	}
}

func TestAddCopiesALargeFileExactly(t *testing.T) {
	const size = 1 << 20

	sb := newSandbox(t)

	// A fixed seed, so a failure can be replayed.
	source := rand.NewChaCha8([32]byte{7})
	body := make([]byte, size)
	_, err := source.Read(body)
	require.NoError(t, err)

	sb.write(t, envName, string(body), fs.FileModeReadOnly)
	require.NoError(t, AddCmd(sb.cfg, []string{envName}))

	requireContent(t, sb.only(t).PayloadPath(), string(body))
}

func TestAddLeavesTheContentReadableThroughTheSymlink(t *testing.T) {
	sb := newSandbox(t)
	item := sb.addEnv(t)

	requireContent(t, item.Meta.Origin, envBody)
}

func TestAddRejectsTheWholeCallOnAPreconditionFailure(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  string
	}{
		{name: "missing", files: []string{"nope"}, want: "file does not exist: nope"},
		{name: "no file", files: nil, want: "at least one file must be added"},
		{
			name:  "duplicate",
			files: []string{envName, envName},
			want:  "is present more than once",
		},
		{
			name:  "nested",
			files: []string{treeName, filepath.Join(treeName, "db.json")},
			want:  "nested paths",
		},
		{
			// Every path is checked before anything is moved, so one bad path
			// leaves the good ones untouched.
			name:  "one missing among good ones",
			files: []string{envName, "nope"},
			want:  "file does not exist: nope",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sb := newSandbox(t)
			sb.write(t, envName, envBody, fs.FileModeReadOnly)
			makeTree(t, filepath.Join(sb.repo, treeName))

			require.ErrorContains(t, AddCmd(sb.cfg, tc.files), tc.want)
			requireEntryCount(t, sb, 0)
			requireRegularFile(t, filepath.Join(sb.repo, envName))
		})
	}
}

func TestAddRefusesAFileItAlreadyManages(t *testing.T) {
	sb := newSandbox(t)
	item := sb.addEnv(t)

	require.ErrorContains(t, addFile(sb.load(t), envName), "already managed by rido")
	require.ErrorIs(t, AddCmd(sb.cfg, []string{envName}), ErrReported)

	requireEntryCount(t, sb, 1)
	requireLinked(t, sb.only(t))
	requireContent(t, item.PayloadPath(), envBody)
}

// A path that fails while being added does not hold back the others.
func TestAddIsIndependentPerPath(t *testing.T) {
	sb := newSandbox(t)
	sb.addEnv(t)
	sb.write(t, "creds.json", "{}", fs.FileModeReadOnly)

	require.ErrorIs(t, AddCmd(sb.cfg, []string{envName, "creds.json"}), ErrReported)

	st := sb.load(t)
	require.Len(t, st.Items, 2)
}

func TestAddOutsideARepository(t *testing.T) {
	sb := newSandbox(t)
	requireLinked(t, sb.addEnv(t))
}

func TestAddRefusesAFileTrackedByGit(t *testing.T) {
	sb := newSandbox(t)
	sb.write(t, envName, envBody, fs.FileModeReadOnly)

	repository, err := git.PlainInit(sb.repo, false)
	require.NoError(t, err)

	worktree, err := repository.Worktree()
	require.NoError(t, err)

	_, err = worktree.Add(envName)
	require.NoError(t, err)

	require.ErrorContains(t, addFile(sb.load(t), envName), "git rm -r --cached "+envName)
	require.ErrorIs(t, AddCmd(sb.cfg, []string{envName}), ErrReported)

	requireEntryCount(t, sb, 0)
	requireRegularFile(t, filepath.Join(sb.repo, envName))
}

// A real git index must be read the same way as a go-git one.
func TestAddRefusesAFileTrackedBySystemGit(t *testing.T) {
	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not on PATH")
	}

	sb := newSandbox(t)
	sb.write(t, envName, envBody, fs.FileModeReadOnly)

	ctx := context.Background()

	for _, args := range [][]string{{"init"}, {"add", envName}} {
		cmd := exec.CommandContext(ctx, gitBin, args...)
		cmd.Dir = sb.repo
		cmd.Env = []string{"HOME=" + sb.root, "PATH=/usr/bin:/bin", "GIT_CONFIG_NOSYSTEM=1"}

		out, e := cmd.CombinedOutput()
		require.NoError(t, e, "git %v: %s", args, out)
	}

	require.ErrorContains(t, addFile(sb.load(t), envName), "git rm -r --cached "+envName)
	requireEntryCount(t, sb, 0)
}

func TestAddAcceptsAGitignoredFile(t *testing.T) {
	sb := newSandbox(t)
	ignore := sb.write(t, ".gitignore", "/"+envName+"\n", fs.FileModeShared)

	_, err := git.PlainInit(sb.repo, false)
	require.NoError(t, err)

	requireLinked(t, sb.addEnv(t))
	requireContent(t, ignore, "/"+envName+"\n")
}

// Under `go test` stdin is empty, so the gitignore question is answered no and
// the file is skipped.
func TestAddSkipsAFileThatIsNeitherTrackedNorIgnored(t *testing.T) {
	sb := newSandbox(t)
	ignore := sb.write(t, ".gitignore", "/other\n", fs.FileModeShared)
	sb.write(t, envName, envBody, fs.FileModeReadOnly)

	_, err := git.PlainInit(sb.repo, false)
	require.NoError(t, err)

	require.ErrorIs(t, addFile(sb.load(t), envName), errSkipped)
	require.ErrorIs(t, AddCmd(sb.cfg, []string{envName}), ErrReported)

	requireEntryCount(t, sb, 0)
	requireRegularFile(t, filepath.Join(sb.repo, envName))
	requireContent(t, ignore, "/other\n")
}
