package work

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
)

// bare wires a resolver answering for any worktree and any identifier, which is
// all deleting needs: whose the worktree is has no bearing on it. An action is
// wired beside it so the steps it returns record either seam, for the tests that
// care what ran behind them. It stands nowhere; a case that turns on where work
// was invoked names it itself.
func bare(t *testing.T, repo string) (Env, *steps) {
	t.Helper()
	var s steps
	r := &resolver{steps: &s, name: "bare", bare: true}
	return Env{Repo: repo, Config: config.Default(), Systems: Systems{
		Resolvers: []Resolver{r},
		Actions:   []Action{&action{steps: &s, name: "on-create"}},
		Named:     r,
	}}, &s
}

// Deleting is git's two commands and nothing besides: the worktree goes and its
// branch goes with it, work never weighing whether the work on it landed.
func TestDeleteTakesTheWorktreeAndItsBranch(t *testing.T) {
	repo := testenv.InitRepo(t)
	path := filepath.Join(repo, defaultDir, "scratch")
	testenv.Git(t, repo, "worktree", "add", "-b", "scratch", path)
	// A commit of its own is what git would refuse to delete the branch over.
	testenv.Git(t, path, "commit", "--allow-empty", "-m", "never landed")
	e, s := bare(t, repo)

	c, err := e.Resolve("scratch")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	d, err := e.Delete(c, false)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if d.Branch != "scratch" || !strings.HasSuffix(d.Path, "scratch") {
		t.Errorf("Delete = %+v; want the worktree at %q and branch scratch", d, path)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("%s is still there: %v", path, err)
	}
	if list := testenv.Git(t, repo, "worktree", "list"); strings.Contains(list, "scratch") {
		t.Errorf("git still lists the worktree:\n%s", list)
	}
	if branches := testenv.Git(t, repo, "branch", "--list", "scratch"); branches != "" {
		t.Errorf("branch scratch is still there: %q", branches)
	}
	// No resolver was asked to prepare or create anything and no action ran: a
	// worktree going away reaches neither seam.
	if len(s.seen) != 0 {
		t.Errorf("deleting ran %q; want git's two commands alone", s.seen)
	}
}

// A detached worktree has no branch to go with it, so nothing is asked of git
// beyond the removal and what went names none.
func TestDeleteADetachedWorktreeTakesNoBranch(t *testing.T) {
	repo := testenv.InitRepo(t)
	path := filepath.Join(repo, defaultDir, "adrift")
	testenv.Git(t, repo, "worktree", "add", "--detach", path)
	e, _ := bare(t, repo)

	c, err := e.Resolve("adrift")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	d, err := e.Delete(c, false)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if d.Branch != "" || !strings.HasSuffix(d.Path, "adrift") {
		t.Errorf("Delete = %+v; want the worktree at %q and no branch", d, path)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("%s is still there: %v", path, err)
	}
}

// A dirty worktree is git's to refuse, and it refuses it before touching
// anything, so nothing is half deleted. --force takes it and its branch.
func TestGitRefusesADirtyWorktree(t *testing.T) {
	repo := testenv.InitRepo(t)
	path := filepath.Join(repo, defaultDir, "scratch")
	testenv.Git(t, repo, "worktree", "add", "-b", "scratch", path)
	testenv.Write(t, filepath.Join(path, "new"), "loose")

	e, _ := bare(t, repo)
	c, err := e.Resolve("scratch")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	_, err = e.Delete(c, false)
	if err == nil || !strings.Contains(err.Error(), "modified or untracked files") ||
		!strings.Contains(err.Error(), "--force") {
		t.Fatalf("Delete = %v; want git's refusal, naming --force", err)
	}
	// Refused before anything went, so neither half of it happened.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the worktree went despite the refusal: %v", err)
	}
	if branches := testenv.Git(t, repo, "branch", "--list", "scratch"); branches == "" {
		t.Error("the branch went despite the refusal")
	}

	if _, err := e.Delete(c, true); err != nil {
		t.Fatalf("Delete(force): %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("--force left %s behind: %v", path, err)
	}
	if branches := testenv.Git(t, repo, "branch", "--list", "scratch"); branches != "" {
		t.Errorf("--force left branch scratch behind: %q", branches)
	}
}

