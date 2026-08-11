package work

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/beads"
	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/forge"
	"github.com/JHK/work-cli/internal/git"
)

// defaultDir is where worktrees go with nothing configured.
var defaultDir = config.Default().Worktree.Directory

// diffCmd is what open.diff renders to: a revision for git to resolve, the same
// one whatever the worktree is.
var diffCmd = []string{"git", "diff", "main-worktree/HEAD..."}

// An Env carrying no settings answers for the compiled-in defaults, which is
// what the tests with no repository to read settings from want.
func TestResolve(t *testing.T) {
	var e Env
	tests := []struct {
		arg  string
		kind Kind
		id   string
		name string
	}{
		{"bd-42", KindBead, "bd-42", "bd-42"},
		{"7", KindPR, "7", "pr-7"},
		{"https://github.com/o/r/pull/91", KindPR, "91", "pr-91"},
		{"https://github.com/o/r/pull/91/files", KindPR, "91", "pr-91"},
		{"pull-request-1", KindBead, "pull-request-1", "pull-request-1"},
		{"007", KindPR, "7", "pr-7"},
	}
	for _, tt := range tests {
		got, err := e.Resolve(tt.arg)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", tt.arg, err)
		}
		if want := (Target{Kind: tt.kind, ID: tt.id, Name: tt.name}); got.Target != want {
			t.Errorf("Resolve(%q) = %+v, want %+v", tt.arg, got.Target, want)
		}
	}

	// An identifier becomes a directory name and a refspec; anything that would
	// traverse, or that git would reject, has to be refused up front.
	for _, arg := range []string{"", "..", "../../etc", "a/b", "/", ".", "-5", "--yes", "docs/pull/99-notes.md"} {
		if got, err := e.Resolve(arg); err == nil {
			t.Errorf("Resolve(%q) = %+v, want an error", arg, got)
		}
	}
}

// A worktree is read off the branch it has checked out, whatever directory it
// sits in.
func TestTargetAt(t *testing.T) {
	var e Env
	ids := listed("one", "one-two", "1234")
	tests := []struct {
		name   string
		branch string
		kind   Kind
		id     string
		label  string
	}{
		{"a bare bead id", "one", KindBead, "one", "one"},
		{"an id and a title slug", "one-and-a-slug", KindBead, "one", "one"},
		{"the longest id wins", "one-two-three", KindBead, "one-two", "one-two"},
		// A numeric branch stays a bead: the PR heuristic belongs to command-line
		// input, not to a worktree that already exists.
		{"a numeric id", "1234", KindBead, "1234", "1234"},
		{"a pull request", "pr-7", KindPR, "7", "pr-7"},
		{"a non-canonical pr", "pr-007", KindPlain, "/wt", "pr-007"},
		{"an unknown branch", "some-branch", KindPlain, "/wt", "some-branch"},
		{"detached", "", KindPlain, "/wt", "wt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.targetAt(git.Worktree{Path: "/wt", Branch: tt.branch}, ids)
			want := Target{Kind: tt.kind, ID: tt.id, Name: tt.label}
			if got != want {
				t.Errorf("targetAt(%q) = %+v, want %+v", tt.branch, got, want)
			}
		})
	}

	// Without the tracker every worktree still lists, as a plain one.
	if got := e.targetAt(git.Worktree{Path: "/wt", Branch: "one"}, nil); got.Kind != KindPlain {
		t.Errorf("targetAt without ids = %+v, want a plain worktree", got)
	}
}

