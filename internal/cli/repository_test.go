package cli

import (
	"path/filepath"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// bare stands a shell in a bare clone, which git reports no toplevel for and
// which lists a row of its own with no working tree behind it.
func bare(t *testing.T, answering ...testenv.Stub) *session {
	t.Helper()
	s := repository(t, answering...)
	dir := t.TempDir()
	testenv.Git(t, dir, "clone", "--bare", s.Repo, "repo.git")
	s.Repo = resolved(t, filepath.Join(dir, "repo.git"))
	s.Dir = s.Repo
	return s
}

// No tool is asked anything: R5 of docs/rules/refusals.md. Every verb opens a
// repository, so go stands for them all.
func TestOutsideARepositoryTheRefusalIsWorksOwn(t *testing.T) {
	s := repository(t)
	s.Dir = t.TempDir()
	s.settings(systemsOn("beads", "github"))

	r := s.run("go")

	r.came(t, result{Code: 1, Errored: []string{"no git repository here"}})

	// git translates that answer, and the refusal is read apart from it. A machine
	// without the locale installed answers in English anyway, which still passes.
	t.Setenv("LC_ALL", "de_DE.UTF-8")

	translated := s.run("go")

	translated.came(t, result{Code: 1, Errored: []string{"no git repository here"}})
}

// A bare clone is the repository itself, from the directory it sits in and from
// inside its worktrees, and the row it lists for itself is no worktree to reach.
func TestABareRepositoryIsTheRepositoryItself(t *testing.T) {
	put := putsUp(t)
	s := bare(t, put.dismisses())
	one := s.opened("one")
	s.opened("two")

	r := s.run("list")

	r.came(t, result{Out: "one\ntwo\n"})

	// Nothing holds the bare directory, so a screen is the other thing that could
	// put a row up for it.
	putting := s.run("go")

	putting.came(t, result{Code: 1, Asked: []string{putUp}})
	testenv.Equal(t, []string{"one", "two"}, retyped(put.rows()), "the row a bare repository lists for itself was put up")

	s.Dir = one

	inside := s.run("list")

	inside.came(t, result{Out: "one\ntwo\n"})
}

// The bare directory is both the repository a worktree lands under and the
// checkout its branch forks from, there being no other.
func TestAWorktreeUnderABareRepositoryForksFromItsHead(t *testing.T) {
	s := bare(t)

	r := s.run("add", "two")

	r.came(t, result{Answered: s.at("two")})
	require.Equal(t, testenv.Git(t, s.Repo, "rev-parse", "HEAD"), testenv.Git(t, r.Answered, "rev-parse", "HEAD"),
		"the worktree is not at the bare repository's HEAD")
}
