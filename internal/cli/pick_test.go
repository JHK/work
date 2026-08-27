package cli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// A row states what to retype and what kind of thing it is, under the mark its
// resolver draws, and the names line up whether or not a worktree exists.
func TestTheRowsLineUpUnderTheirMarks(t *testing.T) {
	put := putsUp(t)
	longer := with(doable, func(b *ticket) { b.ID, b.Title = "bd-longer", "Other" })
	// A ticket bd could not put a title to, whose name is the longest of the lot and
	// sets no column all the same.
	untitled := with(doable, func(b *ticket) { b.ID, b.Title = "bd-untitled-and-longest", "" })
	s := tracking(t, []ticket{longer, doable, untitled}, []ticket{longer, doable, untitled}, []string{"github"}, "",
		put.stub(), testenv.Stub{Name: "gh", Replies: []testenv.Reply{
			// A pull request gh could not put a title to either.
			{To: []string{"list"}, Says: `[{"number":7,"title":"Review this"},{"number":9,"title":""}]`},
		}})
	testenv.Git(t, s.Repo, "remote", "add", "origin", hosted)
	s.openedOn("worked", "bd-longer-other")
	s.openedOn("review", "pr-7")
	// A plain worktree is named by its branch and says nothing more.
	s.opened("spike")

	r := s.run("go")

	r.came(t, result{Code: 1, Asked: []string{listed, vetted, pullRequests(hosted), putUp}}, atOnce)
	// git lists a repository's worktrees by directory name, so the rows are review,
	// spike and worked, then what each resolver offers behind them.
	want := []string{
		highlight + "⎇ ⇄ pr-7     " + reset + "  ·  Review this",
		highlight + "⎇ ◇ spike" + reset,
		highlight + "⎇ ◆ bd-longer" + reset + "  ·  Other",
		"  ⇄ pr-9",
		"  ◆ bd-1       ·  Do a thing",
		"  ◆ bd-untitled-and-longest",
	}
	testenv.Equal(t, want, put.rows(), "the rows the picker offers are not what a reader lines up")
}

// A listing left with no rows is refused in one line naming what there is
// nothing of, rather than an empty screen to dismiss: fzf is never reached.
func TestAnEmptyListingIsRefusedRatherThanPutUp(t *testing.T) {
	// The main checkout alone, standing in it: these four each leave out what they
	// cannot act on, so each is empty. go has the main checkout to offer, so it is
	// not among them.
	for verb, said := range map[string]string{
		"switch": "no worktree to switch to",
		"remove": "no worktree to remove",
		"move":   "no worktree to move",
		"add":    "nothing left to add",
	} {
		t.Run(verb, func(t *testing.T) {
			s := repository(t)

			r := s.run(verb)

			r.came(t, result{Code: 1, Errored: []string{said}})
		})
	}
}

// asks is the prompt as a stand-in records being put up: the same fzf over
// nothing on offer, with the answer already typed into the query.
func asks(preset string) string {
	return "fzf --height 40% --reverse --prompt " + prompt + " --print-query --query " + preset
}

// A move given no destination asks for one with the current name already in it,
// so the answer is the name edited rather than retyped: the prompt hands fzf the
// query and takes back what was typed over it.
func TestMoveAsksForTheDestination(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "fzf", Says: "settled\n", Exits: 1})
	from := s.opened("scratch")
	to := s.at("settled")

	r := s.run("move", "scratch")

	r.came(t, result{Asked: []string{asks("scratch")},
		Out: "moved worktree " + from + " to " + to + "\nrenamed branch scratch to settled\n"})
	require.DirExists(t, to, "the worktree did not land where the answer named")
}

// A branch may be spelled with separators in it, so the question carries the
// directory: an answer left as it stands is a bare name and not a path.
func TestMoveAsksWithTheDirectoryNotTheBranch(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "fzf", Says: "login\n", Exits: 1})
	// The branch is not what the directory is called, which is what the case turns on.
	at := s.at("login")
	testenv.Git(t, s.Repo, "worktree", "add", "-b", "feature/login", at)

	r := s.run("move", "feature/login")

	r.came(t, result{Code: 1, Asked: []string{asks("login")},
		Errored: []string{at + " is where feature/login already sits"}})
}