// A name the listing shows enters the worktree it shows it for, whatever the
// heuristics would otherwise read into that name.
func TestResolveNamesAnOpenWorktree(t *testing.T) {
	repo := initRepo(t)
	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// Named so that git lists the wrong worktree first: the old resolution took
	// whichever branch the ticket pattern owned first.
	made := map[string]string{}
	for dir, branch := range map[string]string{
		"aaa": "spike-2", "zzz": "spike", "digits": "1234", "slashed": "feature/x",
	} {
		path := filepath.Join(e.Repo, defaultDir, dir)
		gitCmd(t, repo, "worktree", "add", "-b", branch, path)
		made[branch] = path
	}

	for _, branch := range []string{"spike", "spike-2", "1234", "feature/x"} {
		c, err := e.Resolve(branch)
		if err != nil {
			t.Errorf("Resolve(%q): %v", branch, err)
			continue
		}
		if !c.Open || !git.SameDir(c.path, made[branch]) {
			t.Errorf("Resolve(%q) = %+v, entering %q; want the worktree at %q", branch, c.Target, c.path, made[branch])
		}
	}

	// A name no worktree answers to resolves as it always did.
	if got, err := e.Resolve("7"); err != nil || got.Target != e.prTarget("7") {
		t.Errorf("Resolve(7) = %+v, %v; want pull request 7", got.Target, err)
	}
	if got, err := e.Resolve("one-abc"); err != nil || got.Target.Kind != KindBead || got.Target.ID != "one-abc" {
		t.Errorf("Resolve(one-abc) = %+v, %v; want the bead", got.Target, err)
	}
}

// A ticket the listing names is still entered on the branch and never on git's
// listing order, so naming it and locating it land in the same worktree.
func TestResolveNamedTakesTheSameWorktreeAsLocating(t *testing.T) {
	repo := initRepo(t)
	// The longer branch sorts first, so git reports it ahead of the one wanted.
	gitCmd(t, repo, "worktree", "add", "-b", "one-a-rather-longer-slug", filepath.Join(repo, defaultDir, "a"))
	gitCmd(t, repo, "worktree", "add", "-b", "one-slug", filepath.Join(repo, defaultDir, "z"))
	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	shim(t, map[string]string{"list": `[{"id":"one"}]`})

	c, err := e.Resolve("one")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if want := filepath.Join(e.Repo, defaultDir, "z"); !c.Open || !git.SameDir(c.path, want) {
		t.Errorf("Resolve(one) = %q, want the shortest branch at %q", c.path, want)
	}
}

// A ticket whose id owns several open branches takes the one that is really its
// own, and never git's listing order.
func TestLocated(t *testing.T) {
	var e Env
	worktrees := []git.Worktree{
		{Path: "/wt/two", Branch: "one-two-a-slug"},
		{Path: "/wt/one", Branch: "one-a-rather-longer-slug"},
	}
	target := Target{Kind: KindBead, ID: "one", Name: "one"}
	if got := e.located(target, worktrees, listed("one", "one-two")); got.Path != "/wt/one" {
		t.Errorf("located() = %q, want the branch one owns", got.Path)
	}
	// The same rule with nothing left to choose between: one-two's worktree is not
	// one's, however alone it stands.
	if got := e.located(target, worktrees[:1], listed("one", "one-two")); got.Path != "" {
		t.Errorf("located() = %q, want no worktree of one's own", got.Path)
	}
	// Without the tracker to say which id owns what, the shortest branch settles it.
	if got := e.located(target, worktrees, nil); got.Path != "/wt/two" {
		t.Errorf("located() without ids = %q, want the shortest branch", got.Path)
	}
	if got := e.located(Target{Kind: KindBead, ID: "nine", Name: "nine"}, worktrees, nil); got.Path != "" {
		t.Errorf("located() for an unopened ticket = %q, want no worktree", got.Path)
	}
}