// git removes the worktree the process stands in and leaves the shell in a
// directory that is gone, so the refusal is work's own.
func TestDeleteRefusesTheWorktreeStoodIn(t *testing.T) {
	repo := testenv.InitRepo(t)
	path := filepath.Join(repo, defaultDir, "scratch")
	testenv.Git(t, repo, "worktree", "add", "-b", "scratch", path)
	e, _ := bare(t, repo)
	c, err := e.Resolve("scratch")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// A worktree of its own under it goes with the tree git takes away, so standing
	// in one is standing in this one.
	nested := filepath.Join(path, "nested")
	testenv.Git(t, repo, "worktree", "add", "-b", "nested", nested)

	for _, dir := range []string{path, filepath.Join(path, "within"), nested} {
		e.Dir = standingIn(t, dir)
		// --force is git's override, not a way past this one.
		for _, force := range []bool{false, true} {
			_, err := e.Delete(c, force)
			if err == nil || !strings.Contains(err.Error(), "standing in") {
				t.Errorf("Delete(force %v) from %q = %v; want it refused", force, dir, err)
			}
		}
	}
}

// A place with no worktree is nothing to delete, whatever else it names.
func TestDeleteWithoutAWorktree(t *testing.T) {
	repo := testenv.InitRepo(t)
	e, _ := bare(t, repo)

	c, err := e.Resolve("nowhere")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if _, err := e.Delete(c, true); err == nil || !strings.Contains(err.Error(), "no worktree") {
		t.Errorf("Delete = %v; want no worktree to delete", err)
	}
}

// Removing the main checkout is git's to refuse, and it refuses it with or
// without --force. Every worktree sits under the main checkout, so the refusal
// has to survive being asked for from inside one.
func TestGitRefusesToRemoveTheMainCheckout(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Git(t, repo, "branch", "--move", "trunk")
	path := filepath.Join(repo, defaultDir, "scratch")
	testenv.Git(t, repo, "worktree", "add", "-b", "scratch", path)
	e, _ := bare(t, repo)

	c, err := e.Resolve("trunk")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// From the main checkout and from a worktree under it alike: the refusal is
	// git's, never work's own rule about the worktree stood in.
	for _, dir := range []string{repo, path} {
		e.Dir = dir
		for _, force := range []bool{false, true} {
			if _, err := e.Delete(c, force); err == nil || !strings.Contains(err.Error(), "main working tree") {
				t.Fatalf("Delete(force=%v) from %q = %v; want git's own refusal", force, dir, err)
			}
		}
	}
	if head := testenv.Git(t, repo, "rev-parse", "--abbrev-ref", "HEAD"); head != "trunk" {
		t.Errorf("the main checkout is on %q; want it left standing on trunk", head)
	}
	if branches := testenv.Git(t, repo, "branch", "--list", "trunk"); branches == "" {
		t.Error("the refusal cost the main checkout its branch")
	}
}

// The picker deleting draws from is the repository's worktrees and nothing
// besides: no place a resolver merely offers is put up for removal.
func TestWorktreesListsWorktreesAlone(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Git(t, repo, "worktree", "add", "-b", "scratch", filepath.Join(repo, defaultDir, "scratch"))
	e, _ := bare(t, repo)
	// A resolver with a place of its own to offer, which deleting must not offer.
	e.Systems.Resolvers = append(e.Systems.Resolvers, &offering{name: "tracker", places: []string{"one"}})

	got, err := e.Worktrees()
	if err != nil {
		t.Fatalf("Worktrees: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Worktrees() = %+v; want the main checkout and the one worktree alone", got)
	}
	if c := got[1]; !c.Open || c.Name != "scratch" || c.branch != "scratch" {
		t.Errorf("Worktrees() = %+v; want the open worktree on branch scratch", c)
	}

	// The same repository offers that place where what is worth starting is asked for.
	all, _, err := e.Candidates()
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if names := names(all); !slices.Equal(names, []string{"main", "scratch", "one"}) {
		t.Errorf("Candidates() = %q; want the worktree and the offer", names)
	}
}
