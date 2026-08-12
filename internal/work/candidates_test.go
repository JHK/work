package work

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
)

// offering is a resolver with places of its own to offer and branches it owns,
// which is as much of a tracker as listing needs.
type offering struct {
	name   string
	places []string // the identifiers it offers, whether or not they have a worktree
	fails  error    // it will not list

	// blocks, where set, waits for a second offer to be in flight before
	// answering, and records whether it ever gave up waiting.
	blocks *meeting

	// mute names no worktree from its branch alone, as a tracker that cannot be
	// reached names none of its own.
	mute bool

	// untitled names its worktrees without a title, as a forge reading a pull
	// request number off a branch has none to give: the listing is where its titles
	// come from.
	untitled bool

	// named counts the times it was asked to name a worktree from its branch
	// alone, which is once per worktree per listing.
	named int
}

func (o *offering) Name() string { return o.name }

// Identify owns an identifier it knows and a branch that opens with one.
func (o *offering) Identify(id string, w worktree.Open) (worktree.Place, error) {
	if !w.None() && id == "" {
		o.named++
		if o.mute {
			return worktree.Place{}, worktree.ErrUnknown
		}
	}
	if w.None() {
		if !slices.Contains(o.places, id) {
			return worktree.Place{}, worktree.ErrUnknown
		}
		return worktree.Place{ID: id, Name: id}, nil
	}
	for _, p := range o.places {
		if id != "" && id != p {
			continue
		}
		if w.Branch == p || strings.HasPrefix(w.Branch, p+"-") {
			title := "a title"
			if o.untitled {
				title = ""
			}
			return worktree.Place{ID: p, Name: p, Branch: w.Branch, Label: title}, nil
		}
	}
	return worktree.Place{}, worktree.ErrUnknown
}

func (o *offering) Offer() ([]worktree.Place, error) {
	if o.blocks != nil {
		o.blocks.wait()
	}
	if o.fails != nil {
		return nil, o.fails
	}
	out := make([]worktree.Place, 0, len(o.places))
	for _, p := range o.places {
		out = append(out, worktree.Place{ID: p, Name: p, Label: "a title"})
	}
	return out, nil
}

func (o *offering) Prepare(p worktree.Place) (worktree.Place, error) {
	p.Branch = p.ID + "-a-slug"
	return p, nil
}

func (o *offering) Create(p worktree.Place, path string) error { return nil }

// meeting is two offers meeting: the first waits for the second, and gives up
// rather than deadlocking where it turns out to be the only one in flight. The
// wait is what a test reads, so it is short: two goroutines in one process meet
// in microseconds or never.
type meeting struct {
	both  chan struct{}
	n     atomic.Int32
	alone atomic.Bool
}

func meet() *meeting { return &meeting{both: make(chan struct{})} }

func (m *meeting) wait() {
	if m.n.Add(1) == 2 {
		close(m.both)
		return
	}
	select {
	case <-m.both:
	case <-time.After(time.Second):
		m.alone.Store(true)
	}
}

// drawing is a resolver that says how a screen marks the rows it answers for.
type drawing struct {
	*offering
	icon string
}

func (d drawing) Icon() string { return d.icon }

func names(list []Candidate) []string {
	out := make([]string, len(list))
	for i, c := range list {
		out[i] = c.Name
	}
	return out
}

