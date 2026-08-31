package cli

import (
	"cmp"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// A ticket that cannot be worked is refused in work's own words, and nothing the
// verdict cannot turn on is asked: docs/references/systems.md#vetting.
func TestATicketThatCannotBeWorkedIsRefused(t *testing.T) {
	epic := with(doable, func(b *ticket) { b.Type = "epic" })
	tests := []struct {
		name string
		bead ticket
		// ready is the tracker calling the ticket unblocked, said the refusal work
		// gives, empty where the ticket is worked, and asks the readiness query put.
		ready bool
		said  string
		asks  bool
	}{
		{"open and ready", doable, true, "", true},
		{"in progress, its dependencies unasked", with(doable, func(b *ticket) { b.Status = "in_progress" }), false, "", false},
		{"an open dependency", doable, false, "bd-1 is blocked by an open dependency", true},
		{"closed", with(doable, func(b *ticket) { b.Status = "closed" }), false, "bd-1 is already closed", false},
		{"unrefined", with(doable, func(b *ticket) { b.Status = "deferred" }), false, "bd-1 is unrefined; refine it first with /refine bd-1", false},
		{"a status nothing is worked from", with(doable, func(b *ticket) { b.Status = "blocked" }), false, "bd-1 is blocked, not workable", false},
		{"no acceptance criteria", with(doable, func(b *ticket) { b.Criteria = "  \n" }), false, "bd-1 has no acceptance criteria; refine it first with /refine bd-1", false},
		{"a status outranking the criteria", with(doable, func(b *ticket) { b.Status, b.Criteria = "blocked", "" }), false, "bd-1 is blocked, not workable", false},
		{"an epic asked for none of it", with(epic, func(b *ticket) { b.Status, b.Criteria = "deferred", "" }), false, "", false},
		{"a closed epic", with(epic, func(b *ticket) { b.Status = "closed" }), false, "bd-1 is already closed", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ready []ticket
			if tt.ready {
				ready = []ticket{tt.bead}
			}
			s := tracking(t, []ticket{tt.bead}, ready, nil, "")
			path := s.at("bd-1")
			asked := []string{listed}
			if tt.asks {
				asked = append(asked, vetted)
			}

			r := s.run("go", "bd-1")

			if tt.said == "" {
				r.came(t, result{Answered: path,
					Asked: append(asked, creates(path, "bd-1-do-a-thing"), claims("bd-1"))})
				require.DirExists(t, path, "the worktree the ticket is worked in is not on disk")
				return
			}
			r.came(t, result{Code: 1, Errored: []string{tt.said}, Asked: asked})
			require.NoDirExists(t, path, "the worktree was made despite the refusal")
		})
	}
}

// A tracker that will not answer is not a ticket that cannot be worked, so the
// refusal carries what work put to it: R5 of docs/rules/refusals.md.
func TestATrackerThatWillNotSayWhetherATicketIsReady(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "bd", Replies: []testenv.Reply{
		{To: []string{"list"}, Says: tickets(doable)},
		{To: []string{"ready"}, Grumbles: "the database is not there", Exits: 1},
	}})
	s.settings(systemsOn("beads"))

	r := s.run("go", "bd-1")

	r.came(t, result{Code: 1, Errored: []string{vetted + ": the database is not there"}, Asked: []string{listed, vetted}})
	require.NoDirExists(t, s.at("bd-1"), "the worktree was made despite the refusal")
}

// The branch a ticket's worktree checks out is the id and a slug of its title,
// cut where a title runs long and dropped where nothing slugs.
func TestTheBranchATicketsWorktreeChecksOut(t *testing.T) {
	tests := []struct{ name, title, branch string }{
		{"a title slugged", "Rate-limit /api/upload", "bd-1-rate-limit-api-upload"},
		{"a title past the limit", strings.Repeat("ab ", 30), "bd-1-" + strings.Repeat("ab-", 13) + "a"},
		{"a title nothing spells", "—", "bd-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bead := with(doable, func(b *ticket) { b.Title = tt.title })
			s := tracking(t, []ticket{bead}, []ticket{bead}, nil, "")
			path := s.at("bd-1")

			r := s.run("add", "bd-1")

			r.came(t, result{Answered: path, Asked: worked("bd-1", path, tt.branch)})
			require.True(t, s.hasBranch(tt.branch), "the ticket's worktree checks out none of the branches the repository has")
		})
	}
}

// patterned is the settings under which a ticket's branch is named by a pattern
// of the repository's own rather than by the compiled-in one.
const patterned = "[branch]\nticket = \"feature/{{.ID}}-{{.Slug}}\"\n"

