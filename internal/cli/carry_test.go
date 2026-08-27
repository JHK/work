package cli

import (
	"os"
	"path/filepath"
	"slices"
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

// A checkout with nothing in hand is refused before any system is asked, and the
// refusal names the verb that only creates.
func TestCarryRefusesACleanCheckout(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, "")

	r := s.run("carry", "bd-1")

	r.came(t, result{Code: 1, Errored: []string{
		"this checkout carries no changes; work add bd-1 makes the worktree and carries nothing"}})
}

// A place that has a worktree already is nowhere to carry to, and the refusal
// costs the checkout nothing.
func TestCarryRefusesAnIdentifierThatAlreadyHasAWorktree(t *testing.T) {
	s := repository(t)
	s.dirty()
	path := s.opened("scratch")
	was := carrying(t, s.Repo)

	r := s.run("carry", "scratch")

	r.came(t, result{Code: 1, Errored: []string{"scratch already has a worktree; enter it with work switch scratch"}})
	require.Equal(t, was, carrying(t, s.Repo), "the checkout no longer carries what it was carrying")
	require.Empty(t, carrying(t, path), "something was carried into the worktree that was already there")
}

// The identifier is named on the command line: no picker stands in for it, and a
// tab press after the verb offers nothing either.
func TestCarryPutsUpNoListing(t *testing.T) {
	put := putsUp(t)
	s := repository(t, put.stub())
	s.dirty()

	r := s.run("carry")

	r.came(t, result{Code: 1}, apart)
	require.Empty(t, rows(s.completes("carry", "")), "a tab press after carry offered rows")
}

// The set of flags is add's: the open-on flag wins over what action.create names,
// and an action is called off for the invocation.
func TestCarryTakesTheFlagsAddTakes(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, claudeOn+"\n[action]\ncreate = \"claude\"\n")
	s.dirty()
	path := s.at("bd-1")

	r := s.run("carry", "bd-1", "--shell", "--no-claim")

	r.came(t, result{Answered: path, Asked: []string{listed, vetted, creates(path, "bd-1-do-a-thing")}})
	require.Empty(t, carrying(t, s.Repo), "the checkout was left carrying something")
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

// dirtyTracker stands a shell in a dirty checkout whose tracker answers one of
// its questions otherwise, which is what a case turning on the carry needs.
func dirtyTracker(t *testing.T, otherwise testenv.Reply) *session {
	t.Helper()
	bd := tracker(tickets(doable), tickets(doable))
	for i, reply := range bd.Replies {
		if slices.Equal(reply.To, otherwise.To) {
			bd.Replies[i] = otherwise
		}
	}
	s := repository(t, bd)
	s.settings(trackerOn)
	s.dirty()
	return s
}

// A restore that will not apply loses nothing: the entry stays in the stash, and
// the refusal names the worktree git put part of the changes in.
func TestACarryThatCannotFinishNamesWhereTheChangesAre(t *testing.T) {
	// The file the restore is about to collide with, written where only the creation
	// can reach: between git making the worktree and work restoring into it.
	s := dirtyTracker(t, testenv.Reply{To: []string{"worktree", "create"},
		Shell: `git worktree add -q "$3" -b "$5"` + "\n" + `printf 'something else entirely' > "$3/untracked"`})
	path := s.at("bd-1")

	r := s.run("carry", "bd-1")

	r.came(t, result{Code: 1, Asked: worked("bd-1", path, "bd-1-do-a-thing")}, apart)
	r.saying(t, "stashed", path)
	require.Contains(t, testenv.Git(t, s.Repo, "stash", "list"), "stash@{0}", "the entry was not kept")
	require.NotEmpty(t, carrying(t, path),
		"the worktree carries nothing; want the part git did put there, which the refusal names")
}

// The checkout is left carrying what it was carrying where an action refuses:
// nothing is moved out of it for a worktree the run does not go through with.
func TestNothingIsCarriedForACreationTheActionsRefuse(t *testing.T) {
	s := dirtyTracker(t, testenv.Reply{To: []string{"update"}, Grumbles: "the ticket is locked", Exits: 1})
	was := carrying(t, s.Repo)
	path := s.at("bd-1")

	r := s.run("carry", "bd-1")

	r.came(t, result{Code: 1, Errored: []string{claims("bd-1") + ": the ticket is locked"},
		Asked: worked("bd-1", path, "bd-1-do-a-thing")})
	require.Equal(t, was, carrying(t, s.Repo), "the checkout no longer carries what it was carrying")
	require.Empty(t, testenv.Git(t, s.Repo, "stash", "list"), "the changes were stashed for a run that refused")
}