// A named target finds its worktree by branch, wherever that worktree sits, and
// the branches a longer id owns are that other ticket's.
func TestMatches(t *testing.T) {
	var e Env
	ids := listed("bd-4", "bd-42")
	tests := []struct {
		arg, branch string
		want        bool
	}{
		{"bd-42", "bd-42", true},
		{"bd-42", "bd-42-port-work-to-go", true},
		{"bd-42", "bd-42.1-a-child", false},
		{"bd-4", "bd-42", false},
		{"bd-4", "bd-42-port-work-to-go", false},
		{"bd-42", "", false},
		{"7", "pr-7", true},
		{"7", "pr-70", false},
	}
	for _, tt := range tests {
		c, err := e.Resolve(tt.arg)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", tt.arg, err)
		}
		w := git.Worktree{Path: "/elsewhere/wt", Branch: tt.branch}
		if got := e.matches(c.Target, w, ids); got != tt.want {
			t.Errorf("Resolve(%q).matches(%q) = %v, want %v", tt.arg, tt.branch, got, tt.want)
		}
	}

	// Where bd names no ids the branch alone stands, so a worktree is still found
	// by the id that opened it.
	if !e.matches(Target{Kind: KindBead, ID: "bd-4", Name: "bd-4"}, git.Worktree{Branch: "bd-4-a-slug"}, nil) {
		t.Error("a ticket does not match its own worktree without the tracker")
	}

	// A plain worktree is only itself, so the path is what identifies it.
	plain := e.targetAt(git.Worktree{Path: "/elsewhere/wt", Branch: "some-branch"}, nil)
	if !e.matches(plain, git.Worktree{Path: "/elsewhere/wt", Branch: "some-branch"}, ids) {
		t.Error("a plain target does not match its own worktree")
	}
	if e.matches(plain, git.Worktree{Path: "/other/wt", Branch: "some-branch"}, ids) {
		t.Error("a plain target matches another worktree on the same branch")
	}
}

// Discovery is what git reports, so a worktree outside the configured directory
// is offered and entered where it is, and the bead it holds is not offered as
// fresh.
func TestWorktreeDiscovery(t *testing.T) {
	repo := initRepo(t)
	outside := filepath.Join(t.TempDir(), "elsewhere-wt")
	gitCmd(t, repo, "worktree", "add", "-b", "one-oxc-outside", outside)

	e := Env{Repo: repo, Config: config.Default()}
	// The main checkout is git's first entry and never a place to work.
	list, err := git.Linked(repo)
	if err != nil {
		t.Fatalf("Linked: %v", err)
	}
	if len(list) != 1 || !git.SameDir(list[0].Path, outside) {
		t.Fatalf("Linked = %+v, want only %q", list, outside)
	}

	s := state(t, e, "one-oxc")
	if !s.Exists || !git.SameDir(s.Path, outside) {
		t.Errorf("one-oxc = exists %v at %q, want the worktree at %q", s.Exists, s.Path, outside)
	}

	// A target with no worktree is created under the configured directory,
	// whatever the ones already open do.
	fresh := state(t, e, "one-abc")
	if want := filepath.Join(repo, defaultDir, "one-abc"); fresh.Exists || fresh.Path != want {
		t.Errorf("one-abc = exists %v at %q, want a fresh path at %q", fresh.Exists, fresh.Path, want)
	}
}

// The setting decides where a new worktree goes, and nothing else: one created
// under an earlier value is still found and entered where it sits.
func TestConfiguredWorktreeDirectory(t *testing.T) {
	repo := initRepo(t)
	body := []byte("[worktree]\ndirectory = \"trees\"\n")
	if err := os.WriteFile(filepath.Join(repo, config.RepoFile), body, 0o644); err != nil {
		t.Fatalf("write %s: %v", config.RepoFile, err)
	}

	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// From e.Repo, not repo: git reports the repository with symlinks resolved,
	// and a worktree that does not exist yet cannot be resolved to compare.
	trees := filepath.Join(e.Repo, "trees", "one-abc")
	if s := state(t, e, "one-abc"); s.Path != trees {
		t.Errorf("one-abc = %q, want %q", s.Path, trees)
	}

	gitCmd(t, repo, "worktree", "add", "-b", "one-abc", trees)
	e.Config.Worktree.Directory = "elsewhere"
	s := state(t, e, "one-abc")
	if !s.Exists || !git.SameDir(s.Path, trees) {
		t.Errorf("one-abc = exists %v at %q, want the worktree at %q", s.Exists, s.Path, trees)
	}
}

