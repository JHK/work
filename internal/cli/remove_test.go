package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// Removing is git's two commands and nothing besides: the worktree goes, its
// branch goes with it, and a line says each. No ticket is touched and no tracker
// asked: a worktree going away reaches neither seam.
func TestRemoveTakesTheWorktreeAndItsBranch(t *testing.T) {
	s := repository(t)
	path := s.opened("scratch")
	// A commit of its own is what git would refuse to delete the branch over.
	testenv.Git(t, path, "commit", "--allow-empty", "-m", "never landed")

	r := s.run("remove", "scratch")

	r.came(t, result{Out: "removed worktree " + path + "\ndeleted branch scratch\n"})
	require.NoDirExists(t, path, "the worktree is still on disk")
	require.False(t, s.hasWorktree(path), "git still reports the worktree")
	require.False(t, s.hasBranch("scratch"), "branch scratch is still there")
}

func TestRemoveADetachedWorktreeTakesNoBranch(t *testing.T) {
	s := repository(t)
	path := s.detached("adrift")

	r := s.run("remove", "adrift")

	r.came(t, result{Out: "removed worktree " + path + "\n"})
	require.NoDirExists(t, path, "the worktree is still on disk")
}

// With no name the picker stands in for it, over the worktrees open, and --force
// is read all the same.
func TestRemoveWithNoNameTakesThePickersRow(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no name", []string{"remove"}},
		{"forced with no name", []string{"remove", "--force"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The row index is fzf's answer, and the main checkout is not among the rows.
			s := repository(t, testenv.Stub{Name: "fzf", Says: "0\tscratch\n"})
			path := s.opened("scratch")

			r := s.run(tt.args...)

			r.came(t, result{Out: "removed worktree " + path + "\ndeleted branch scratch\n", Asked: []string{putUp}})
			require.NoDirExists(t, path, "the worktree the picker offered is still on disk")
		})
	}
}

// A worktree carrying changes is work's own to refuse, ahead of git and naming
// --force: see R5 of docs/rules/refusals.md. Nothing is half removed, and
// --force takes it wherever that sits.
func TestRemoveRefusesAWorktreeCarryingChanges(t *testing.T) {
	tests := []struct {
		name  string
		force []string
	}{
		{"forced before the name", []string{"remove", "--force", "scratch"}},
		{"forced after the name", []string{"remove", "scratch", "--force"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			path := s.opened("scratch")
			testenv.Write(t, filepath.Join(path, "new"), "loose")

			r := s.run("remove", "scratch")

			r.came(t, result{Code: 1, Errored: []string{"scratch carries changes; take it and them with work remove --force scratch"}})
			require.DirExists(t, path, "the worktree went despite the refusal")
			require.True(t, s.hasBranch("scratch"), "the branch went despite the refusal")

			forced := s.run(tt.force...)

			forced.came(t, result{Out: "removed worktree " + path + "\ndeleted branch scratch\n"})
			require.NoDirExists(t, path, "--force left the worktree behind")
			require.False(t, s.hasBranch("scratch"), "--force left branch scratch behind")
		})
	}
}

// The main checkout cannot be removed, with or without --force. Every worktree
// sits under it, so the refusal survives being typed from inside one.
func TestRemoveRefusesTheMainCheckout(t *testing.T) {
	tests := []struct {
		name    string
		stoodIn bool
	}{
		{"from the main checkout", false},
		{"from a worktree under it", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			scratch := s.opened("scratch")
			if tt.stoodIn {
				s.Dir = scratch
			}

			for _, args := range [][]string{{"remove", "main"}, {"remove", "--force", "main"}} {
				r := s.run(args...)

				r.came(t, result{Code: 1, Errored: []string{"main is the main checkout; work remove acts on a worktree under it"}})
			}
			require.True(t, s.hasWorktree(s.Repo), "the main checkout is no longer a worktree git reports")
			require.True(t, s.hasBranch("main"), "the refusal cost the main checkout its branch")
		})
	}
}

// git would take the worktree the shell stands in and leave it in a directory
// that is gone, so the refusal is work's own. A worktree nested in that one goes
// with it, so standing in the nested one is standing in this one.
func TestRemoveRefusesTheWorktreeStoodIn(t *testing.T) {
	tests := []struct {
		name string
		// stoodIn is where the shell stands, which the case makes under the worktree.
		stoodIn func(t *testing.T, s *session, scratch string) string
	}{
		{"the worktree itself", func(_ *testing.T, _ *session, scratch string) string { return scratch }},
		{"a directory under it", func(t *testing.T, _ *session, scratch string) string {
			// A directory and nothing in it, so the worktree is clean and only the rule
			// about where the shell stands can be what refuses.
			within := filepath.Join(scratch, "within")
			require.NoError(t, os.MkdirAll(within, 0o755))
			return within
		}},
		{"a worktree nested in it", func(t *testing.T, s *session, scratch string) string {
			nested := filepath.Join(scratch, "nested")
			testenv.Git(t, s.Repo, "worktree", "add", "-b", "nested", nested)
			return nested
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			scratch := s.opened("scratch")
			s.Dir = tt.stoodIn(t, s, scratch)

			// --force takes an unclean worktree; it is no way past this one.
			for _, args := range [][]string{{"remove", "scratch"}, {"remove", "--force", "scratch"}} {
				r := s.run(args...)

				r.came(t, result{Code: 1, Errored: []string{"scratch is the worktree you are standing in; run work remove from outside it"}})
			}
			require.DirExists(t, scratch, "the worktree went despite the refusal")
			require.True(t, s.hasBranch("scratch"), "the branch went despite the refusal")
		})
	}
}

// Nothing to remove is refused before anything is touched, and the refusal
// invites no verb that would create it. A name nothing answers for and a place
// a system answers for that has no worktree read as their own thing.
func TestRemoveWhatIsNotThere(t *testing.T) {
	tests := []struct {
		name     string
		settings string
		target   string
		said     string
	}{
		{
			"a name nothing answers for",
			"", "nowhere",
			"nothing answers for \"nowhere\"",
		},
		{
			// A bare number is the forge's by its spelling alone, so the place is made
			// without gh being asked for it.
			"a place with no worktree open",
			forgeOn, "7",
			"pr-7 has no worktree to remove",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			s.settings(tt.settings)
			scratch := s.opened("scratch")

			r := s.run("remove", tt.target)

			r.came(t, result{Code: 1, Errored: []string{tt.said}})
			require.DirExists(t, scratch, "the worktree that is there went with the refusal")
		})
	}
}
