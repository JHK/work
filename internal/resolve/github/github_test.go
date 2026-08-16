package github

import (
	"errors"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// pattern is the compiled-in branch naming, which a Branch that never reached
// Load answers with.
var pattern config.Branch

// An identifier is a bare number, a pull request URL, or a name this resolver's
// own branch pattern could have produced, so what the picker shows can be
// retyped. Anything else is another resolver's.
func TestIdentifyReadsAnIdentifier(t *testing.T) {
	r := New(t.TempDir(), pattern)
	for _, arg := range []string{"7", "pr-7", "https://github.com/o/r/pull/7", "o/r/pull/7", "https://github.com/o/r/pull/7/files"} {
		got, err := r.Identify(arg, worktree.Open{})
		if err != nil {
			t.Fatalf("Identify(%q): %v", arg, err)
		}
		want := worktree.Place{ID: "7", Name: "pr-7", Branch: "pr-7"}
		if got != want {
			t.Errorf("Identify(%q) = %+v; want %+v", arg, got, want)
		}
	}
}

// A number reaches one worktree and one refspec however it is spelled, so the
// digits are canonicalised rather than carried.
func TestIdentifyCanonicalisesTheNumber(t *testing.T) {
	r := New(t.TempDir(), pattern)
	for _, arg := range []string{"007", "pr-007", "https://github.com/o/r/pull/007"} {
		got, err := r.Identify(arg, worktree.Open{})
		if err != nil {
			t.Fatalf("Identify(%q): %v", arg, err)
		}
		if got.ID != "7" || got.Name != "pr-7" {
			t.Errorf("Identify(%q) = %+v; want pull request 7 under one name", arg, got)
		}
	}
}

// An identifier this resolver does not answer for passes to the next one; a
// number it does answer for but cannot make a pull request of stops the run,
// there being nothing further to try.
func TestIdentifyPassesOnWhatIsNotAPullRequest(t *testing.T) {
	r := New(t.TempDir(), pattern)
	for _, arg := range []string{"", "bd-42", "pull-request-1", "docs/pull/99-notes.md", "one-two"} {
		if _, err := r.Identify(arg, worktree.Open{}); !errors.Is(err, worktree.ErrUnknown) {
			t.Errorf("Identify(%q) = %v; want it left to the next resolver", arg, err)
		}
	}
	// A zero is recognised as a number and refused outright, rather than passed on:
	// there is nothing further to try.
	for _, arg := range []string{"0", "pr-0", "https://github.com/o/r/pull/0"} {
		_, err := r.Identify(arg, worktree.Open{})
		if err == nil || errors.Is(err, worktree.ErrUnknown) {
			t.Errorf("Identify(%q) = %v; want a number that is no pull request refused", arg, err)
		}
	}
}

// Which identifiers are this forge's is answerable from the spelling alone, which
// is what a forge switched off is asked and all it is asked: a pull request URL is
// ours wherever the host is, and a name another resolver would take is not, gh
// hearing nothing either way.
func TestClaimsReadsTheSpellingAlone(t *testing.T) {
	r := New(t.TempDir(), pattern)
	for _, arg := range []string{"7", "pr-7", "https://github.com/o/r/pull/7", "o/r/pull/7"} {
		p, its := r.Claims(arg)
		if !its || p.Name != "pr-7" {
			t.Errorf("Claims(%q) = %+v, %v; want the pull request it is spelled as", arg, p, its)
		}
	}
	// A name for another resolver, and a number this one recognises and can make no
	// pull request of: neither is a spelling turning the forge back on would answer.
	for _, arg := range []string{"", "bd-42", "one-two", "0", "pr-0"} {
		if p, its := r.Claims(arg); its {
			t.Errorf("Claims(%q) = %+v, true; want it claimed by nobody", arg, p)
		}
	}
}

// A worktree's branch is the name a pull request already carries, so nothing
// about it can have moved on and the spelling settles it. A branch the pattern
// would have spelled otherwise is no pull request's of ours.
func TestIdentifyNamesAWorktreeByItsBranch(t *testing.T) {
	r := New(t.TempDir(), pattern)
	mine, err := r.Identify("", worktree.Open{Path: "/wt/pr-7", Branch: "pr-7"})
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	want := worktree.Place{ID: "7", Name: "pr-7", Branch: "pr-7"}
	if mine != want {
		t.Errorf("Identify = %+v; want %+v", mine, want)
	}

	for _, branch := range []string{"pr-007", "some-branch", "", "bd-42-a-slug"} {
		o := worktree.Open{Path: "/wt/other", Branch: branch}
		if _, err := r.Identify("", o); !errors.Is(err, worktree.ErrUnknown) {
			t.Errorf("Identify(branch %q) = %v; want the worktree left to another resolver", branch, err)
		}
	}
}

// Asked whether a worktree is a given pull request's, the branch the number
// renders to is the whole answer: a prefix of it is another pull request's.
func TestIdentifyConfirmsAPullRequestAgainstAWorktree(t *testing.T) {
	r := New(t.TempDir(), pattern)
	tests := []struct {
		id, branch string
		want       bool
	}{
		{"7", "pr-7", true},
		{"007", "pr-7", true},
		{"7", "pr-70", false},
		{"70", "pr-7", false},
		{"7", "some-branch", false},
		{"bd-42", "pr-7", false},
	}
	for _, tt := range tests {
		o := worktree.Open{Path: "/wt", Branch: tt.branch}
		_, err := r.Identify(tt.id, o)
		if got := err == nil; got != tt.want {
			t.Errorf("Identify(%q, branch %q) = %v; want answered %v", tt.id, tt.branch, err, tt.want)
		}
		// Either way this is a worktree another resolver may claim, never a run to stop.
		if err != nil && !errors.Is(err, worktree.ErrUnknown) {
			t.Errorf("Identify(%q, branch %q) = %v; want unrecognition rather than a failure", tt.id, tt.branch, err)
		}
	}
}

// The configured pattern names the branch a worktree checks out, which is also
// the name a row shows and the user retypes.
func TestConfiguredBranchPattern(t *testing.T) {
	r := New(t.TempDir(), configured(t, "review/{{.Number}}"))

	got, err := r.Identify("7", worktree.Open{})
	if err != nil || got.Name != "review/7" {
		t.Fatalf("Identify(7) = %+v, %v; want the configured name", got, err)
	}
	// And the name it shows is one to retype.
	again, err := r.Identify("review/7", worktree.Open{})
	if err != nil || again != got {
		t.Errorf("Identify(review/7) = %+v, %v; want %+v", again, err, got)
	}
	// The worktree that branch names is that pull request's.
	if _, err := r.Identify("7", worktree.Open{Path: "/wt", Branch: "review/7"}); err != nil {
		t.Errorf("Identify(7, branch review/7) = %v; want the worktree taken", err)
	}
}

// The rows are the open pull requests, drafts included, each under the branch
// its worktree checks out, which is what a row shows and the user retypes.
func TestOfferListsTheOpenPullRequests(t *testing.T) {
	repo := testenv.InitRepo(t)
	// gh is asked only where origin says which repository to ask about.
	testenv.Git(t, repo, "remote", "add", "origin", "https://github.com/o/r")
	gh(t, `[{"number":7,"title":"Seventh pull request"},{"number":9,"title":"Ninth pull request"}]`)

	got, err := New(repo, pattern).Offer()
	if err != nil {
		t.Fatalf("Offer: %v", err)
	}
	want := []worktree.Place{
		{ID: "7", Name: "pr-7", Branch: "pr-7", Label: "Seventh pull request"},
		{ID: "9", Name: "pr-9", Branch: "pr-9", Label: "Ninth pull request"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("Offer() = %+v; want %+v", got, want)
	}
}

// gh being absent, unauthenticated or offline costs these rows and nothing else,
// which the core reads as a resolver that will not list. The refusal names the
// command gh was put, that being all a reader is given to go on.
func TestOfferReportsAGhThatWillNotAnswer(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Git(t, repo, "remote", "add", "origin", "https://github.com/o/r")
	testenv.Stubs(t, testenv.Stub{Name: "gh", Grumbles: "not authenticated", Exits: 1})

	_, err := New(repo, pattern).Offer()
	if err == nil {
		t.Fatal("Offer() with a gh that fails: want the failure reported")
	}
	if head, tail := "gh pr list ", ": not authenticated"; !strings.HasPrefix(err.Error(), head) || !strings.HasSuffix(err.Error(), tail) {
		t.Errorf("Offer = %v; want a refusal from %q to %q", err, head, tail)
	}
}

// A repository with no origin has no host to ask about, so it has no pull
// requests rather than a listing work could not make: gh is left unasked, and
// there is nothing to report to anybody.
func TestOfferWithoutAnOrigin(t *testing.T) {
	repo := testenv.InitRepo(t)
	ran := gh(t, "[]")

	got, err := New(repo, pattern).Offer()
	if err != nil || len(got) != 0 {
		t.Errorf("Offer() = %+v, %v; want no rows and nothing to say", got, err)
	}
	if n := len(ran()); n != 0 {
		t.Errorf("gh was asked %d times; want a repository with no origin asking nothing", n)
	}
}

// A branch left behind by an earlier review is behind the pull request's head, so
// the fetch is made regardless and the local branch is fallen back to only when
// the fetch cannot advance it.
func TestCreateFallsBackToABranchAlreadyLocal(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Git(t, repo, "branch", "pr-7")
	r := New(repo, pattern)
	p, _ := r.Identify("7", worktree.Open{})

	path := filepath.Join(repo, "trees", "pr-7")
	// There is no remote to fetch from, so this is the fallback and nothing else.
	if err := r.Create(p, path); err != nil {
		t.Fatalf("Create: %v", err)
	}
	list, err := git.Worktrees(repo)
	if err != nil {
		t.Fatalf("Worktrees: %v", err)
	}
	if len(list) != 2 || !slices.ContainsFunc(list, func(w git.Worktree) bool {
		return git.SameDir(w.Path, path) && w.Branch == "pr-7"
	}) {
		t.Errorf("the repository has %+v; want the main checkout and the one worktree at %q on branch pr-7", list, path)
	}
}

// With neither a fetch nor a branch of its own there is nothing to check out, and
// the refusal is git's.
func TestCreateRefusesWhatItCannotFetch(t *testing.T) {
	repo := testenv.InitRepo(t)
	r := New(repo, pattern)
	p, _ := r.Identify("7", worktree.Open{})

	if err := r.Create(p, filepath.Join(repo, "trees", "pr-7")); err == nil {
		t.Error("Create with nothing to fetch and no branch: want it refused")
	}
}

// Preparing has nothing to ask: the branch is named from the number alone, so a
// host that cannot be reached never costs a worktree.
func TestPrepareAsksNothing(t *testing.T) {
	r := New(t.TempDir(), pattern)
	p, _ := r.Identify("7", worktree.Open{})
	got, err := r.Prepare(p)
	if err != nil || got != p {
		t.Errorf("Prepare = %+v, %v; want the place unchanged", got, err)
	}
}

// What this resolver tells a command about the worktree it resolved is the pull
// request's number, and nothing that would name a forge.
func TestSupplyIsTheNumber(t *testing.T) {
	r := New(t.TempDir(), pattern)
	p, _ := r.Identify("7", worktree.Open{})
	vals, err := r.Supply(worktree.Tree{Place: p, Path: "/wt"})
	if err != nil {
		t.Fatalf("Supply: %v", err)
	}
	if want := (worktree.Values{"Number": "7"}); !maps.Equal(vals, want) {
		t.Errorf("Supply = %v; want %v", vals, want)
	}
}

// configured reads one settings file holding a pull-request pattern, so the
// pattern is bound as loading binds it.
func configured(t *testing.T, text string) config.Branch {
	t.Helper()
	repo := t.TempDir()
	testenv.Write(t, filepath.Join(repo, config.RepoFile), "[branch]\npull-request = \""+text+"\"\n")
	cfg, err := config.Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg.Branch
}

// gh puts a gh on PATH ahead of the real one that answers with out. work's one
// question to the host is which pull requests are open, which is a listing
// rather than a session.
func gh(t *testing.T, out string) func() []string {
	t.Helper()
	return testenv.Stubs(t, testenv.Stub{Name: "gh", Says: out})
}
