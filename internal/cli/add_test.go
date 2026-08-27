package cli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// The ticket a worktree was just made for is claimed, unless the flag that calls
// the claim off was given. What else the tracker is asked is work go's own case.
func TestAddClaimsTheTicketItMadeAWorktreeFor(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		claimed bool
	}{
		{"claimed", []string{"add", "bd-1"}, true},
		{"the claim called off", []string{"add", "bd-1", "--no-claim"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tracking(t, []ticket{doable}, []ticket{doable}, "")
			path := s.at("bd-1")
			asked := []string{listed, vetted, creates(path, "bd-1-do-a-thing")}
			if tt.claimed {
				asked = append(asked, claims("bd-1"))
			}

			r := s.run(tt.args...)

			r.came(t, result{Answered: path, Asked: asked})
			require.True(t, s.hasBranch("bd-1-do-a-thing"), "the worktree is not on the branch the ticket names")
		})
	}
}

// With no identifier the picker stands in for one, over what has no worktree
// yet: the forge put the row up, and the pull request already worked on is not
// among the rows.
func TestAddWithNoIdentifierTakesThePickersRow(t *testing.T) {
	s := reviewing(t, "",
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
				s = tracking(t, []ticket{doable}, []ticket{doable}, "")
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

// Which key names the action is the moment's own: action.create for the worktree
// just made, action.enter for the same worktree entered again.
func TestTheMomentDecidesWhatAWorktreeOpensOn(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "claude"})
	s.settings(claudeOn + "\n[action]\ncreate = \"claude\"\nenter = \"shell\"\n")

	made := s.hands("add", "scratch")

	made.came(t, result{Asked: []string{"claude --permission-mode auto --name=scratch"}})

	entered := s.run("switch", "scratch")

	entered.came(t, result{Answered: s.at("scratch")})
}

func TestAnOpenOnFlagWinsOverTheKey(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "claude"})
	s.settings(claudeOn + "\n[action]\ncreate = \"claude\"\n")

	r := s.run("add", "scratch", "--shell")

	r.came(t, result{Answered: s.at("scratch")})
}

// The command a fresh worktree opens on is rendered from what the resolver that
// answered supplied, which for a ticket is its id and its title.
func TestAClaudeSessionIsOpenedOnWhatTheWorktreeWasMadeFor(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, claudeOn, testenv.Stub{Name: "claude"})

	r := s.hands("add", "bd-1", "--claude")

	r.came(t, result{Asked: append(worked("bd-1", s.at("bd-1"), "bd-1-do-a-thing"),
		"claude --permission-mode auto --name=bd-1: Do a thing /start bd-1")})
}

// An action that throws a refusal away rather than handing it on says it on the
// log the run carries, which is the only way it reaches the reader. Granting the
// trust is best effort, so the worktree is still made: internal/action/mise.
func TestAnActionSaysWhatItThrewAway(t *testing.T) {
	s := repository(t)
	s.settings(miseOn)

	r := s.run("add", "scratch")

	// Warned, so no reader has to know --log-level to hear it.
	r.came(t, result{Answered: s.at("scratch"), Warned: []string{"mise trust: exit status 1"},
		Asked: []string{"mise trust"}})
	require.True(t, s.hasBranch("scratch"), "the worktree was refused over a grant that only means the session prompts")
}

// dirty puts one of each sort of change in the checkout the shell stands in:
// staged, unstaged, untracked and ignored. The worktree directory is ignored, as
// a repository does.
func (s *session) dirty() {
	s.t.Helper()
	testenv.Write(s.t, filepath.Join(s.Dir, "tracked"), "as committed")
	testenv.Write(s.t, filepath.Join(s.Dir, ".gitignore"), "ignored\n"+defaultDir+"/\n")
	testenv.Git(s.t, s.Dir, "add", "tracked", ".gitignore")
	testenv.Git(s.t, s.Dir, "commit", "-m", "a file to change")

	testenv.Write(s.t, filepath.Join(s.Dir, "staged"), "staged")
	testenv.Git(s.t, s.Dir, "add", "staged")
	testenv.Write(s.t, filepath.Join(s.Dir, "tracked"), "changed")
	testenv.Write(s.t, filepath.Join(s.Dir, "untracked"), "untracked")
	testenv.Write(s.t, filepath.Join(s.Dir, "ignored"), "ignored")
}

// carrying is what git reports the checkout at path is holding.
func carrying(t *testing.T, path string) string {
	t.Helper()
	return testenv.Git(t, path, "status", "--porcelain")
}