// What the repository offers to work on is every worktree git knows, then what
// each resolver offers that has none. One place is offered once however it was
// found, keyed on the name it is retyped as.
func TestCandidatesGatherTheWorktreesThenTheOffers(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Git(t, repo, "worktree", "add", "-b", "one-a-slug", filepath.Join(repo, defaultDir, "one"))
	testenv.Git(t, repo, "worktree", "add", "-b", "loose", filepath.Join(repo, defaultDir, "loose"))

	tracker := &offering{name: "tracker", places: []string{"one", "two"}}
	e, _ := bare(t, repo)
	e.Systems.Resolvers = append([]Resolver{tracker}, e.Systems.Resolvers...)

	got, err := e.Candidates()
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Candidates() = %q; want the two worktrees and the one offer left", names(got))
	}
	// The worktrees come first, in git's order; one is open, so the tracker's offer
	// of it is not a second row.
	open := names(got[:2])
	slices.Sort(open)
	if want := []string{"loose", "one"}; !slices.Equal(open, want) {
		t.Errorf("the open rows are %q; want %q ahead of what is merely offered", open, want)
	}
	for _, c := range got[:2] {
		if !c.Open {
			t.Errorf("row %q is not open; want the worktrees ahead of the offers", c.Name)
		}
		// A worktree nothing recognised is still one to reach, adopted by the resolver
		// that takes what is left.
		if c.Name == "loose" && c.Source != "bare" {
			t.Errorf("the unrecognised worktree is %+v; want it adopted by the resolver taking what is left", c.Place)
		}
		if c.Name == "one" && (c.Source != "tracker" || c.Label != "a title") {
			t.Errorf("the recognised worktree is %+v; want it under the resolver that named it", c.Place)
		}
	}
	if got[2].Name != "two" || got[2].Open {
		t.Errorf("the offered row is %+v; want a place with no worktree yet", got[2].Place)
	}
}

// The mark a row is drawn with comes from the resolver that answered for it,
// whether that row was offered, adopted from a worktree or asked for by name, so
// a screen draws a system it was never built against. A resolver that names no
// mark leaves it empty rather than borrowing whichever one the screen prefers.
func TestARowIsMarkedByTheResolverThatAnsweredForIt(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Git(t, repo, "worktree", "add", "-b", "one-a-slug", filepath.Join(repo, defaultDir, "one"))
	testenv.Git(t, repo, "worktree", "add", "-b", "loose", filepath.Join(repo, defaultDir, "loose"))

	const mark = "◆"
	tracker := drawing{&offering{name: "tracker", places: []string{"one", "two"}}, mark}
	e, _ := bare(t, repo)
	e.Systems.Resolvers = append([]Resolver{tracker}, e.Systems.Resolvers...)

	got, err := e.Candidates()
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	for _, c := range got {
		want := ""
		if c.Source == tracker.name {
			want = mark
		}
		if c.Icon != want {
			t.Errorf("%q is marked %q by %s; want %q", c.Name, c.Icon, c.Source, want)
		}
	}

	c, err := e.Resolve("two")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if c.Icon != mark {
		t.Errorf("the place named outright is marked %q; want %q", c.Icon, mark)
	}
}

// Naming a worktree asks as little as it can, so a resolver reading a branch
// alone may have no title for it; the listing that has one arrives beside it. The
// row is the worktree's, and the offer is what completes it, or the title would be
// dropped along with the duplicate row that carried it.
func TestAnOpenPlaceTakesTheTitleFromItsOffer(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Git(t, repo, "worktree", "add", "-b", "one", filepath.Join(repo, defaultDir, "one"))

	e, _ := bare(t, repo)
	forge := &offering{name: "forge", places: []string{"one"}, untitled: true}
	e.Systems.Resolvers = append([]Resolver{forge}, e.Systems.Resolvers...)

	got, err := e.Candidates()
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Candidates() = %q; want the one place, whether it was found open or offered", names(got))
	}
	if !got[0].Open || got[0].Source != "forge" {
		t.Fatalf("the row is %+v; want the worktree under the resolver that named it", got[0].Place)
	}
	if got[0].Label != "a title" {
		t.Errorf("the row is %+v; want the title its offer had", got[0].Place)
	}
}