// An answer nobody gave is nothing to act on: an interruption and a query
// emptied out each end the run saying nothing and leaving the worktree be.
func TestMoveWithoutAnAnswerToThePrompt(t *testing.T) {
	tests := []struct {
		name string
		fzf  testenv.Stub
	}{
		{"interrupted", testenv.Stub{Name: "fzf", Exits: 130}},
		{"emptied out", testenv.Stub{Name: "fzf", Says: "\n", Exits: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t, tt.fzf)
			from := s.opened("scratch")

			r := s.run("move", "scratch")

			r.came(t, result{Code: 1, Asked: []string{asks("scratch")}})
			require.DirExists(t, from, "the worktree moved on an answer nobody gave")
		})
	}
}

// A prompt that would not run at all is one the user has to be told about, a
// missing binary above all, rather than an answer declined.
func TestMoveWhenThePromptWillNotRun(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "fzf", Says: "settled\n", Exits: 2})
	from := s.opened("scratch")

	r := s.run("move", "scratch")

	r.came(t, result{Code: 1, Errored: []string{"fzf: exit status 2"}, Asked: []string{asks("scratch")}})
	require.DirExists(t, from, "the worktree moved on a prompt that never ran")
}

// What a candidate is refused over is known from the candidate alone, so the
// refusal lands with a destination and without, and the question is never put.
// None of the three names a verb that would make what move cannot act on.
func TestMoveRefusesBeforeAsking(t *testing.T) {
	tests := []struct {
		name, settings, target, said string
		stoodIn                      bool
	}{
		{"the worktree stood in", "", "scratch",
			"scratch is the worktree you are standing in; run work move from outside it", true},
		// A bare number is the forge's by its spelling alone, so the place is made
		// without gh being asked for it.
		{"a place with no worktree open", on("github"), "7", "pr-7 has no worktree to move", false},
		{"a name nothing answers for", "", "typo", `nothing answers for "typo"`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			s.settings(tt.settings)
			path := s.opened("scratch")
			if tt.stoodIn {
				s.Dir = path
			}

			// A destination given and one left for the prompt alike: neither is what the
			// refusal waits on.
			for _, args := range [][]string{{"move", tt.target}, {"move", tt.target, "settled"}} {
				r := s.run(args...)

				r.came(t, result{Code: 1, Errored: []string{tt.said}})
			}
		})
	}
}

// A system the picker's listing could not reach is said on the way past, one
// line each, so a listing short of its rows does not read as a repository with
// none.
func TestThePickerSaysEverySystemItsListingCouldNotReach(t *testing.T) {
	s := repository(t)
	// A remote is what the forge reads the repository off, so both systems are asked
	// and both stand-ins fail. It carries the suffix a clone URL has, which the forge
	// is asked with as it stands.
	remote := hosted + ".git"
	testenv.Git(t, s.Repo, "remote", "add", "origin", remote)
	s.settings(on("beads", "github"))
	s.opened("scratch")

	// fzf stands in failing, so the run ends cancelled with nothing else said: what
	// work said is the listing's own.
	r := s.run("go")

	r.came(t, result{Code: 1,
		Warned: []string{pullRequests(remote) + ": exit status 1", vetted + ": exit status 1"},
		Asked:  []string{listed, vetted, pullRequests(remote), putUp}}, atOnce)
}

// screened is what each verb's screen was put up with, in an order git's own has
// no say in.
func screened(s *session, put *screen) map[string][]string {
	t := s.t
	t.Helper()
	out := map[string][]string{}
	for _, verb := range []string{"go", "switch", "add", "remove", "move"} {
		r := s.run(verb)

		// Dismissed, so the rows are all that comes of the screen and no verb goes on
		// to act on one.
		r.came(t, result{Code: 1}, besides("Asked"))
		out[verb] = slices.Sorted(slices.Values(retyped(put.rows())))
	}
	return out
}

