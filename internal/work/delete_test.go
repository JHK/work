package work

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Deleting is git's two commands and nothing else: whatever the worktree was
// made for, it and its branch go, and no tracker is asked along the way.
func TestDeleteCostsNoTracker(t *testing.T) {
	repo := initRepo(t)
	noTracker(t)
	tests := []struct {
		name   string
		arg    string
		branch string
	}{
		{"a ticket's", "one", "one-a-slug"},
		{"a pull request's", "7", "pr-7"},
		{"a bare one", "scratch", "scratch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(repo, defaultDir, tt.branch)
			gitCmd(t, repo, "worktree", "add", "-b", tt.branch, path)
			e, err := Open(repo)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}

			c, err := e.Resolve(tt.arg)
			if err != nil {
				t.Fatalf("Resolve(%q): %v", tt.arg, err)
			}
			d, err := e.Delete(c, false)
			if err != nil {
				t.Fatalf("Delete(%q): %v", tt.arg, err)
			}
			if d.Branch != tt.branch || !strings.HasSuffix(d.Path, tt.branch) {
				t.Errorf("Delete(%q) = %+v; want the worktree at %q and branch %s", tt.arg, d, path, tt.branch)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("%s is still there: %v", path, err)
			}
			if list := gitCmd(t, repo, "worktree", "list"); strings.Contains(list, tt.branch) {
				t.Errorf("git still lists the worktree:\n%s", list)
			}
			if branches := gitCmd(t, repo, "branch", "--list", tt.branch); branches != "" {
				t.Errorf("branch %s is still there: %q", tt.branch, branches)
			}
		})
	}
}

// Both of git's gates are weighed before anything is removed, so a refusal
// leaves the worktree standing rather than half a deletion.
func TestDeleteRefuses(t *testing.T) {
	tests := []struct {
		name  string
		spoil func(t *testing.T, path string)
		want  string
	}{
		{
			"a modified file",
			func(t *testing.T, path string) {
				writeFile(t, filepath.Join(path, "tracked"), "changed")
			},
			"modified or untracked files",
		},
		{
			"an untracked file",
			func(t *testing.T, path string) {
				writeFile(t, filepath.Join(path, "new"), "loose")
			},
			"modified or untracked files",
		},
		{
			"a branch not fully merged",
			func(t *testing.T, path string) {
				writeFile(t, filepath.Join(path, "new"), "committed")
				gitCmd(t, path, "add", ".")
				gitCmd(t, path, "commit", "-m", "ahead")
			},
			"not fully merged",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := initRepo(t)
			writeFile(t, filepath.Join(repo, "tracked"), "root")
			gitCmd(t, repo, "add", ".")
			gitCmd(t, repo, "commit", "-m", "tracked")
			noTracker(t)
			path := filepath.Join(repo, defaultDir, "scratch")
			gitCmd(t, repo, "worktree", "add", "-b", "scratch", path)
			tt.spoil(t, path)

			e, err := Open(repo)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			c, err := e.Resolve("scratch")
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}

			_, err = e.Delete(c, false)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Delete() = %v; want the reason %q", err, tt.want)
			}
			if !strings.Contains(err.Error(), "--force") {
				t.Errorf("Delete() = %v; want --force named", err)
			}
			// Refused before anything went, so neither half of it happened.
			if _, err := os.Stat(path); err != nil {
				t.Errorf("the worktree went despite the refusal: %v", err)
			}
			if branches := gitCmd(t, repo, "branch", "--list", "scratch"); branches == "" {
				t.Error("the branch went despite the refusal")
			}

			if _, err := e.Delete(c, true); err != nil {
				t.Fatalf("Delete(force): %v", err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("--force left %s behind: %v", path, err)
			}
			if branches := gitCmd(t, repo, "branch", "--list", "scratch"); branches != "" {
				t.Errorf("--force left branch scratch behind: %q", branches)
			}
		})
	}
}

// The gate is the branch against the main checkout, so an upstream it has
// diverged from is not a second one: git branch -d would weigh that instead and
// refuse with the worktree already gone.
func TestDeleteTakesABranchItsUpstreamHasLeftBehind(t *testing.T) {
	repo := initRepo(t)
	noTracker(t)
	gitCmd(t, repo, "branch", "elsewhere")
	path := filepath.Join(repo, defaultDir, "scratch")
	gitCmd(t, repo, "worktree", "add", "-b", "scratch", path)
	writeFile(t, filepath.Join(path, "new"), "committed")
	gitCmd(t, path, "add", ".")
	gitCmd(t, path, "commit", "-m", "landed")
	gitCmd(t, repo, "merge", "--no-ff", "-m", "merge", "scratch")
	// A local branch stands in for the remote one: what makes it an upstream is
	// the configuration, not where it sits.
	gitCmd(t, repo, "config", "branch.scratch.remote", ".")
	gitCmd(t, repo, "config", "branch.scratch.merge", "refs/heads/elsewhere")

	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	c, err := e.Resolve("scratch")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if _, err := e.Delete(c, false); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if branches := gitCmd(t, repo, "branch", "--list", "scratch"); branches != "" {
		t.Errorf("the worktree went and branch scratch stayed: %q", branches)
	}
}

// git removes the worktree the process stands in and leaves the shell in a
// directory that is gone, so the refusal is work's own.
func TestDeleteRefusesTheWorktreeStoodIn(t *testing.T) {
	repo := initRepo(t)
	noTracker(t)
	path := filepath.Join(repo, defaultDir, "scratch")
	gitCmd(t, repo, "worktree", "add", "-b", "scratch", path)
	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
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

// A target with no worktree is nothing to delete, whatever else it names.
func TestDeleteWithoutAWorktree(t *testing.T) {
	repo := initRepo(t)
	noTracker(t)
	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	c, err := e.Resolve("nowhere")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if _, err := e.Delete(c, true); err == nil || !strings.Contains(err.Error(), "no worktree") {
		t.Errorf("Delete() = %v; want no worktree to delete", err)
	}
}

// The picker deleting draws from is the repository's worktrees and nothing
// besides: no ready ticket and no pull request is offered up for removal.
func TestWorktreesListsWorktreesAlone(t *testing.T) {
	repo := initRepo(t)
	gitCmd(t, repo, "worktree", "add", "-b", "scratch", filepath.Join(repo, defaultDir, "scratch"))
	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// A tracker with a ready bead of its own, which deleting must not offer.
	shim(t, map[string]string{"list": `[{"id":"one","title":"The first bead"}]`, "ready": `[{"id":"one","title":"The first bead"}]`})

	got, err := e.Worktrees()
	if err != nil {
		t.Fatalf("Worktrees: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Worktrees() = %+v; want the one worktree alone", got)
	}
	if c := got[0]; !c.Open || c.Target.Name != "scratch" || c.branch != "scratch" {
		t.Errorf("Worktrees() = %+v; want the open worktree on branch scratch", c)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