// Only the resolver that answered for the worktree completes its row: another's
// offer of the same name is another place that happens to be spelled alike, and
// its title would be a title for something else.
func TestAnOpenPlaceTakesNoTitleFromAnotherResolversOffer(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Git(t, repo, "worktree", "add", "-b", "one", filepath.Join(repo, defaultDir, "one"))

	e, _ := bare(t, repo)
	e.Systems.Resolvers = append([]Resolver{
		// The forge names the worktree and has nothing to list, as gh that is absent or
		// unauthenticated has nothing to list.
		&offering{name: "forge", places: []string{"one"}, untitled: true, fails: errors.New("gh is not answering")},
		&offering{name: "tracker", places: []string{"one"}},
	}, e.Systems.Resolvers...)

	got, err := e.Candidates()
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(got) != 1 || got[0].Source != "forge" {
		t.Fatalf("Candidates() = %+v; want the one worktree under the resolver that named it", got)
	}
	if got[0].Label != "" {
		t.Errorf("the row is %+v; want no title rather than another resolver's", got[0].Place)
	}
}

// A resolver that will not answer costs its own rows and never the worktrees.
func TestAResolverThatWillNotOfferCostsItsOwnRows(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Git(t, repo, "worktree", "add", "-b", "loose", filepath.Join(repo, defaultDir, "loose"))

	e, _ := bare(t, repo)
	e.Systems.Resolvers = append([]Resolver{
		&offering{name: "tracker", places: []string{"one"}, fails: errors.New("bd is not answering")},
	}, e.Systems.Resolvers...)

	got, err := e.Candidates()
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if want := []string{"loose"}; !slices.Equal(names(got), want) {
		t.Errorf("Candidates() = %q; want the worktrees alone", names(got))
	}
}

// An offer the core could not make a directory for could be shown but never
// retyped, so it is left off.
func TestCandidatesDropAnOfferThatCouldNotBeMade(t *testing.T) {
	repo := testenv.InitRepo(t)
	e, _ := bare(t, repo)
	e.Systems.Resolvers = append([]Resolver{
		&offering{name: "tracker", places: []string{"..", "a/b", "-5", "fine"}},
	}, e.Systems.Resolvers...)

	got, err := e.Candidates()
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if want := []string{"fine"}; !slices.Equal(names(got), want) {
		t.Errorf("Candidates() = %q; want only the name a worktree could be made for", names(got))
	}
}

// No resolver reads another's answer, so the offers are issued at once and the
// wait is the slowest of them rather than their sum.
func TestCandidatesListAtOnce(t *testing.T) {
	repo := testenv.InitRepo(t)
	e, _ := bare(t, repo)
	both := meet()
	e.Systems.Resolvers = []Resolver{
		&offering{name: "first", places: []string{"one"}, blocks: both},
		&offering{name: "second", places: []string{"two"}, blocks: both},
	}

	if _, err := e.Candidates(); err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if both.alone.Load() {
		t.Error("an offer ran with no other in flight; want them issued at once")
	}
}

// A name a worktree is listed under is that worktree, whatever a resolver would
// make of the name, so what the picker and the completion show is entered by
// retyping it.
func TestResolveNamesAnOpenWorktree(t *testing.T) {
	repo := testenv.InitRepo(t)
	// The directories are named so that git lists them in an order the answer must
	// not depend on: aaa holds spike-2 and zzz holds spike.
	made := []struct{ dir, branch string }{
		{"aaa", "spike-2"}, {"zzz", "spike"}, {"digits", "1234"}, {"slashed", "feature/x"},
	}
	for _, w := range made {
		testenv.Git(t, repo, "worktree", "add", "-b", w.branch, filepath.Join(repo, defaultDir, w.dir))
	}
	e, _ := bare(t, repo)

	for _, w := range made {
		c, err := e.Resolve(w.branch)
		if err != nil {
			t.Errorf("Resolve(%q): %v", w.branch, err)
			continue
		}
		if want := filepath.Join(repo, defaultDir, w.dir); !c.Open || !git.SameDir(c.path, want) {
			t.Errorf("Resolve(%q) = %+v, entering %q; want the worktree at %q", w.branch, c.Place, c.path, want)
		}
	}

	// A name no worktree answers to goes through the chain as it always did.
	if got, err := e.Resolve("nowhere"); err != nil || got.Open {
		t.Errorf("Resolve(nowhere) = %+v, %v; want a place with no worktree", got.Place, err)
	}
}