func TestAConfiguredPatternNamesATicketsBranch(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, nil, patterned)
	path := s.at("bd-1")

	r := s.run("add", "bd-1")

	r.came(t, result{Answered: path, Asked: worked("bd-1", path, "feature/bd-1-do-a-thing")})
	require.True(t, s.hasBranch("feature/bd-1-do-a-thing"), "the configured pattern did not name the branch on disk")
}

// A worktree made under that pattern is its ticket's whatever the title said
// when it was made, the id alone being what the pattern recognises.
func TestAConfiguredPatternFindsATicketsWorktreeAgain(t *testing.T) {
	s := tracking(t, []ticket{doable}, nil, nil, patterned)
	older := s.openedOn("older", "feature/bd-1-an-older-title")

	r := s.run("switch", "bd-1")

	r.came(t, result{Answered: older, Asked: []string{listed}})
}

// One on a branch its id owns, the longest of the ids owning a branch taking
// it: docs/references/cli.md#identifiers.
func TestWhichWorktreeATicketReaches(t *testing.T) {
	known := []ticket{doable, with(doable, func(b *ticket) { b.ID, b.Title = "bd-1-2", "A ticket named after it" })}
	tests := []struct {
		name, branch, id string
		// said is the refusal where the worktree is not that ticket's.
		said string
	}{
		{"its own branch", "bd-1", "bd-1", ""},
		{"its branch under a slug", "bd-1-do-a-thing", "bd-1", ""},
		{"a title it has since been retitled from", "bd-1-an-older-title", "bd-1", ""},
		{"the longest id owning a branch takes it", "bd-1-2-a-slug", "bd-1-2", ""},
		{"a branch a longer id owns is not the shorter ticket's", "bd-1-2-a-slug", "bd-1", "bd-1 has no worktree open; work add bd-1 makes one"},
		{"a branch no ticket owns", "some-branch", "bd-1", "bd-1 has no worktree open; work add bd-1 makes one"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tracking(t, known, nil, nil, "")
			path := s.openedOn("wt", tt.branch)

			r := s.run("switch", tt.id)

			want := result{Answered: path, Asked: []string{listed}}
			if tt.said != "" {
				want.Answered, want.Code, want.Errored = "", 1, []string{tt.said}
			}
			r.came(t, want)
		})
	}
}

// A ticket's branch forks from the checkout the shell stands in rather than from
// the main checkout, the tracker taking no fork point of its own.
func TestATicketsBranchForksFromTheCheckoutTheShellStandsIn(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, nil, "")
	s.Dir = s.opened("scratch")
	testenv.Git(t, s.Dir, "commit", "--allow-empty", "-m", "not on main")
	path := s.at("bd-1")

	r := s.run("add", "bd-1")

	r.came(t, result{Answered: path, Asked: worked("bd-1", path, "bd-1-do-a-thing")})
	head := testenv.Git(t, s.Dir, "rev-parse", "HEAD")
	require.Equal(t, head, testenv.Git(t, r.Answered, "rev-parse", "HEAD"),
		"the ticket's branch forked from somewhere other than the checkout the shell stands in")
}

// A pull request is reached by its number, by its branch or by the URL of the
// page it is read on, however padded, and the forge is asked nothing.
func TestAPullRequestIsReachedHoweverItIsSpelled(t *testing.T) {
	tests := []struct {
		name, id string
		// branch is what the one worktree open has checked out, pr-7 where the case
		// names none, and said the refusal where that worktree is not the one asked for.
		branch, said string
	}{
		{"a bare number", "7", "", ""},
		{"a padded number", "007", "", ""},
		{"the branch its worktree checks out", "pr-7", "", ""},
		{"a padded branch", "pr-007", "", ""},
		{"a pull request URL", "https://github.com/o/r/pull/7", "", ""},
		{"a URL without its host", "o/r/pull/7", "", ""},
		{"a URL of a page under it", "https://github.com/o/r/pull/7/files", "", ""},
		{"another pull request's number", "70", "", "pr-70 has no worktree open; work add pr-70 makes one"},
		// A branch the pattern would have spelled otherwise is nobody's but the
		// worktree's own, so 7 does not reach it.
		{"a branch the pattern spells otherwise", "7", "pr-007", "pr-7 has no worktree open; work add pr-7 makes one"},
		{"a zero", "0", "", `"0" is not a pull request number`},
		{"a zero under the branch pattern", "pr-0", "", `"pr-0" is not a pull request number`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			s.settings(systemsOn("github"))
			path := s.openedOn("wt", cmp.Or(tt.branch, "pr-7"))

			r := s.run("switch", tt.id)

			want := result{Answered: path}
			if tt.said != "" {
				want.Answered, want.Code, want.Errored = "", 1, []string{tt.said}
			}
			r.came(t, want)
		})
	}
}