// Reaching and entering leave out the worktree stood in, removing and moving
// the main checkout too.
func TestEachVerbOffersWhatItCanActOn(t *testing.T) {
	tests := []struct {
		name string
		// from is where the shell stands, of the session and the worktree outside it.
		from                 func(s *session, away string) string
		reach, enter, remove []string
	}{
		{
			"the main checkout",
			func(s *session, _ string) string { return s.Repo },
			[]string{"away", "one", "spare", "two"}, []string{"away", "one", "two"}, []string{"away", "one", "two"},
		},
		{
			"a directory under it",
			func(s *session, _ string) string { return filepath.Join(s.Repo, "under") },
			[]string{"away", "one", "spare", "two"}, []string{"away", "one", "two"}, []string{"away", "one", "two"},
		},
		{
			"a worktree",
			func(s *session, _ string) string { return s.at("one") },
			[]string{"away", "spare", "trunk", "two"}, []string{"away", "trunk", "two"}, []string{"away", "two"},
		},
		{
			"a directory under a worktree",
			func(s *session, _ string) string { return filepath.Join(s.at("two"), "under") },
			[]string{"away", "one", "spare", "trunk"}, []string{"away", "one", "trunk"}, []string{"away", "one"},
		},
		{
			"a worktree outside the repository",
			func(_ *session, away string) string { return away },
			[]string{"one", "spare", "trunk", "two"}, []string{"one", "trunk", "two"}, []string{"one", "two"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			put := putsUp(t)
			// The tracker offers a place with no worktree yet, which is the row add has of
			// its own.
			spare := with(doable, func(b *ticket) { b.ID = "spare" })
			s := tracking(t, []ticket{spare}, []ticket{spare}, nil, "", put.stub())
			testenv.Git(t, s.Repo, "branch", "--move", "trunk")
			s.opened("one")
			s.opened("two")
			// git reports a worktree outside the repository like any other. Standing in one,
			// nothing holds the main checkout, so the listing is the only thing leaving it out.
			away := filepath.Join(t.TempDir(), "away")
			testenv.Git(t, s.Repo, "worktree", "add", "-b", "away", away)
			s.Dir = tt.from(s, away)
			require.NoError(t, os.MkdirAll(s.Dir, 0o755), "the directory the shell stands in")

			// Moving offers what removing offers, and adding the one place no worktree
			// stands for.
			want := map[string][]string{"go": tt.reach, "switch": tt.enter,
				"remove": tt.remove, "move": tt.remove, "add": {"spare"}}
			testenv.Equal(t, want, screened(s, put), "what the verbs offer from "+tt.name+" is not what they can act on")
		})
	}
}

// A worktree nested in another goes with the tree above it, so standing in the
// nested one leaves both out: the listing and the refusal are one rule.
func TestTheWorktreeAboveTheOneStoodInIsNeitherOfferedNorRemoved(t *testing.T) {
	put := putsUp(t)
	s := repository(t, put.stub())
	one := s.opened("one")
	nested := filepath.Join(one, "nested")
	testenv.Git(t, s.Repo, "worktree", "add", "-b", "nested", nested)
	s.opened("two")
	s.Dir = nested

	r := s.run("remove")

	r.came(t, result{Code: 1, Asked: []string{putUp}})
	testenv.Equal(t, []string{"two"}, retyped(put.rows()), "removing offered a worktree it would refuse")

	refused := s.run("remove", "one")

	refused.came(t, result{Code: 1, Errored: []string{"one is the worktree you are standing in; run work remove from outside it"}})
}

