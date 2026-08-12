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
// care what ran behind them.
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

// Deleting is git's two commands and nothing besides: whatever the worktree was
// made for, it and its branch go, and nothing behind either seam is asked
// anything along the way.
func TestDeleteTakesTheWorktreeAndItsBranch(t *testing.T) {
	tests := []struct {
		name   string
		branch string
	}{
		{"a ticket's", "one-a-slug"},
		{"a pull request's", "pr-7"},
		{"a bare one", "scratch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testenv.InitRepo(t)
			path := filepath.Join(repo, defaultDir, tt.branch)
			testenv.Git(t, repo, "worktree", "add", "-b", tt.branch, path)
			e, s := bare(t, repo)

			c, err := e.Resolve(tt.branch)
			if err != nil {
				t.Fatalf("Resolve(%q): %v", tt.branch, err)
			}
			d, err := e.Delete(c, false)
			if err != nil {
				t.Fatalf("Delete(%q): %v", tt.branch, err)
			}

			if d.Branch != tt.branch || !strings.HasSuffix(d.Path, tt.branch) {
				t.Errorf("Delete = %+v; want the worktree at %q and branch %s", d, path, tt.branch)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("%s is still there: %v", path, err)
			}
			if list := testenv.Git(t, repo, "worktree", "list"); strings.Contains(list, tt.branch) {
				t.Errorf("git still lists the worktree:\n%s", list)
			}
			if branches := testenv.Git(t, repo, "branch", "--list", tt.branch); branches != "" {
				t.Errorf("branch %s is still there: %q", tt.branch, branches)
			}
			// No resolver was asked to prepare or create anything and no action ran: a
			// worktree going away reaches neither seam, which is what makes a ticket's
			// worktree and a bare one go the same way.
			if len(s.seen) != 0 {
				t.Errorf("deleting ran %q; want git's two commands alone", s.seen)
			}
		})
	}
}

// A dirty worktree is refused by the removal itself, before it touches
// anything, so nothing is half deleted.
func TestDeleteRefuses(t *testing.T) {
	tests := []struct {
		name  string
		spoil func(t *testing.T, path string)
		want  string
	}{
		{
			"a modified file",
			func(t *testing.T, path string) {
				testenv.Write(t, filepath.Join(path, "tracked"), "changed")
			},
			"modified or untracked files",
		},
		{
			"an untracked file",
			func(t *testing.T, path string) {
				testenv.Write(t, filepath.Join(path, "new"), "loose")
			},
			"modified or untracked files",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testenv.InitRepo(t)
			testenv.Write(t, filepath.Join(repo, "tracked"), "root")
			testenv.Git(t, repo, "add", ".")
			testenv.Git(t, repo, "commit", "-m", "tracked")
			path := filepath.Join(repo, defaultDir, "scratch")
			testenv.Git(t, repo, "worktree", "add", "-b", "scratch", path)
			tt.spoil(t, path)

			e, _ := bare(t, repo)
			c, err := e.Resolve("scratch")
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}

			_, err = e.Delete(c, false)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Delete = %v; want the reason %q", err, tt.want)
			}
			if !strings.Contains(err.Error(), "--force") {
				t.Errorf("Delete = %v; want --force named", err)
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
		})
	}
}

// Whether a branch's work has landed is git's to judge, and a branch it will not
// delete leaves the worktree standing: both are weighed before either goes. git
// weighs the branch against its upstream where it has one, so one merged into the
// main checkout is refused all the same, the question never being work's own.
// --force takes both either way.
func TestDeleteRefusesABranchGitWillNotDelete(t *testing.T) {
	tests := []struct {
		name string
		land func(t *testing.T, repo string)
	}{
		{"work that never landed", func(*testing.T, string) {}},
		{"work landed on the main checkout but not on its upstream", func(t *testing.T, repo string) {
			testenv.Git(t, repo, "branch", "elsewhere")
			testenv.Git(t, repo, "merge", "--no-ff", "-m", "merge", "scratch")
			// A local branch stands in for the remote one: what makes it an upstream
			// is the configuration, not where it sits.
			testenv.Git(t, repo, "config", "branch.scratch.remote", ".")
			testenv.Git(t, repo, "config", "branch.scratch.merge", "refs/heads/elsewhere")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testenv.InitRepo(t)
			testenv.Write(t, filepath.Join(repo, ".gitignore"), "built\n")
			testenv.Git(t, repo, "add", ".gitignore")
			testenv.Git(t, repo, "commit", "-m", "ignore")
			path := filepath.Join(repo, defaultDir, "scratch")
			testenv.Git(t, repo, "worktree", "add", "-b", "scratch", path)
			testenv.Git(t, path, "commit", "--allow-empty", "-m", "ahead")
			// What a worktree is provisioned with is ignored rather than tracked, so a
			// worktree left standing has to be the one that stood there.
			testenv.Write(t, filepath.Join(path, "built"), "provisioned")
			tt.land(t, repo)

			e, _ := bare(t, repo)
			c, err := e.Resolve("scratch")
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}

			// git's own words, the branch named and the flag that takes both.
			_, err = e.Delete(c, false)
			if err == nil || !strings.Contains(err.Error(), "merged") ||
				!strings.Contains(err.Error(), "scratch") || !strings.Contains(err.Error(), "--force") {
				t.Fatalf("Delete() = %v; want git's refusal, the branch named and --force named", err)
			}
			// Refused with both still there, the worktree back on its branch and the
			// files under it untouched.
			if head := testenv.Git(t, path, "rev-parse", "--abbrev-ref", "HEAD"); head != "scratch" {
				t.Errorf("%s is on %q; want it standing on scratch", path, head)
			}
			if branches := testenv.Git(t, repo, "branch", "--list", "scratch"); branches == "" {
				t.Fatal("the branch went despite git's refusal")
			}
			if _, err := os.Stat(filepath.Join(path, "built")); err != nil {
				t.Errorf("the refusal cost the worktree its ignored files: %v", err)
			}

			// --force takes the same worktree the refusal left standing: nothing has to be
			// put back by hand first.
			if _, err := e.Delete(c, true); err != nil {
				t.Fatalf("Delete(force): %v", err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("--force left %s behind: %v", path, err)
			}
			if branches := testenv.Git(t, repo, "branch", "--list", "scratch"); branches != "" {
				t.Errorf("--force left branch scratch behind: %q", branches)
			}
		})
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

	for _, dir := range []string{path, filepath.Join(path, "within")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		t.Chdir(dir)
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
	if len(got) != 1 {
		t.Fatalf("Worktrees() = %+v; want the one worktree alone", got)
	}
	if c := got[0]; !c.Open || c.Name != "scratch" || c.branch != "scratch" {
		t.Errorf("Worktrees() = %+v; want the open worktree on branch scratch", c)
	}

	// The same repository offers that place where what is worth starting is asked for.
	all, err := e.Candidates()
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if names := names(all); !slices.Equal(names, []string{"scratch", "one"}) {
		t.Errorf("Candidates() = %q; want the worktree and the offer", names)
	}
}
