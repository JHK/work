package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// Moving is git's two commands and nothing besides: the directory moves, the
// branch takes the destination's last element with it, and a line says each.
func TestMoveTakesTheWorktreeAndItsBranch(t *testing.T) {
	s := repository(t)
	from := s.opened("scratch")
	to := s.at("settled")

	r := s.run("move", "scratch", "settled")

	r.came(t, result{Out: "moved worktree " + from + " to " + to + "\nrenamed branch scratch to settled\n"})
	require.NoDirExists(t, from, "the worktree is still where it was")
	require.True(t, s.hasWorktree(to), "git does not report the worktree where it landed")
	require.False(t, s.hasBranch("scratch"), "branch scratch is still there")

	// The branch is the address, so the worktree is reached under the name it took.
	again := s.run("switch", "settled")

	again.came(t, result{Answered: to})
}

// A bare name lands the worktree beside where it sits; one carrying a separator
// is a path. The last element names the branch either way.
func TestMoveReadsTheDestination(t *testing.T) {
	elsewhere := t.TempDir()
	tests := []struct {
		name, dest string
		want       func(s *session) string
	}{
		{"a bare name", "settled", func(s *session) string { return s.at("settled") }},
		{"a path of its own", filepath.Join(elsewhere, "settled"), func(*session) string { return filepath.Join(elsewhere, "settled") }},
		{"a relative path", filepath.Join("..", "beside"), func(s *session) string { return filepath.Join(s.Repo, "beside") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			from := s.opened("scratch")
			// A relative destination is read from here, never from the worktree.
			s.Dir = defaultDir(s.Repo)

			r := s.run("move", "scratch", "--", tt.dest)

			to := tt.want(s)
			r.came(t, result{Out: "moved worktree " + from + " to " + to + "\nrenamed branch scratch to " + filepath.Base(to) + "\n"})
			require.DirExists(t, to, "the worktree did not land where the destination named")
		})
	}
}

// Each is refused before anything moves: neither half of it lands.
func TestMoveRefusesTheDestination(t *testing.T) {
	tests := []struct {
		name, dest string
		said       func(s *session) string
	}{
		{"a directory already there", "occupied", func(s *session) string {
			return s.at("occupied") + " is already there; take that directory away first"
		}},
		{"a branch already there", "spoken-for", func(*session) string { return "branch spoken-for already exists" }},
		{"where it already sits", "scratch", func(s *session) string {
			return s.at("scratch") + " is where scratch already sits"
		}},
		// It would reach git branch --move as a switch, never its branch-name check.
		{"a name opening with a dash", "-foo", func(*session) string {
			return `git will not name a branch "-foo"`
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			from := s.opened("scratch")
			// git moves a worktree inside a directory already there rather than refusing it,
			// so a plain one is the case the refusal is for.
			require.NoError(t, os.Mkdir(s.at("occupied"), 0o755))
			testenv.Git(t, s.Repo, "branch", "spoken-for")

			r := s.run("move", "scratch", "--", tt.dest)

			r.came(t, result{Code: 1, Errored: []string{tt.said(s)}})
			require.DirExists(t, from, "the worktree moved despite the refusal")
			require.True(t, s.hasBranch("scratch"), "the branch was renamed despite the refusal")
		})
	}
}

// The branch is renamed before the directory moves, so a name only git can
// refuse costs nothing: neither half lands, and no rollback was needed.
func TestMoveRefusesANameGitWillNotTakeBeforeAnythingMoves(t *testing.T) {
	s := repository(t)
	from := s.opened("scratch")

	r := s.run("move", "scratch", "--", "a b")

	r.refused(t, "git branch --move scratch a b", "is not a valid branch name")
	require.DirExists(t, from, "the worktree moved despite the refusal")
	require.True(t, s.hasBranch("scratch"), "the branch was renamed despite the refusal")
}

// The refusal has to survive being asked for from inside a worktree that sits
// under the main worktree.
func TestMoveRefusesTheMainWorktree(t *testing.T) {
	s := repository(t)
	testenv.Git(t, s.Repo, "branch", "--move", "trunk")
	s.Dir = s.opened("scratch")

	r := s.run("move", "trunk", filepath.Join(t.TempDir(), "elsewhere"))

	r.came(t, result{Code: 1, Errored: []string{"trunk is the main worktree; work move acts on a worktree under it"}})
	require.Equal(t, "trunk", testenv.Git(t, s.Repo, "rev-parse", "--abbrev-ref", "HEAD"), "the main worktree was left off trunk")
	require.True(t, s.hasWorktree(s.Repo), "the main worktree is no longer a worktree git reports")
}

func TestMoveADetachedWorktreeTakesNoBranch(t *testing.T) {
	s := repository(t)
	from := s.detached("adrift")
	to := s.at("settled")

	r := s.run("move", "adrift", "settled")

	r.came(t, result{Out: "moved worktree " + from + " to " + to + "\n"})
	require.DirExists(t, to, "the worktree did not land")
	require.False(t, s.hasBranch("settled"), "a branch a detached worktree never had was made")
}