// What is worth working on is every worktree git knows, the main checkout at
// their head, then the offers with none: one place counted once however found.
func TestTheRowsAreTheWorktreesThenWhatHasNoneYet(t *testing.T) {
	put := putsUp(t)
	other := with(doable, func(b *ticket) { b.ID, b.Title = "bd-2", "Another thing" })
	s := tracking(t, []ticket{doable, other}, []ticket{doable, other}, nil, "", put.stub())
	// The ticket's own worktree, a worktree no system answers for, and the one the
	// shell stands in so that the main checkout is a row.
	s.openedOn("worked", "bd-1-do-a-thing")
	s.opened("loose")
	s.Dir = s.opened("spike")

	r := s.run("go")

	r.came(t, result{Code: 1, Asked: []string{listed, vetted, putUp}}, atOnce)
	rows := put.rows()
	names := retyped(rows)
	require.Equal(t, "main", names[0], "the first row is not the main checkout git reports first")
	testenv.Equal(t, []string{"bd-1", "loose", "main"}, slices.Sorted(slices.Values(names[:3])),
		"the open rows are not what is ahead of what is merely offered")
	for _, row := range rows[:3] {
		require.Contains(t, row, openMark, "row %q is not marked open; want the worktrees ahead of the offers", row)
	}
	// The ticket has a worktree, so the tracker's offer of it is not a second row,
	// and the one place with none is what is left.
	testenv.Equal(t, []string{"bd-2"}, retyped(rows[3:]), "the offered rows are not the places with no worktree yet")
	require.NotContains(t, rows[3], openMark, "the offered row came back with a worktree")
}

// Naming a worktree asks as little as it can, so a system reading a branch alone
// may have no title for it: its own offer beside it is what completes the row.
func TestAnOpenRowTakesTheTitleFromItsOffer(t *testing.T) {
	put := putsUp(t)
	s := reviewing(t, nil, "", put.stub(), testenv.Stub{Name: "gh", Replies: []testenv.Reply{
		{To: []string{"list"}, Says: `[{"number":7,"title":"Review this"}]`},
	}})
	s.openedOn("review", "pr-7")

	r := s.run("go")

	r.came(t, result{Code: 1, Asked: []string{pullRequests(s.Origin), putUp}}, atOnce)
	rows := put.rows()
	testenv.Equal(t, []string{"pr-7"}, retyped(rows), "the pull request was counted other than once, open and offered")
	require.Regexp(t, `pr-7\s+·\s+Review this`, plain(rows[0]), "the row did not take the title its own offer had")
}

// Only the system that answered for the worktree completes its row: another's
// offer of the same name is another place that happens to be spelled alike.
func TestAnOpenRowTakesNoTitleFromAnotherSystemsOffer(t *testing.T) {
	put := putsUp(t)
	// A ticket spelled as the forge names a pull request, which is the one way two
	// systems put one name up.
	alike := with(doable, func(b *ticket) { b.ID, b.Title = "pr-7", "Another place spelled alike" })
	// No origin, so the forge names the worktree off its branch and has nothing to
	// offer, as gh that is absent or unauthenticated has nothing to offer.
	s := tracking(t, []ticket{alike}, []ticket{alike}, []string{"github"}, "", put.stub())
	s.openedOn("review", "pr-7")

	r := s.run("go")

	r.came(t, result{Code: 1, Asked: []string{listed, vetted, putUp}}, atOnce)
	rows := put.rows()
	testenv.Equal(t, []string{"pr-7"}, retyped(rows), "the place was counted other than once")
	require.NotContains(t, rows[0], "·", "the row took another system's title")
}

// An offer the core could not make a directory for could be shown but never
// retyped, so it is left off.
func TestAnOfferNoWorktreeCouldBeMadeForIsLeftOff(t *testing.T) {
	put := putsUp(t)
	unusable := with(doable, func(b *ticket) { b.ID = ".." })
	s := tracking(t, []ticket{doable, unusable}, []ticket{doable, unusable}, nil, "", put.stub())

	r := s.run("add")

	r.came(t, result{Code: 1, Asked: []string{listed, vetted, putUp}}, atOnce)
	testenv.Equal(t, []string{"bd-1"}, retyped(put.rows()), "a name no worktree could be made for was offered")
}