// A place whose id owns several open branches takes the one that is really its
// own: the shortest branch settles it, so naming a place and locating it land in
// the same worktree and never in whichever one git listed first.
func TestResolveTakesTheShortestBranchWhereSeveralMatch(t *testing.T) {
	repo := testenv.InitRepo(t)
	// The longer branch sorts first, so git reports it ahead of the one wanted.
	testenv.Git(t, repo, "worktree", "add", "-b", "one-a-rather-longer-slug", filepath.Join(repo, defaultDir, "a"))
	testenv.Git(t, repo, "worktree", "add", "-b", "one-slug", filepath.Join(repo, defaultDir, "z"))

	e, _ := bare(t, repo)
	e.Systems.Resolvers = append([]Resolver{&offering{name: "tracker", places: []string{"one"}}}, e.Systems.Resolvers...)

	c, err := e.Resolve("one")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if want := filepath.Join(repo, defaultDir, "z"); !c.Open || !git.SameDir(c.path, want) {
		t.Errorf("Resolve(one) = %q; want the shortest branch at %q", c.path, want)
	}
	// The place as the resolver describes it on that worktree, which knows the branch
	// it was found by rather than the one Prepare would render today.
	if c.Branch != "one-slug" {
		t.Errorf("Resolve(one) = %+v; want the branch the worktree actually has", c.Place)
	}
}

// Several open worktrees still answering for one place is a repository arranged
// by hand: the shortest branch takes it, so it is settled on the branch and never
// on git's listing order.
//
// The tracker here names no worktree from its branch alone, as one that cannot be
// reached names none, so both fall to the resolver taking what is left and neither
// is listed under the name typed. That is what puts the identifier, rather than
// the listing, in charge of finding the worktree.
func TestSeveralWorktreesForOnePlaceGoByTheShortestBranch(t *testing.T) {
	repo := testenv.InitRepo(t)
	// The longer branch sorts first, so git reports it ahead of the one wanted.
	testenv.Git(t, repo, "worktree", "add", "-b", "one-a-rather-longer-slug", filepath.Join(repo, defaultDir, "a"))
	testenv.Git(t, repo, "worktree", "add", "-b", "one-slug", filepath.Join(repo, defaultDir, "z"))

	e, _ := bare(t, repo)
	e.Systems.Resolvers = append([]Resolver{
		&offering{name: "tracker", places: []string{"one"}, mute: true},
	}, e.Systems.Resolvers...)

	c, err := e.Resolve("one")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if want := filepath.Join(repo, defaultDir, "z"); !c.Open || !git.SameDir(c.path, want) {
		t.Errorf("Resolve(one) = %q; want the shortest branch at %q", c.path, want)
	}
	// The place as the resolver describes it on that worktree, which knows the branch
	// it found the place by rather than the one Prepare would render today.
	if c.Source != "tracker" || c.Branch != "one-slug" {
		t.Errorf("Resolve(one) = %+v; want the branch that worktree actually has", c.Place)
	}

	// A place none of them answers for has no worktree of its own, rather than
	// whichever one was left over.
	nine, err := e.Resolve("nine")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if nine.Open {
		t.Errorf("Resolve(nine) = %+v at %q; want no worktree of its own", nine.Place, nine.path)
	}
}

