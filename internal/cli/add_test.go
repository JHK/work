package cli

import (
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// The ticket a worktree was just made for is claimed, and nothing on the command
// line calls that off: docs/references/tickets.md. What else the tracker is
// asked is work go's own case.
func TestAddClaimsTheTicketItMadeAWorktreeFor(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, nil, "")
	path := s.at("bd-1")

	r := s.run("add", "bd-1")

	r.came(t, result{Answered: path, Asked: worked("bd-1", path, "bd-1-do-a-thing")})
	require.True(t, s.hasBranch("bd-1-do-a-thing"), "the worktree is not on the branch the ticket names")
}

// With no identifier the picker stands in for one, over what has no worktree
// yet: the forge put the row up, and the pull request already worked on is not
// among the rows.
func TestAddWithNoIdentifierTakesThePickersRow(t *testing.T) {
	s := reviewing(t, nil, "",
		testenv.Stub{Name: "fzf", Says: "0\tpr-7\n"},
		testenv.Stub{Name: "gh", Replies: []testenv.Reply{
			{To: []string{"list"}, Says: `[{"number":7,"title":"Review this"},{"number":8,"title":"Already open"}]`},
		}})
	// The row fzf answers with is an index, so the one worktree open costs the
	// listing its row rather than shifting it.
	s.opened("pr-8")

	r := s.run("add")

	r.came(t, result{Answered: s.at("pr-7"), Asked: []string{pullRequests(s.Origin), putUp}})
	require.True(t, s.hasBranch("pr-7"), "the pull request's branch is not there")
}

// What cannot be made is refused before anything is on disk: an identifier that
// already has a worktree, and a branch or a directory already in the way.
func TestAddRefuses(t *testing.T) {
	tests := []struct {
		name string
		// tracked wires the tracker, for a row whose identifier is a ticket.
		tracked bool
		id      string
		// set prepares the repository and says what the refusal reads.
		set func(s *session) string
	}{
		{"an identifier that already has a worktree", true, "bd-1", func(s *session) string {
			s.openedOn("bd-1", "bd-1-do-a-thing")
			return "bd-1 already has a worktree; enter it with work switch bd-1"
		}},
		{"a name that already has one", false, "scratch", func(s *session) string {
			s.opened("scratch")
			return "scratch already has a worktree; enter it with work switch scratch"
		}},
		{"a branch already there", false, "other", func(s *session) string {
			testenv.Git(s.t, s.Repo, "branch", "other")
			return "branch other already exists; enter its worktree with work other"
		}},
		{"a directory already where the worktree would go", false, "blocked", func(s *session) string {
			testenv.Write(s.t, s.at("blocked/kept"), "something")
			return s.at("blocked") + " is already there; take that directory away first"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			var asked []string
			if tt.tracked {
				s = tracking(t, []ticket{doable}, []ticket{doable}, nil, "")
				asked = []string{listed}
			}
			said := tt.set(s)

			r := s.run("add", tt.id)

			r.came(t, result{Code: 1, Errored: []string{said}, Asked: asked})
		})
	}
}

// The name becomes a directory of its own and an argument to git, so a spelling
// no worktree could carry is refused: docs/references/cli.md#identifiers.
func TestAddRefusesANameNoWorktreeCouldCarry(t *testing.T) {
	for _, id := range []string{"..", "../etc", "a/b", "docs/pull/99-notes.md"} {
		t.Run(id, func(t *testing.T) {
			s := repository(t)

			r := s.run("add", id)

			r.came(t, result{Code: 1, Errored: []string{`"` + id + `" is not a usable worktree name`}})
		})
	}
}

// The command a fresh worktree opens on is rendered from what the resolver that
// answered supplied, which for a ticket is its id and its title.
func TestAClaudeSessionIsOpenedOnWhatTheWorktreeWasMadeFor(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, []string{"claude"}, "", testenv.Stub{Name: "claude"})

	r := s.hands("add", "bd-1")

	r.came(t, result{Asked: append(worked("bd-1", s.at("bd-1"), "bd-1-do-a-thing"),
		ticketSessionOn("bd-1", "Do a thing"))})
}

// An action that throws a refusal away rather than handing it on says it on the
// log the run carries, which is the only way it reaches the reader. Granting the
// trust is best effort, so the worktree is still made: internal/action/mise.
func TestAnActionSaysWhatItThrewAway(t *testing.T) {
	s := repository(t)
	s.settings(on("mise"))

	r := s.run("add", "scratch")

	// Warned, so no reader has to know --log-level to hear it.
	r.came(t, result{Answered: s.at("scratch"), Warned: []string{"mise trust: exit status 1"},
		Asked: []string{"mise trust"}})
	require.True(t, s.hasBranch("scratch"), "the worktree was refused over a grant that only means the session prompts")
}

// Falling through is for a key whose values nothing supplied. A key that renders
// nothing to run is a misconfiguration, refused rather than fallen past.
func TestAKeyThatRendersNoCommandRefusesRatherThanFallingThrough(t *testing.T) {
	// Every value the key has is supplied, and the one element it holds still
	// renders empty for a ticket carrying no title.
	untitled := with(doable, func(b *ticket) { b.Title = "" })
	s := tracking(t, []ticket{untitled}, []ticket{untitled},
		[]string{"claude"}, claudeTable+"start-ticket = [\"{{.Title}}\"]\n")

	r := s.run("add", "bd-1")

	r.came(t, result{Code: 1, Asked: worked("bd-1", s.at("bd-1"), "bd-1")}, apart)
	require.Contains(t, r.Errored[0], "claude.start-ticket", "the refusal does not name the key work could not render")
}

// The claim is the tracker's own places alone: a pull request the forge answered
// for is another system's, and bd is left out of it.
func TestAPlaceAnotherSystemAnsweredForIsNotClaimed(t *testing.T) {
	s := reviewing(t, []string{"beads"}, "", testenv.Stub{Name: "bd", Replies: []testenv.Reply{{To: []string{"list"}, Says: "[]"}}})

	r := s.run("add", "7")

	r.came(t, result{Answered: s.at("pr-7"), Asked: []string{listed}})
}