// The forge is asked ahead of the tracker, so a bare number is the pull request
// of that number rather than the ticket: docs/references/cli.md#identifiers.
func TestTheForgeIsAskedAheadOfTheTracker(t *testing.T) {
	// A ticket the tracker calls 7, which is the spelling the forge reads as its own.
	numbered := with(doable, func(b *ticket) { b.ID = "7" })
	s := reviewing(t, []string{"beads"}, "", tracker(tickets(numbered), "[]"))

	r := s.run("add", "7")

	// The tracker is asked what the checkout is, and never about the identifier:
	// had it answered for 7, the worktree would be the ticket's and claimed.
	r.came(t, result{Answered: s.at("pr-7"), Asked: []string{listed}})
}

// The last resolver takes whatever worktree is left, so one no system recognises
// is still drawn, under a mark of that resolver's own.
func TestTheLastResolverMarksTheWorktreeNoSystemRecognises(t *testing.T) {
	put := putsUp(t)
	// The tracker and the forge are both wired, so both decline before it is asked.
	s := tracking(t, nil, nil, []string{"github"}, "", put.dismisses())
	s.openedOn("held", "a-branch-no-system-names")

	r := s.run("switch")

	r.came(t, result{Code: 1, Asked: []string{listed, putUp}})
	require.Contains(t, plain(put.rows()[0]), "◇ a-branch-no-system-names",
		"the worktree's row carries no mark of the resolver that adopted it")
}

// A pull request's worktree checks out the head git fetches for it, which is the
// one question about it the forge is not asked.
func TestAPullRequestsWorktreeChecksOutTheHeadItFetches(t *testing.T) {
	s := reviewing(t, nil, "")

	r := s.run("add", "https://github.com/o/r/pull/7")

	r.came(t, result{Answered: s.at("pr-7")})
	require.True(t, s.hasBranch("pr-7"), "the pull request's branch is not there")
	require.Equal(t, reviewHead, testenv.Git(t, r.Answered, "log", "-1", "--format=%s"), "the worktree checks out something other than the pull request's head")
}

// A branch an earlier review left behind is behind the pull request's head, so
// the fetch is made regardless and that branch taken only where none can be.
func TestAPullRequestFallsBackToTheBranchAnEarlierReviewLeft(t *testing.T) {
	s := repository(t)
	s.settings(systemsOn("github"))
	// No origin to fetch from, so this is the fallback and nothing else.
	testenv.Git(t, s.Repo, "branch", "pr-7")

	r := s.run("add", "7")

	r.came(t, result{Answered: s.at("pr-7")})
	require.Equal(t, "pr-7", testenv.Git(t, r.Answered, "rev-parse", "--abbrev-ref", "HEAD"), "the worktree checks out a branch of its own")
}

// With neither a fetch nor a branch already there, there is nothing to check
// out, and the refusal is git's own.
func TestAPullRequestWithNothingToCheckOutIsRefused(t *testing.T) {
	s := repository(t)
	s.settings(systemsOn("github"))

	r := s.run("add", "7")

	r.refused(t, "git fetch origin pull/7/head:pr-7")
	require.NoDirExists(t, s.at("pr-7"), "the worktree was made despite the refusal")
}

// A configured pattern names the branch a pull request's worktree checks out,
// which is also the name that pull request is retyped as.
func TestAConfiguredPatternNamesAPullRequestsBranch(t *testing.T) {
	s := reviewing(t, nil, "[branch]\npull-request = \"review-{{.Number}}\"\n")

	made := s.run("add", "7")

	made.came(t, result{Answered: s.at("review-7")})
	require.True(t, s.hasBranch("review-7"), "the configured pattern did not name the branch on disk")

	again := s.run("switch", "review-7")

	again.came(t, result{Answered: made.Answered})
}

// The command a fresh worktree opens on is rendered from what the resolver that
// answered supplied, which for a pull request is its number and title. Nothing
// of the tracker's own reaches it: the arms naming beads all render to nothing.
func TestAClaudeSessionIsOpenedOnThePullRequestItWasMadeFor(t *testing.T) {
	put := putsUp(t)
	s := reviewing(t, []string{"claude"}, "",
		testenv.Stub{Name: "gh", Replies: []testenv.Reply{
			{To: []string{"list"}, Says: `[{"number":7,"title":"Review this"}]`},
		}},
		put.answers("0\tpr-7\n"), testenv.Stub{Name: "claude"})

	r := s.hands("add")

	r.came(t, result{Asked: []string{pullRequests(s.Origin), putUp, "claude --name=PR #7: Review this"}})
}