// Run from inside a linked worktree, work still nests new worktrees under the
// main checkout.
func TestOpenFromLinkedWorktree(t *testing.T) {
	repo := initRepo(t)
	wt := filepath.Join(repo, defaultDir, "one-abc")
	gitCmd(t, repo, "worktree", "add", "-b", "one-abc", wt)

	e, err := Open(wt)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !git.SameDir(e.Repo, repo) {
		t.Errorf("Open(%q).Repo = %q, want %q", wt, e.Repo, repo)
	}
}

// A new branch forks from the checkout work was invoked in, so a worktree asked
// for from inside another starts from the work under foot rather than from what
// the main checkout happens to have. It still lands under the main checkout's
// worktree directory. Both verbs fork the same way, git alone for a name of the
// user's own and bd for a ticket.
func TestForkFromTheCurrentCheckout(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")
	repo := initRepo(t)
	base := gitCmd(t, repo, "rev-parse", "HEAD")
	under := filepath.Join(repo, defaultDir, "under-foot")
	gitCmd(t, repo, "worktree", "add", "-b", "under-foot", under)
	gitCmd(t, under, "commit", "--allow-empty", "-m", "not on main yet")
	ahead := gitCmd(t, under, "rev-parse", "HEAD")

	// in_progress and named by the listing, so entering the ticket asks bd neither
	// for the bead itself nor for its readiness; untitled, so its branch is its id.
	shim(t, map[string]string{"list": `[{"id":"one","status":"in_progress","issue_type":"task","acceptance_criteria":"It works"}]`})

	// add asks for a name of its own, switch for a ticket.
	add := func(e Env, name string) (Candidate, error) { return e.Create(name) }
	tests := []struct {
		name string
		find func(Env, string) (Candidate, error)
	}{
		{"scratch", add},
		{"one", Env.Resolve},
	}
	e, err := Open(under)
	if err != nil {
		t.Fatalf("Open(%q): %v", under, err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := tt.find(e, tt.name)
			if err != nil {
				t.Fatalf("finding %q: %v", tt.name, err)
			}
			got, err := e.Enter(c, Options{Action: ActionShell})
			if err != nil {
				t.Fatalf("Enter(%q): %v", tt.name, err)
			}
			if want := filepath.Join(repo, defaultDir, tt.name); !git.SameDir(got.State.Path, want) {
				t.Errorf("created at %q; want %q under the main checkout", got.State.Path, want)
			}
			if head := gitCmd(t, repo, "rev-parse", tt.name); head != ahead {
				t.Errorf("branch %s is at %q; want the checkout under foot at %q rather than the main checkout's %q", tt.name, head, ahead, base)
			}
		})
	}
}

// A worktree a listing already found is taken from it. The repository here is
// not one, so a worktree still asked for would not be found at all.
func TestInspectAtTakesTheListedWorktree(t *testing.T) {
	e := Env{Repo: filepath.Join(t.TempDir(), "not-a-repo"), Config: config.Default()}
	target := Target{Kind: KindBead, ID: "one", Name: "one"}

	if s := e.inspectAt(Candidate{Target: target, path: "/elsewhere/wt"}); !s.Exists || s.Path != "/elsewhere/wt" {
		t.Errorf("inspectAt() = exists %v at %q, want the listed worktree", s.Exists, s.Path)
	}
	// An empty path is a target without one, which is where a fresh worktree goes.
	if s := e.inspectAt(Candidate{Target: target}); s.Exists || s.Path != filepath.Join(e.Repo, defaultDir, "one") {
		t.Errorf("inspectAt() = exists %v at %q, want a fresh path under %s", s.Exists, s.Path, defaultDir)
	}
}

