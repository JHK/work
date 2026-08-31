package cli

import (
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// A worktree already open is reached under the name it is listed by, whatever a
// system would make of that name: docs/references/cli.md#identifiers.
func TestGoEntersTheWorktreeAnIdentifierAlreadyHas(t *testing.T) {
	tests := []struct {
		name string
		// dir is the worktree's directory, empty for the main checkout, and branch
		// what it has checked out.
		dir, branch string
		id          string
		forge       bool // the settings put the forge in the chain
	}{
		{"a worktree of its own", "scratch", "scratch", "scratch", false},
		{"a branch spelled with a separator", "login", "feature/x", "feature/x", false},
		{"the main checkout", "", "main", "main", false},
		// A verb wins the first position, so this name is only reachable through go:
		// R1 of docs/rules/command-grammar.md.
		{"a name a verb wins the first position with", "add", "add", "add", false},
		// A bare number is the forge's by its spelling alone, and the worktree takes it
		// all the same.
		{"a name a system would read as its own", "1234", "1234", "1234", true},
		// The number reaches the worktree open on the branch the forge names for it,
		// which is the name that worktree goes by.
		{"a pull request by its number", "pr-7", "pr-7", "7", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			if tt.forge {
				s.settings(systemsOn("github"))
			}
			path := s.Repo
			if tt.dir != "" {
				path = s.openedOn(tt.dir, tt.branch)
			}

			r := s.run("go", tt.id)

			r.came(t, result{Answered: path})
		})
	}
}

func TestTheBareFormReachesWhatGoReaches(t *testing.T) {
	s := repository(t)
	path := s.opened("scratch")

	r := s.run("scratch")

	r.came(t, result{Answered: path})
}

// A ticket with no worktree is vetted, made, claimed and handed over, in that
// order: one run, one line per question the tracker was put.
func TestGoMakesTheWorktreeATicketHasNone(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, nil, "")
	path := s.at("bd-1")

	r := s.run("go", "bd-1")

	r.came(t, result{Answered: path, Asked: worked("bd-1", path, "bd-1-do-a-thing")})
	require.DirExists(t, path, "the worktree is not on disk")
	// The branch is the id and a slug of the title, which is docs/references/configuration.md#branch-patterns.
	require.True(t, s.hasBranch("bd-1-do-a-thing"), "the worktree is not on the branch the ticket names")
}

// Over the worktrees open and whatever has none; the row it comes back with is
// the one reached.
func TestGoWithNoIdentifierTakesThePickersRow(t *testing.T) {
	// The row index is fzf's answer, and the worktree the shell stands in is not
	// among the rows: the main checkout is where this one stands.
	s := repository(t, testenv.Stub{Name: "fzf", Says: "0\tscratch\n"})
	path := s.opened("scratch")

	r := s.run("go")

	r.came(t, result{Answered: path, Asked: []string{putUp}})
}

// An identifier no system in the chain answers for is refused before anything is
// made, and names the verb that would make a worktree of it only where add would
// take it.
func TestGoRefusesWhatNothingInTheChainAnswersFor(t *testing.T) {
	tests := []struct {
		name     string
		settings string
		id       string
		said     string
	}{
		{
			"a name nothing answers for", "", "typo",
			`nothing answers for "typo"; work add typo makes a worktree of it`,
		},
		{
			// Wired or not, no system reads a path into a place, and none may: the name is
			// about to become a directory of its own, so add is not offered.
			"a name no worktree could carry, the forge wired", systemsOn("github"), "a/b",
			`nothing answers for "a/b"`,
		},
		{
			// The spelling the forge would have read, where nothing is wired to read it.
			"a pull request URL nothing wired answers for", "", "https://host/owner/repo/pull/7",
			`nothing answers for "https://host/owner/repo/pull/7"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			s.settings(tt.settings)

			r := s.run("go", tt.id)

			r.came(t, result{Code: 1, Errored: []string{tt.said}})
		})
	}
}

// Where several worktrees answer for one ticket, the shortest branch takes it,
// never git's listing order: docs/references/cli.md#identifiers.
func TestGoTakesTheShortestOfTheBranchesATicketOwns(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, nil, "")
	// The longer branch is made first, so git reports it ahead of the one wanted.
	s.openedOn("a", "bd-1-do-a-thing-again")
	shortest := s.openedOn("z", "bd-1-do")

	r := s.run("go", "bd-1")

	r.came(t, result{Answered: shortest, Asked: []string{listed}})
}

// A detached worktree goes by its directory rather than by a branch, so a name it
// shares with a branch is that branch's.
func TestGoTakesTheBranchAheadOfADetachedWorktreeOfThatName(t *testing.T) {
	s := repository(t)
	held := s.openedOn("held", "spike")
	s.detached("spike")

	r := s.run("go", "spike")

	r.came(t, result{Answered: held})
}

// A system that recognises the identifier and cannot answer for it stops the
// run, and what it was put is what the refusal carries: R5 of docs/rules/refusals.md.
func TestGoSaysWhatTheTrackerWouldNotAnswer(t *testing.T) {
	s := repository(t)
	s.settings(systemsOn("beads"))

	r := s.run("go", "bd-1")

	r.came(t, result{Code: 1, Errored: []string{listed + ": exit status 1"}, Asked: []string{listed}})
}