// A pull request the forge was never asked to list carries no title, and the
// session is named by its number alone.
func TestAPullRequestWithNoTitleIsNamedByItsNumber(t *testing.T) {
	s := reviewing(t, []string{"claude"}, "", testenv.Stub{Name: "claude"})

	r := s.hands("add", "7")

	r.came(t, result{Asked: []string{"claude --name=PR #7"}})
}

// A name the tracker does not list is nobody's, so the verb that resolves one
// takes it at its word: a place of its own, on a branch spelled exactly as the
// name is.
func TestANameNoSystemAnswersForIsTakenAtItsWord(t *testing.T) {
	// A name a ticket id could have been read into, the tracker being what settles
	// that it is not one.
	const name = "one-two"
	s := tracking(t, nil, nil, nil, "")

	r := s.run("add", name)

	r.came(t, result{Answered: s.at(name), Asked: []string{listed}})
	require.True(t, s.hasBranch(name), "the branch is not spelled as the name is")
}

// A detached worktree has no branch for a ticket to own, so it stands for
// nothing but itself and is reached under the directory it sits in.
func TestADetachedWorktreeIsReachedByItsDirectory(t *testing.T) {
	s := tracking(t, []ticket{doable}, nil, nil, "")
	path := s.detached("spike")

	r := s.run("go", "spike")

	r.came(t, result{Answered: path, Asked: []string{listed}})
}

// Which worktree a name reaches is git's own answer, so one outside the
// configured directory is reached where it sits: docs/explanation/worktree-identity.md.
func TestAWorktreeOutsideTheConfiguredDirectoryIsReachedWhereItSits(t *testing.T) {
	s := repository(t)
	outside := filepath.Join(t.TempDir(), "elsewhere")
	testenv.Git(t, s.Repo, "worktree", "add", "-b", "spike", outside)

	r := s.run("switch", "spike")

	r.came(t, result{Answered: resolved(t, outside)})
}

// The rows are what each system offers, under the title it gave them, and one
// run asks each system for its listing once however many rows come back.
func TestThePickersRowsAreWhatEachSystemOffers(t *testing.T) {
	put := putsUp(t)
	s := tracking(t, []ticket{doable}, []ticket{doable}, []string{"github"}, "",
		testenv.Stub{Name: "gh", Replies: []testenv.Reply{
			{To: []string{"list"}, Says: `[{"number":7,"title":"Review this"}]`},
		}},
		// The rows are the worktrees, then what each resolver offers in the order they
		// are asked: the forge's pull request, then the tracker's ticket.
		put.answers("1\tbd-1\n"))
	// The remote is what the forge reads the repository off; no row of its own is
	// made, so there is nothing to fetch.
	testenv.Git(t, s.Repo, "remote", "add", "origin", hosted)

	r := s.run("add")

	path := s.at("bd-1")
	r.came(t, result{Answered: path, Asked: []string{listed, vetted, pullRequests(hosted), putUp,
		creates(path, "bd-1-do-a-thing"), claims("bd-1")}}, atOnce)
	rows := strings.Join(put.rows(), "\n")
	require.Regexp(t, `⇄ pr-7\s+·\s+Review this`, plain(rows), "the forge's row is not what the picker was put up with")
	require.Regexp(t, `◆ bd-1\s+·\s+Do a thing`, plain(rows), "the tracker's row is not what the picker was put up with")
}

// The worktrees are still offered, and what work put to it is said: R5 of
// docs/rules/refusals.md.
func TestASystemThatWillNotAnswerCostsItsOwnRowsAlone(t *testing.T) {
	s := repository(t,
		testenv.Stub{Name: "bd", Grumbles: "the database is not there", Exits: 1},
		testenv.Stub{Name: "gh", Grumbles: "not authenticated", Exits: 1},
		testenv.Stub{Name: "fzf", Says: "0\tscratch\n"})
	s.settings(systemsOn("beads", "github"))
	testenv.Git(t, s.Repo, "remote", "add", "origin", hosted)
	path := s.opened("scratch")

	r := s.run("go")

	r.came(t, result{Answered: path,
		Warned: []string{pullRequests(hosted) + ": not authenticated", vetted + ": the database is not there"},
		Asked:  []string{listed, vetted, pullRequests(hosted), putUp}}, atOnce)
}

// A repository with no origin has no pull requests rather than a listing work
// could not make, and a name of your own is typed rather than offered.
func TestWhatNoSystemOffersIsNotPutUp(t *testing.T) {
	s := repository(t)
	s.settings(systemsOn("github"))
	s.opened("scratch")

	r := s.run("add")

	r.came(t, result{Code: 1, Errored: []string{"nothing left to add"}})
}