// A bead the listing already named is taken from it, and one it could not name
// is asked for. The repository here is not one, so the query that stands in for
// the unnamed bead fails rather than answers.
func TestInspectAtTakesTheListedBead(t *testing.T) {
	e := Env{Repo: filepath.Join(t.TempDir(), "not-a-repo"), Config: config.Default()}
	target := Target{Kind: KindBead, ID: "one", Name: "one"}
	bead := beads.Bead{ID: "one", Title: "The first bead", Status: "open", Type: "task"}

	if s := e.inspectAt(Candidate{Target: target, bead: bead}); s.Bead != bead || s.TicketErr != nil {
		t.Errorf("inspectAt() = %+v, %v; want the listing's own record without asking bd", s.Bead, s.TicketErr)
	}
	if s := e.inspectAt(Candidate{Target: target}); s.TicketErr == nil {
		t.Errorf("inspectAt() = %+v; want a bead the listing could not name asked for", s.Bead)
	}
	// A worktree that already exists needs no ticket, named or not.
	if s := e.inspectAt(Candidate{Target: target, path: "/elsewhere/wt"}); s.Bead != (beads.Bead{}) || s.TicketErr != nil {
		t.Errorf("inspectAt() = %+v, %v; want bd left unasked for an existing worktree", s.Bead, s.TicketErr)
	}
}

