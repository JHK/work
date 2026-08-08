package work

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/git"
)

// A worktree asked for by name is created on a branch spelled the same way,
// forked from the main checkout, and opens on a session of its own: there is no
// ticket to prompt one with. Every command combines with it, as it does with any
// other fresh worktree.
func TestCreateBareWorktree(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")
	t.Setenv("VISUAL", "vi")
	repo := initRepo(t)
	base := gitCmd(t, repo, "rev-parse", "HEAD")
	noTracker(t)
	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// An agent that would be consulted fails the case rather than answering it.
	e.Conversations = stubConversations{err: errors.New("a fresh worktree carries none")}

	session := []string{"claude", "--permission-mode", "auto"}
	tests := []struct {
		name string
		opts Options
		want []string
	}{
		{"scratch", Options{}, append(session, "--name=scratch")},
		{"scratch-agent", Options{Action: ActionAgent}, append(session, "--name=scratch-agent")},
		{"scratch-shell", Options{Action: ActionShell}, []string{"/usr/bin/fish"}},
		{"scratch-editor", Options{Action: ActionEditor}, []string{"vi", filepath.Join(e.Repo, defaultDir, "scratch-editor")}},
		{"scratch-diff", Options{Action: ActionDiff}, []string{"git", "diff", base}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := e.Create(tt.name)
			if err != nil {
				t.Fatalf("Create(%q): %v", tt.name, err)
			}
			if c.Open || c.Target.Kind != KindPlain || c.Target.Name != tt.name {
				t.Fatalf("Create(%q) = %+v, open %v; want a plain target of its own", tt.name, c.Target, c.Open)
			}

			got, err := e.Enter(c, tt.opts)
			if err != nil {
				t.Fatalf("Enter(%+v): %v", tt.opts, err)
			}
			if want := filepath.Join(e.Repo, defaultDir, tt.name); got.State.Path != want {
				t.Errorf("created at %q; want %q", got.State.Path, want)
			}
			if !slices.Equal(got.Handoff.Run, tt.want) {
				t.Errorf("enter(%+v) runs %q; want %q", tt.opts, got.Handoff.Run, tt.want)
			}
			// The name is the branch, verbatim, and it forked from the main checkout.
			if head := gitCmd(t, repo, "rev-parse", tt.name); head != base {
				t.Errorf("branch %s is at %q; want the main checkout's %q", tt.name, head, base)
			}
		})
	}
}

// The name is taken as it is, so a number is a worktree of that name and not the
// pull request the guess would read into it. Re-entering one needs no flag: the
// listing names it, as it does a ticket's.
func TestCreateSkipsTheGuess(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")
	repo := initRepo(t)
	noTracker(t)
	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	c, err := e.Create("7")
	if err != nil {
		t.Fatalf("Create(7): %v", err)
	}
	if _, err := e.Enter(c, Options{Action: ActionShell}); err != nil {
		t.Fatalf("Enter(7): %v", err)
	}

	again, err := e.Resolve("7")
	if err != nil {
		t.Fatalf("Resolve(7): %v", err)
	}
	if !again.Open || again.Target.Kind != KindPlain || again.Target.Name != "7" {
		t.Errorf("Resolve(7) = %+v, open %v; want the bare worktree it names", again.Target, again.Open)
	}
	if want := filepath.Join(e.Repo, defaultDir, "7"); !git.SameDir(again.path, want) {
		t.Errorf("Resolve(7) enters %q; want %q", again.path, want)
	}
}

// --create asserts the name is free. A branch already holding it is a worktree
// to re-enter, which is the same name without the flag.
func TestCreateRefuses(t *testing.T) {
	repo := initRepo(t)
	gitCmd(t, repo, "branch", "taken")
	noTracker(t)
	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// The worktree-name rule is the one every identifier is held to.
	for _, name := range []string{"", "..", "../../etc", "a/b", "-5", "feature/x"} {
		if _, err := e.Create(name); err == nil || !strings.Contains(err.Error(), "usable worktree name") {
			t.Errorf("Create(%q) = %v; want the name refused", name, err)
		}
	}

	_, err = e.Create("taken")
	if err == nil || !strings.Contains(err.Error(), "work taken") {
		t.Errorf("Create(taken) = %v; want the branch named and work taken pointed at", err)
	}
}

// noTracker puts a bd on PATH that fails whatever it is asked, so a path that
// reaches the tracker fails its test rather than passing quietly.
func noTracker(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bd"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write bd: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