// Parking is the whole of the move: the only difference afterwards is which
// directory the changes are in, what was staged still staged.
// docs/references/cli.md#add.
func TestAddCarriesTheCheckoutsWorkIntoTheWorktree(t *testing.T) {
	s := repository(t)
	s.dirty()

	r := s.run("add", "carried")

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
func TestAddLeavesIgnoredFilesWhereTheyAre(t *testing.T) {
	s := repository(t)
	s.dirty()

	r := s.run("add", "carried")

	r.came(t, result{Answered: s.at("carried")})
	require.FileExists(t, filepath.Join(s.Repo, "ignored"), "the checkout's ignored file did not stay where it was")
	require.NoFileExists(t, filepath.Join(r.Answered, "ignored"), "the ignored file travelled into the worktree")
}

// A clean checkout has nothing to carry, so nothing another stash left behind is
// restored into the worktree. The directory work is about to make is not work in
// hand either.
func TestAddCarriesNothingOutOfACleanCheckout(t *testing.T) {
	s := repository(t)
	testenv.Write(t, filepath.Join(s.Repo, "tracked"), "as committed")
	testenv.Git(t, s.Repo, "add", "tracked")
	testenv.Git(t, s.Repo, "commit", "-m", "a file to change")
	// An entry of the user's own, from work the parking has no business restoring.
	testenv.Write(t, filepath.Join(s.Repo, "tracked"), "an older experiment")
	testenv.Git(t, s.Repo, "stash", "push")

	r := s.run("add", "carried")

	r.came(t, result{Answered: s.at("carried")})
	require.Contains(t, testenv.Git(t, s.Repo, "stash", "list"), "stash@{0}", "the entry that was already there is gone")
	require.Empty(t, carrying(t, r.Answered), "something was restored into the worktree")
}

// The verbs that only reach a worktree leave the checkout as they found it, so
// starting a ticket is not carrying what was in hand.
func TestOnlyAddCarriesTheCheckoutsWork(t *testing.T) {
	s := repository(t)
	s.dirty()
	path := s.opened("reached")
	was := carrying(t, s.Repo)

	r := s.run("switch", "reached")

	r.came(t, result{Answered: path})
	require.Equal(t, was, carrying(t, s.Repo), "the checkout no longer carries what it was carrying")
	require.Empty(t, carrying(t, r.Answered), "the worktree is more than a checkout of the branch")
}

// Falling through is for a key whose values nothing supplied. A key that renders
// nothing to run is a misconfiguration, refused rather than fallen past.
func TestAKeyThatRendersNoCommandRefusesRatherThanFallingThrough(t *testing.T) {
	// Every value the key has is supplied, and the one element it holds still
	// renders empty for a ticket carrying no title.
	untitled := with(doable, func(b *ticket) { b.Title = "" })
	s := tracking(t, []ticket{untitled}, []ticket{untitled},
		claudeOn+"start-ticket = [\"{{.Title}}\"]\n")

	r := s.run("add", "bd-1", "--claude")

	r.came(t, result{Code: 1, Asked: worked("bd-1", s.at("bd-1"), "bd-1")}, apart)
	require.Contains(t, r.Errored[0], "claude.start-ticket", "the refusal does not name the key work could not render")
}

// The claim is the tracker's own places alone: a pull request the forge answered
// for is another system's, and bd is left out of it.
func TestAPlaceAnotherSystemAnsweredForIsNotClaimed(t *testing.T) {
	s := reviewing(t, trackerOn, testenv.Stub{Name: "bd", Replies: []testenv.Reply{{To: []string{"list"}, Says: "[]"}}})

	r := s.run("add", "7")

	r.came(t, result{Answered: s.at("pr-7"), Asked: []string{listed}})
}

// parking stands a shell in a dirty checkout whose tracker answers one of its
// questions otherwise, which is what a case turning on the parking needs.
func parking(t *testing.T, otherwise testenv.Reply) *session {
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
func TestAParkingThatCannotFinishNamesWhereTheChangesAre(t *testing.T) {
	// The file the restore is about to collide with, written where only the creation
	// can reach: between git making the worktree and work restoring into it.
	s := parking(t, testenv.Reply{To: []string{"worktree", "create"},
		Shell: `git worktree add -q "$3" -b "$5"` + "\n" + `printf 'something else entirely' > "$3/untracked"`})
	path := s.at("bd-1")

	r := s.run("add", "bd-1")

	r.came(t, result{Code: 1, Asked: worked("bd-1", path, "bd-1-do-a-thing")}, apart)
	r.saying(t, "stashed", path)
	require.Contains(t, testenv.Git(t, s.Repo, "stash", "list"), "stash@{0}", "the entry was not kept")
	require.NotEmpty(t, carrying(t, path),
		"the worktree carries nothing; want the part git did put there, which the refusal names")
}

// The checkout is left carrying what it was carrying where an action refuses:
// nothing is moved out of it for a worktree the run does not go through with.
func TestNothingIsParkedForACreationTheActionsRefuse(t *testing.T) {
	s := parking(t, testenv.Reply{To: []string{"update"}, Grumbles: "the ticket is locked", Exits: 1})
	was := carrying(t, s.Repo)
	path := s.at("bd-1")

	r := s.run("add", "bd-1")

	r.came(t, result{Code: 1, Errored: []string{claims("bd-1") + ": the ticket is locked"},
		Asked: worked("bd-1", path, "bd-1-do-a-thing")})
	require.Equal(t, was, carrying(t, s.Repo), "the checkout no longer carries what it was carrying")
	require.Empty(t, testenv.Git(t, s.Repo, "stash", "list"), "the changes were stashed for a run that refused")
}
