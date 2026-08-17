package work

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
)

// making is a resolver whose creation is git's own, a stash needing a worktree
// to apply to. then puts a file in that worktree before the parking runs.
type making struct {
	*resolver
	from string
	then func(path string)
}

func (m making) Create(p worktree.Place, path string) error {
	if err := git.NewWorktree(m.from, path, p.Branch); err != nil {
		return err
	}
	if m.then != nil {
		m.then(path)
	}
	return nil
}

// parking is the core over a repository the shell stands in, with git making the
// worktrees rather than a stand-in recording that it would have.
func parking(t *testing.T, then func(path string)) Env {
	t.Helper()
	var s steps
	e, r := env(t, &s, &opener{steps: &s, name: "far"})
	e.Dir = e.Repo
	m := making{resolver: r, from: e.Repo, then: then}
	e.Systems.Resolvers[0], e.Systems.Named = m, m
	return e
}

// at is where the worktree for a place lands.
func at(e Env, name string) string { return filepath.Join(e.Repo, defaultDir, name) }

// dirty puts one of each sort of change in the checkout: staged, unstaged,
// untracked and ignored. The worktree directory is ignored, as a repository does.
func dirty(t *testing.T, repo string) {
	t.Helper()
	testenv.Write(t, filepath.Join(repo, "tracked"), "as committed")
	testenv.Write(t, filepath.Join(repo, ".gitignore"), "ignored\n"+defaultDir+"/\n")
	testenv.Git(t, repo, "add", "tracked", ".gitignore")
	testenv.Git(t, repo, "commit", "-m", "a file to change")

	testenv.Write(t, filepath.Join(repo, "staged"), "staged")
	testenv.Git(t, repo, "add", "staged")
	testenv.Write(t, filepath.Join(repo, "tracked"), "changed")
	testenv.Write(t, filepath.Join(repo, "untracked"), "untracked")
	testenv.Write(t, filepath.Join(repo, "ignored"), "ignored")
}

// added is the place a name of the user's own reaches, with the worktree made
// for it and the parking run.
func added(t *testing.T, e Env, name string) {
	t.Helper()
	c, err := e.Add(name)
	if err != nil {
		t.Fatalf("Add(%q): %v", name, err)
	}
	if _, err := e.Enter(c, Options{Park: true}); err != nil {
		t.Fatalf("Enter: %v", err)
	}
}

// Parking is the whole of the move: the only difference afterwards is which
// directory the changes are in, what was staged still staged.
func TestParkingMovesTheWorkingStateIntoTheWorktree(t *testing.T) {
	e := parking(t, nil)
	dirty(t, e.Repo)

	added(t, e, "x")

	if left := testenv.Git(t, e.Repo, "status", "--porcelain"); left != "" {
		t.Errorf("the checkout was left carrying %q; want it clean", left)
	}
	if on := testenv.Git(t, e.Repo, "rev-parse", "--abbrev-ref", "HEAD"); on != "main" {
		t.Errorf("the checkout was left on %q; want the branch it was already on", on)
	}

	want := "A  staged\n M tracked\n?? untracked"
	if got := testenv.Git(t, at(e, "x"), "status", "--porcelain"); got != want {
		t.Errorf("the worktree carries %q; want %q", got, want)
	}
	if body, err := os.ReadFile(filepath.Join(at(e, "x"), "tracked")); err != nil || string(body) != "changed" {
		t.Errorf("the worktree's tracked file reads %q (%v); want the change that was in the checkout", body, err)
	}
}

// Ignored files are the machine's rather than the work's: a build tree and a
// virtual environment belong to the checkout they were made in.
func TestParkingLeavesIgnoredFilesWhereTheyAre(t *testing.T) {
	e := parking(t, nil)
	dirty(t, e.Repo)

	added(t, e, "x")

	if _, err := os.Stat(filepath.Join(e.Repo, "ignored")); err != nil {
		t.Errorf("the checkout's ignored file: %v; want it left where it was", err)
	}
	if _, err := os.Stat(filepath.Join(at(e, "x"), "ignored")); !os.IsNotExist(err) {
		t.Errorf("the worktree's ignored file: %v; want it never to have travelled", err)
	}
}

// A clean checkout saves nothing, so nothing another stash left behind is
// restored into the worktree.
func TestParkingACleanCheckoutLeavesTheStashAlone(t *testing.T) {
	e := parking(t, nil)
	testenv.Write(t, filepath.Join(e.Repo, "tracked"), "as committed")
	testenv.Git(t, e.Repo, "add", "tracked")
	testenv.Git(t, e.Repo, "commit", "-m", "a file to change")

	// An entry of the user's own, from work the parking has no business restoring.
	testenv.Write(t, filepath.Join(e.Repo, "tracked"), "an older experiment")
	testenv.Git(t, e.Repo, "stash", "push")

	added(t, e, "x")

	if list := testenv.Git(t, e.Repo, "stash", "list"); !strings.Contains(list, "stash@{0}") {
		t.Errorf("the stash holds %q; want the entry that was already there", list)
	}
	if left := testenv.Git(t, at(e, "x"), "status", "--porcelain"); left != "" {
		t.Errorf("the worktree carries %q; want nothing restored into it", left)
	}
}

// The verbs that only reach a worktree leave the checkout as they found it, so
// starting a ticket is not parking what was in hand.
func TestWithoutParkingTheCheckoutKeepsWhatItWasCarrying(t *testing.T) {
	e := parking(t, nil)
	dirty(t, e.Repo)
	was := testenv.Git(t, e.Repo, "status", "--porcelain")

	c, err := e.Add("x")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := e.Enter(c, Options{}); err != nil {
		t.Fatalf("Enter: %v", err)
	}

	if now := testenv.Git(t, e.Repo, "status", "--porcelain"); now != was {
		t.Errorf("the checkout carries %q; want the %q it was carrying", now, was)
	}
	if left := testenv.Git(t, at(e, "x"), "status", "--porcelain"); left != "" {
		t.Errorf("the worktree carries %q; want a checkout of the branch alone", left)
	}
}

// A restore that will not apply loses nothing: the entry stays, and the refusal
// names the worktree git put part of it in.
func TestAParkingThatCannotFinishNamesWhereTheChangesAre(t *testing.T) {
	e := parking(t, func(path string) {
		testenv.Write(t, filepath.Join(path, "untracked"), "something else entirely")
	})
	dirty(t, e.Repo)

	c, err := e.Add("x")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err = e.Enter(c, Options{Park: true})
	if err == nil {
		t.Fatal("Enter over a restore that cannot apply: want it refused")
	}
	if !strings.Contains(err.Error(), "stashed") || !strings.Contains(err.Error(), at(e, "x")) {
		t.Errorf("Enter = %v; want it to say the changes are stashed and name the worktree holding part of them", err)
	}
	if list := testenv.Git(t, e.Repo, "stash", "list"); !strings.Contains(list, "stash@{0}") {
		t.Errorf("the stash holds %q; want the entry kept", list)
	}
	if got := testenv.Git(t, at(e, "x"), "status", "--porcelain"); got == "" {
		t.Error("the worktree carries nothing; want the part git did put there, which the refusal names")
	}
}
