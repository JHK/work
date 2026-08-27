package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// carrying is what git reports the checkout at path is holding.
func carrying(t *testing.T, path string) string {
	t.Helper()
	return testenv.Git(t, path, "status", "--porcelain")
}

// Carrying is the whole of the move: the only difference afterwards is which
// directory the changes are in, what was staged still staged.
// docs/references/cli.md#carry.
func TestCarryTakesTheCheckoutsWorkIntoTheWorktree(t *testing.T) {
	s := repository(t)
	s.dirty()

	r := s.run("carry", "carried")

	r.came(t, result{Answered: s.at("carried")})
	require.Empty(t, carrying(t, s.Repo), "the checkout was left carrying something")
	require.Equal(t, "main", testenv.Git(t, s.Repo, "rev-parse", "--abbrev-ref", "HEAD"),
		"the checkout was left off the branch it was already on")
	require.Equal(t, "A  staged\n M tracked\n?? untracked", carrying(t, r.Answered),
		"the worktree does not carry what the checkout was carrying")
	body, err := os.ReadFile(filepath.Join(r.Answered, "tracked"))
	require.NoError(t, err)
	require.Equal(t, "changed", string(body), "the worktree's tracked file is not the change that was in the checkout")
}

// Ignored files are the machine's rather than the work's: a build tree and a
// virtual environment belong to the checkout they were made in.
func TestCarryLeavesIgnoredFilesWhereTheyAre(t *testing.T) {
	s := repository(t)
	s.dirty()

	r := s.run("carry", "carried")

	r.came(t, result{Answered: s.at("carried")})
	require.FileExists(t, filepath.Join(s.Repo, "ignored"), "the checkout's ignored file did not stay where it was")
	require.NoFileExists(t, filepath.Join(r.Answered, "ignored"), "the ignored file travelled into the worktree")
}

// A checkout with nothing in hand is refused, and the refusal names the verb
// that only creates.
func TestCarryRefusesACleanCheckout(t *testing.T) {
	s := repository(t)

	r := s.run("carry", "carried")

	r.came(t, result{Code: 1, Errored: []string{
		"this checkout carries no changes; work add carried makes the worktree and carries nothing"}})
}

// A name that has a worktree already is nowhere to carry to, and the refusal
// costs the checkout nothing.
func TestCarryRefusesANameThatAlreadyHasAWorktree(t *testing.T) {
	s := repository(t)
	s.dirty()
	path := s.opened("scratch")
	was := carrying(t, s.Repo)

	r := s.run("carry", "scratch")

	r.came(t, result{Code: 1, Errored: []string{"scratch already has a worktree; enter it with work switch scratch"}})
	require.Equal(t, was, carrying(t, s.Repo), "the checkout no longer carries what it was carrying")
	require.Empty(t, carrying(t, path), "something was carried into the worktree that was already there")
}

// The name is given on the command line: no picker stands in for it, and a tab
// press after the verb offers nothing either.
func TestCarryPutsUpNoListing(t *testing.T) {
	put := putsUp(t)
	s := repository(t, put.stub())
	s.dirty()

	r := s.run("carry")

	r.came(t, result{Code: 1}, apart)
	require.Empty(t, rows(s.completes("carry", "")), "a tab press after carry offered rows")
}

// What travels is what the checkout was carrying: an entry of the user's own is
// none of the carry's business, and stays in the stash.
func TestCarryRestoresOnlyTheEntryItMade(t *testing.T) {
	s := repository(t)
	s.dirty()
	// An entry of the user's own, from work the carry has no business restoring.
	testenv.Git(t, s.Repo, "stash", "push", "--", "tracked")

	r := s.run("carry", "carried")

	r.came(t, result{Answered: s.at("carried")})
	require.Equal(t, "A  staged\n?? untracked", carrying(t, r.Answered),
		"the worktree carries other than what the checkout was carrying")
	require.Len(t, strings.Split(strings.TrimSpace(testenv.Git(t, s.Repo, "stash", "list")), "\n"), 1,
		"the entry that was already there is gone, or the carry left one of its own")
}

// The verbs that only create and the ones that only reach leave the checkout as
// they found it, so starting a ticket is not carrying what was in hand.
func TestOnlyCarryTakesTheCheckoutsWork(t *testing.T) {
	for _, tt := range []struct{ verb, id string }{
		{"add", "made"}, {"switch", "reached"},
	} {
		t.Run(tt.verb, func(t *testing.T) {
			s := repository(t)
			s.dirty()
			s.opened("reached")
			was := carrying(t, s.Repo)

			r := s.run(tt.verb, tt.id)

			r.came(t, result{Answered: s.at(tt.id)})
			require.Equal(t, was, carrying(t, s.Repo), "the checkout no longer carries what it was carrying")
			require.Empty(t, carrying(t, r.Answered), "the worktree is more than a checkout of the branch")
		})
	}
}

// A restore that will not apply loses nothing: the entry stays in the stash, and
// the refusal names the worktree git put part of the changes in.
func TestACarryThatCannotFinishNamesWhereTheChangesAre(t *testing.T) {
	// The file the restore is about to collide with, written where only an action
	// can reach: between git making the worktree and work restoring into it.
	s := repository(t, testenv.Stub{Name: "mise", Shell: `printf 'something else entirely' > untracked`})
	s.settings(miseOn)
	s.dirty()
	path := s.at("carried")

	r := s.run("carry", "carried")

	r.came(t, result{Code: 1, Asked: []string{"mise trust"}}, apart)
	r.saying(t, "stashed", path)
	require.Contains(t, testenv.Git(t, s.Repo, "stash", "list"), "stash@{0}", "the entry was not kept")
	require.NotEmpty(t, carrying(t, path),
		"the worktree carries nothing; want the part git did put there, which the refusal names")
}

// The name is taken as it is spelled, whatever a tracker or a forge would answer
// for it, and neither is asked. docs/references/cli.md#carry.
func TestCarryNeverResolvesTheName(t *testing.T) {
	for _, name := range []string{"bd-1", "7"} {
		t.Run(name, func(t *testing.T) {
			s := tracking(t, []ticket{doable}, []ticket{doable}, forgeOn)
			s.dirty()

			r := s.run("carry", name)

			r.came(t, result{Answered: s.at(name)})
			require.True(t, s.hasBranch(name), "the worktree is not on the branch the name spells")
		})
	}
}

// Carry reaches the naming rule with no resolver in the way.
func TestCarryRefusesANameNoWorktreeCouldCarry(t *testing.T) {
	s := repository(t)
	s.dirty()

	r := s.run("carry", "../etc")

	r.came(t, result{Code: 1, Errored: []string{`"../etc" is not a usable worktree name`}})
}