// One invocation asks the resolvers to name each open worktree once: the listing
// resolution made is the one entering reads, whatever the place turns out to be.
func TestOneInvocationNamesEachWorktreeOnce(t *testing.T) {
	repo := testenv.InitRepo(t)
	for dir, branch := range map[string]string{
		"one": "one-a-slug", "loose": "loose", "other": "other-a-slug",
	} {
		testenv.Git(t, repo, "worktree", "add", "-b", branch, filepath.Join(repo, defaultDir, dir))
	}
	e, s := bare(t, repo)
	tracker := &offering{name: "tracker", places: []string{"one", "two"}}
	e.Systems.Resolvers = append([]Resolver{tracker}, e.Systems.Resolvers...)
	e.Systems.Openers = []Opener{&opener{steps: s, name: "far"}}
	e.Config.Action = config.Action{CreateName: "far", EnterName: "far"}

	// A place with a worktree, one without, and a worktree nothing recognises.
	for _, arg := range []string{"one", "two", "loose"} {
		t.Run(arg, func(t *testing.T) {
			tracker.named = 0
			c, err := e.Resolve(arg)
			if err != nil {
				t.Fatalf("Resolve(%q): %v", arg, err)
			}
			if _, err := e.Enter(c, Options{}); err != nil {
				t.Fatalf("Enter(%q): %v", arg, err)
			}
			// Three worktrees, each named once. A second listing would name them again.
			if tracker.named != 3 {
				t.Errorf("entering %q named a worktree from its branch %d times; want one listing of the three", arg, tracker.named)
			}
		})
	}
}

// The name becomes a directory of its own and an argument to git, so it may not
// traverse and may not open with a dash. It is the core's rule, whatever a
// resolver would like to call a place.
func TestResolveRefusesANameNoWorktreeCouldCarry(t *testing.T) {
	repo := testenv.InitRepo(t)
	e, _ := bare(t, repo)

	for _, arg := range []string{"", "..", "../../etc", "a/b", "/", ".", "-5", "--yes", "docs/pull/99-notes.md"} {
		if got, err := e.Resolve(arg); err == nil {
			t.Errorf("Resolve(%q) = %+v; want the name refused", arg, got.Place)
		}
	}
}

// Adding names its resolver rather than putting the name through the chain,
// which would read it as some tracker's identifier. The name is held to the same
// rule every identifier is.
func TestAddNamesItsOwnResolver(t *testing.T) {
	repo := testenv.InitRepo(t)
	e, _ := bare(t, repo)
	// A resolver that would take the name first if the chain were asked at all.
	e.Systems.Resolvers = append([]Resolver{&offering{name: "tracker", places: []string{"7"}}}, e.Systems.Resolvers...)

	c, err := e.Add("7")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if c.Open || c.Source != "bare" || c.Name != "7" {
		t.Errorf("Add(7) = %+v; want a place of the named resolver's own", c.Place)
	}

	for _, name := range []string{"", "..", "../../etc", "a/b", "-5", "feature/x"} {
		if _, err := e.Add(name); err == nil || !strings.Contains(err.Error(), "usable worktree name") {
			t.Errorf("Add(%q) = %v; want the name refused", name, err)
		}
	}
}

// Discovery is what git reports, so a worktree outside the configured directory
// is listed and entered where it is, while a place with none is created under
// the configured directory whatever the ones already open do.
func TestWorktreeDiscovery(t *testing.T) {
	repo := testenv.InitRepo(t)
	outside := filepath.Join(t.TempDir(), "elsewhere-wt")
	testenv.Git(t, repo, "worktree", "add", "-b", "one-oxc-outside", outside)

	e, s := bare(t, repo)
	e.Systems.Openers = []Opener{&opener{steps: s, name: "far"}}
	e.Config.Action = config.Action{CreateName: "far", EnterName: "far"}
	// The main checkout is git's first entry and never a place to work.
	list, err := git.Linked(repo)
	if err != nil {
		t.Fatalf("Linked: %v", err)
	}
	if len(list) != 1 || !git.SameDir(list[0].Path, outside) {
		t.Fatalf("Linked = %+v; want only %q", list, outside)
	}

	c, err := e.Resolve("one-oxc-outside")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !c.Open || !git.SameDir(c.path, outside) {
		t.Errorf("Resolve = %+v at %q; want the worktree at %q", c.Place, c.path, outside)
	}

	// A place with no worktree is created under the configured directory, whatever
	// the ones already open do.
	fresh, err := e.Resolve("one-abc")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if fresh.Open {
		t.Fatalf("one-abc = %+v; want a place with no worktree yet", fresh.Place)
	}
	if _, err := e.Enter(fresh, Options{}); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	made := e.Systems.Named.(*resolver).created
	if want := filepath.Join(repo, defaultDir, "one-abc"); made != want {
		t.Errorf("one-abc was created at %q; want %q", made, want)
	}
}

