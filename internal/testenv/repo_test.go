package testenv_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

func TestInitRepoHandsBackWhatGitWouldHaveBuilt(t *testing.T) {
	got := testenv.InitRepo(t)
	want := t.TempDir()
	testenv.Git(t, want, "init", "-b", "main")
	testenv.Git(t, want, "commit", "--allow-empty", "-m", "root")

	for _, ask := range [][]string{
		{"rev-parse", "--abbrev-ref", "HEAD"},
		{"rev-list", "--count", "HEAD"},
		{"ls-tree", "-r", "--name-only", "HEAD"},
		{"config", "--local", "--list"},
		{"status", "--porcelain"},
	} {
		asked := "git " + strings.Join(ask, " ")
		require.Equal(t, testenv.Git(t, want, ask...), testenv.Git(t, got, ask...), asked+" reads differently in a copied repository")
	}
}

// A copy carries nothing of where it was built: it takes a worktree of its own,
// and the next caller gets none of it.
func TestEachRepositoryIsTheCallersOwn(t *testing.T) {
	first, second := testenv.InitRepo(t), testenv.InitRepo(t)
	testenv.Git(t, first, "worktree", "add", "-b", "one", filepath.Join(t.TempDir(), "one"))

	require.NotEmpty(t, testenv.Git(t, first, "branch", "--list", "one"), "the branch added to a copied repository is not in it")
	require.Empty(t, testenv.Git(t, second, "branch", "--list", "one"), "a repository carries what was written to the one before it")
}