// The picker's three sources meet in one list: a worktree is offered once,
// under whatever title its own adapter gave it, and what has none is offered
// fresh.
func TestCandidates(t *testing.T) {
	var e Env
	worktrees := []git.Worktree{
		{Path: "/wt/one", Branch: "one-a-slug"},
		{Path: "/wt/pr-7", Branch: "pr-7"},
		{Path: "/wt/loose", Branch: "loose"},
	}
	// Every bead names a branch and titles a row; only the ready ones are offered
	// fresh, and "one" is claimed already.
	known := []beads.Bead{
		{ID: "one", Title: "The first bead"},
		{ID: "two", Title: "The second bead"},
		{ID: "three", Title: "The third bead"},
	}
	ready := []beads.Bead{{ID: "one", Title: "The first bead"}, {ID: "two", Title: "The second bead"}}
	pulls := []forge.PR{{Number: 7, Title: "Seventh pull request"}, {Number: 9, Title: "Ninth pull request"}}

	// A bead row carries the record it was listed from, so entering it needs no
	// lookup of its own, and an open one the branch it has, so deleting it needs
	// none either.
	want := []Candidate{
		{Target: Target{Kind: KindBead, ID: "one", Name: "one"}, Label: "The first bead", Open: true, branch: "one-a-slug", bead: known[0]},
		{Target: e.prTarget("7"), Label: "Seventh pull request", Open: true, branch: "pr-7"},
		{Target: Target{Kind: KindPlain, ID: "/wt/loose", Name: "loose"}, Open: true, branch: "loose"},
		{Target: e.prTarget("9"), Label: "Ninth pull request"},
		{Target: Target{Kind: KindBead, ID: "two", Name: "two"}, Label: "The second bead", ready: true, bead: ready[1]},
	}
	got := e.candidates(worktrees, known, ready, pulls)
	if len(got) != len(want) {
		t.Fatalf("candidates() = %+v, want %d rows", got, len(want))
	}
	for i := range want {
		got[i].path = "" // where git put it is not what this test is about
		if got[i] != want[i] {
			t.Errorf("row %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	// A forge that would not answer costs the pull request rows and their titles,
	// nothing else.
	got = e.candidates(worktrees, known, ready, nil)
	if len(got) != 4 {
		t.Fatalf("candidates() without gh = %+v, want the worktrees and the ready bead", got)
	}
	if pr := got[1]; pr.Target != e.prTarget("7") || pr.Label != "" || !pr.Open {
		t.Errorf("pr row without gh = %+v, want an untitled open worktree", pr)
	}
}

// One listing answers a whole invocation: entering an identifier asks git for
// its worktrees once and bd for its ids at most once, whatever the target's kind
// and however many worktrees its branch matches. Counted rather than reasoned
// about, so a second lookup cannot come back unobserved.
func TestEnterAsksOnce(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")
	repo := initRepo(t)
	for dir, branch := range map[string]string{
		"one": "one-a-slug", "one-two": "one-two-a-slug", "pr": "pr-7",
		"loose": "loose", "four": "four-a-slug", "four-five": "four-five-a-slug",
	} {
		gitCmd(t, repo, "worktree", "add", "-b", branch, filepath.Join(repo, defaultDir, dir))
	}
	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// A bead the listing names, a pull request by number and by name, a plain
	// worktree, and a bead matched on its branch alone.
	for _, arg := range []string{"one", "one-two", "7", "pr-7", "loose", "four"} {
		t.Run(arg, func(t *testing.T) {
			// four is no bead of bd's, so its worktree is found by its branch rather
			// than named by the listing, and four-five's is a second branch that match
			// has to be narrowed against.
			ran := shim(t, map[string]string{"list": `[{"id":"one"},{"id":"one-two"}]`})
			c, err := e.Resolve(arg)
			if err != nil {
				t.Fatalf("Resolve(%q): %v", arg, err)
			}
			if !c.Open {
				t.Fatalf("Resolve(%q) = %+v, want the worktree it already has", arg, c.Target)
			}
			if _, err := e.Enter(c, Options{Action: ActionShell}); err != nil {
				t.Fatalf("Enter(%q): %v", arg, err)
			}

			out := ran()
			lists, ids := spawns(out, "git worktree list"), spawns(out, "bd list")
			if lists != 1 || ids > 1 {
				t.Errorf("entering %q asked git %d times and bd %d; want one listing and at most one id list", arg, lists, ids)
			}
		})
	}
}

// The same listing serves the vetting: a bead it named is worked from that
// record, so entering a fresh ticket spawns no bd show at all, and only a bead
// it could not name is asked for by itself.
func TestEnterVetsFromTheListing(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")
	repo := initRepo(t)
	// Some worktree, so there is a listing for a fresh ticket to be named by.
	gitCmd(t, repo, "worktree", "add", "-b", "other-a-slug", filepath.Join(repo, defaultDir, "other"))
	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// in_progress, so the vetting needs no readiness query either, and show would
	// answer for no bead at all.
	answers := map[string]string{
		"list": `[{"id":"one","title":"The first bead","status":"in_progress","issue_type":"task","acceptance_criteria":"It works"}]`,
		"show": `[]`,
	}

	ran := shim(t, answers)
	c, err := e.Resolve("one")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if c.Open {
		t.Fatalf("Resolve(one) = %+v, want a ticket with no worktree yet", c.Target)
	}
	if _, err := e.Enter(c, Options{Action: ActionShell}); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if n := spawns(ran(), "bd show"); n != 0 {
		t.Errorf("entering a listed bead asked bd show %d times; want it vetted from the listing", n)
	}

	// A name the listing does not carry is still asked for, and still fails entry
	// as the bead nothing knows.
	ran = shim(t, answers)
	c, err = e.Resolve("two")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	_, err = e.Enter(c, Options{Action: ActionShell})
	if err == nil || !strings.Contains(err.Error(), `no bead "two"`) {
		t.Errorf("Enter(two) = %v; want the missing bead refused", err)
	}
	if n := spawns(ran(), "bd show"); n != 1 {
		t.Errorf("entering an unlisted bead asked bd show %d times; want it asked for once", n)
	}
}

// The picker hands over the record it offered the row from, so choosing a fresh
// ticket adds no query of its own: readiness came with the listing too.
func TestEnterFromThePicker(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")
	repo := initRepo(t)
	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	bead := `[{"id":"one","title":"The first bead","status":"open","issue_type":"task","acceptance_criteria":"It works"}]`
	ran := shim(t, map[string]string{"list": bead, "ready": bead, "show": `[]`})

	candidates, err := e.Candidates()
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Target.ID != "one" || candidates[0].Open {
		t.Fatalf("Candidates() = %+v, want the ready bead offered fresh", candidates)
	}
	if _, err := e.Enter(candidates[0], Options{Action: ActionShell}); err != nil {
		t.Fatalf("Enter: %v", err)
	}

	out := ran()
	if shows, ready := spawns(out, "bd show"), spawns(out, "bd ready"); shows != 0 || ready != 1 {
		t.Errorf("picking a fresh ticket asked bd show %d times and bd ready %d; want the picker's own listings to serve both", shows, ready)
	}
}

// The picker's listings are issued at once: none of them reads another's
// answer, so none of them waits for one.
func TestCandidatesListsAtOnce(t *testing.T) {
	repo := initRepo(t)
	// gh is asked only where origin says which repository to ask about.
	gitCmd(t, repo, "remote", "add", "origin", "https://github.com/o/r")
	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	alone := stalling(t)

	if _, err := e.Candidates(); err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if alone() {
		t.Error("a listing ran with no other in flight; want them issued at once")
	}
}

// stalling puts a bd and a gh on PATH that each wait for a second of them to be
// running, and hands back whether either gave up waiting. A listing issued on
// its own never finds company; the markers outlive their process, so only the
// first of a sequence gives up.
func stalling(t *testing.T) func() bool {
	t.Helper()
	dir := t.TempDir()
	stub := fmt.Sprintf(`#!/bin/sh
: > %[1]q/running-$$
n=0
until [ "$(ls %[1]q | grep -c '^running-')" -ge 2 ]; do
	n=$((n + 1))
	if [ "$n" -gt 100 ]; then
		: > %[1]q/alone
		break
	fi
	sleep 0.05
done
echo '[]'
`, dir)
	for _, tool := range []string{"bd", "gh"} {
		if err := os.WriteFile(filepath.Join(dir, tool), []byte(stub), 0o755); err != nil {
			t.Fatalf("write %s: %v", tool, err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	return func() bool {
		_, err := os.Stat(filepath.Join(dir, "alone"))
		return err == nil
	}
}

// shim puts a counting git and a stub bd on PATH ahead of the real ones, and
// hands back what they have been asked to run. answers is the JSON each bd
// subcommand replies with; one with no answer of its own replies nothing.
func shim(t *testing.T, answers map[string]string) func() []string {
	t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("git: %v", err)
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	write := func(name, body string, mode os.FileMode) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), mode); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	for sub, body := range answers {
		write("answer-"+sub, body, 0o644)
	}
	write("git", fmt.Sprintf("#!/bin/sh\nprintf 'git %%s\\n' \"$*\" >> %q\nexec %q \"$@\"\n", log, realGit), 0o755)
	// bd is answered rather than run: the beads are the test's, and there is no
	// tracker database here to hold them. Creating a worktree is the exception,
	// carried out the way the real bd carries it out — git worktree add, run
	// wherever bd stands — because that is what settles the fork point.
	write("bd", fmt.Sprintf(`#!/bin/sh
printf 'bd %%s\n' "$*" >> %[1]q
if [ "$1" = worktree ]; then
	exec git worktree add "$3" -b "$5"
fi
cat %[2]q/answer-"$1" 2>/dev/null
exit 0
`, log, dir), 0o755)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	return func() []string {
		out, err := os.ReadFile(log)
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("read %s: %v", log, err)
		}
		return strings.Split(strings.TrimSpace(string(out)), "\n")
	}
}

// spawns counts the invocations of one command in what a shim recorded.
func spawns(ran []string, prefix string) int {
	n := 0
	for _, cmd := range ran {
		if strings.HasPrefix(cmd, prefix) {
			n++
		}
	}
	return n
}

// listed is a listing that names ids and nothing else, for the tests that only
// care which beads bd knows about.
func listed(ids ...string) []beads.Bead {
	out := make([]beads.Bead, len(ids))
	for i, id := range ids {
		out[i] = beads.Bead{ID: id}
	}
	return out
}

// state is the path a front end takes: an identifier resolved, then inspected
// on what that resolution already found.
func state(t *testing.T, e Env, arg string) State {
	t.Helper()
	c, err := e.Resolve(arg)
	if err != nil {
		t.Fatalf("Resolve(%q): %v", arg, err)
	}
	return e.inspectAt(c)
}

func initRepo(t *testing.T) string {
	t.Helper()
	// Whoever runs the tests has settings of their own, which Open would read.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	gitCmd(t, dir, "init", "-b", "main")
	gitCmd(t, dir, "commit", "--allow-empty", "-m", "root")
	return dir
}

// gitCmd runs one git command and hands back its stdout, for the callers that
// are asking git something rather than telling it something.
func gitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// The user's own config would otherwise reach in: commit.gpgsign alone makes
	// the empty commit depend on a working key.
	cmd.Env = append(cmd.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(string(out))
}

func TestSlug(t *testing.T) {
	tests := []struct{ title, want string }{
		{"Port work to Go", "port-work-to-go"},
		{"Rate-limit /api/upload", "rate-limit-api-upload"},
		{"  Trailing punctuation!!  ", "trailing-punctuation"},
		{strings.Repeat("ab ", 30), strings.Repeat("ab-", 13) + "a"},
		{"—", ""},
	}
	for _, tt := range tests {
		if got := slug(tt.title); got != tt.want {
			t.Errorf("slug(%q) = %q, want %q", tt.title, got, tt.want)
		}
		if len(slug(tt.title)) > slugLen {
			t.Errorf("slug(%q) is longer than %d", tt.title, slugLen)
		}
	}
}

func TestBranch(t *testing.T) {
	var e Env
	pr, _ := e.Resolve("7")
	if got, err := e.Branch(State{Target: pr.Target}); err != nil || got != "pr-7" {
		t.Errorf("PR branch = %q, %v", got, err)
	}

	bead, _ := e.Resolve("bd-42")
	s := State{Target: bead.Target}
	s.Bead.Title = "Port work to Go"
	if got, err := e.Branch(s); err != nil || got != "bd-42-port-work-to-go" {
		t.Errorf("bead branch = %q, %v", got, err)
	}
}

// A configured pattern names the branch a new worktree checks out, and finds
// that worktree again: the id is matched where the pattern puts it, so a prefix
// costs nothing and a ticket retitled since is still the branch's owner.
func TestConfiguredBranchPattern(t *testing.T) {
	repo := initRepo(t)
	body := "[branch]\nticket = \"feature/{{.ID}}-{{.Slug}}\"\npull-request = \"review/{{.Number}}\"\n"
	if err := os.WriteFile(filepath.Join(repo, config.RepoFile), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", config.RepoFile, err)
	}
	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	target := Target{Kind: KindBead, ID: "one-abc", Name: "one-abc"}
	s := State{Target: target}
	s.Bead.Title = "A new title"
	if got, err := e.Branch(s); err != nil || got != "feature/one-abc-a-new-title" {
		t.Errorf("bead branch = %q, %v; want the configured pattern's", got, err)
	}

	// Made when the bead was titled otherwise, and found all the same.
	wt := filepath.Join(repo, defaultDir, "one-abc")
	gitCmd(t, repo, "worktree", "add", "-b", "feature/one-abc-an-older-title", wt)
	if got := state(t, e, "one-abc"); !got.Exists || !git.SameDir(got.Path, wt) {
		t.Errorf("one-abc = exists %v at %q, want the worktree at %q", got.Exists, got.Path, wt)
	}
	// And the picker reads the same worktree back off its branch.
	if got := e.targetAt(git.Worktree{Path: wt, Branch: "feature/one-abc-an-older-title"}, listed("one-abc")); got != target {
		t.Errorf("targetAt() = %+v, want %+v", got, target)
	}

	// A pull request's branch is the name it is retyped as.
	pr, err := e.Resolve("7")
	if err != nil || pr.Target.Name != "review/7" {
		t.Fatalf("Resolve(7) = %+v, %v; want the configured name", pr.Target, err)
	}
	if got, err := e.Resolve("review/7"); err != nil || got.Target != pr.Target {
		t.Errorf("Resolve(review/7) = %+v, %v; want %+v", got.Target, err, pr.Target)
	}
}