// The setting decides where a new worktree goes and nothing else: one created
// under an earlier value is still found and entered where it sits.
func TestConfiguredWorktreeDirectory(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Write(t, filepath.Join(repo, config.RepoFile), "[worktree]\ndirectory = \"trees\"\n")

	var s steps
	r := &resolver{steps: &s, name: "bare", bare: true, branch: "one-abc"}
	e, err := Open(repo, func(string, string, config.Config) Systems {
		return Systems{Resolvers: []Resolver{r}, Named: r, Openers: []Opener{&opener{steps: &s, name: "far"}}}
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	e.Config.Action = config.Action{CreateName: "far", EnterName: "far"}

	// From e.Repo, not repo: git reports the repository with symlinks resolved, and
	// a worktree that does not exist yet cannot be resolved to compare.
	trees := filepath.Join(e.Repo, "trees", "one-abc")
	fresh, err := e.Resolve("one-abc")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if _, err := e.Enter(fresh, Options{}); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if r.created != trees {
		t.Errorf("one-abc was created at %q; want %q", r.created, trees)
	}

	testenv.Git(t, repo, "worktree", "add", "-b", "one-abc", trees)
	e.Config.Worktree.Directory = "elsewhere"
	c, err := e.Resolve("one-abc")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !c.Open || !git.SameDir(c.path, trees) {
		t.Errorf("one-abc = open %v at %q; want the worktree at %q", c.Open, c.path, trees)
	}
}

// Run from inside a linked worktree, work still finds the main checkout and
// nests new worktrees under it. The wiring is handed both, the checkout work was
// invoked in being what a resolver forks a new branch from.
func TestOpenFromLinkedWorktree(t *testing.T) {
	repo := testenv.InitRepo(t)
	wt := filepath.Join(repo, defaultDir, "one-abc")
	testenv.Git(t, repo, "worktree", "add", "-b", "one-abc", wt)

	var wired, invoked string
	e, err := Open(wt, func(repo, checkout string, _ config.Config) Systems {
		wired, invoked = repo, checkout
		return Systems{}
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !git.SameDir(e.Repo, repo) || !git.SameDir(wired, repo) {
		t.Errorf("Open(%q) = repo %q, wired %q; want the main checkout %q", wt, e.Repo, wired, repo)
	}
	if !git.SameDir(invoked, wt) {
		t.Errorf("the wiring was handed the checkout %q; want the one work was invoked in, %q", invoked, wt)
	}
}

// A reader's listing is git's own answer: the branch each worktree has checked
// out, the main checkout excepted, and a detached one under its directory,
// which is the name it goes by wherever it has no branch to go by.
func TestBranchesAreWhatTheWorktreesHaveCheckedOut(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Git(t, repo, "worktree", "add", "-b", "one-a-slug", filepath.Join(repo, defaultDir, "one"))
	testenv.Git(t, repo, "worktree", "add", "--detach", filepath.Join(repo, defaultDir, "adrift"))

	e, _ := bare(t, repo)
	got, err := e.Branches()
	if err != nil {
		t.Fatalf("Branches: %v", err)
	}
	slices.Sort(got)
	if want := []string{"adrift", "one-a-slug"}; !slices.Equal(got, want) {
		t.Errorf("Branches() = %q; want %q", got, want)
	}
}

// Without a repository git would answer for whatever directory the process
// happens to be in, so there is nothing to list rather than someone else's
// worktrees.
func TestWithoutARepositoryNothingIsListed(t *testing.T) {
	var e Env
	got, err := e.Worktrees()
	if err != nil || len(got) != 0 {
		t.Errorf("Worktrees() = %+v, %v; want nothing listed", got, err)
	}
	branches, err := e.Branches()
	if err != nil || len(branches) != 0 {
		t.Errorf("Branches() = %q, %v; want nothing listed", branches, err)
	}
}
