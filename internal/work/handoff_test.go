package work

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/beads"
)

// execHandoff is the environment variable the re-exec helper below keys on.
// Exec replaces the process, so the only way to observe it is from a child.
const execHandoff = "WORK_TEST_HANDOFF_DIR"

func TestMain(m *testing.M) {
	if dir := os.Getenv(execHandoff); dir != "" {
		if err := (Handoff{Dir: dir, Run: []string{"pwd"}}).Exec(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	os.Exit(m.Run())
}

func TestHandoffExec(t *testing.T) {
	dir := t.TempDir()

	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), execHandoff+"="+dir)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("re-exec: %v", err)
	}
	// pwd resolves symlinks the temp dir may carry; compare what the shell sees.
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != want {
		t.Errorf("ran in %q, want %q", strings.TrimSpace(string(out)), want)
	}
}

func TestHandoffExecRejects(t *testing.T) {
	if err := (Handoff{Dir: "/tmp"}).Exec(); err == nil {
		t.Error("Exec with no command: want an error")
	}
	if err := (Handoff{Dir: "/no/such/dir", Run: []string{"pwd"}}).Exec(); err == nil {
		t.Error("Exec into a missing directory: want an error")
	}
	if err := (Handoff{Dir: "/tmp", Run: []string{"no-such-binary-xyz"}}).Exec(); err == nil {
		t.Error("Exec of a missing binary: want an error")
	}
}

func TestShell(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")
	if got := Shell(); len(got) != 1 || got[0] != "/usr/bin/fish" {
		t.Errorf("Shell() = %q, want the login shell", got)
	}

	t.Setenv("SHELL", "")
	if got := Shell(); len(got) != 1 || got[0] != "/bin/sh" {
		t.Errorf("Shell() = %q, want the fallback", got)
	}
}

func TestEditor(t *testing.T) {
	t.Setenv("EDITOR", "vi")
	t.Setenv("VISUAL", "gvim")
	got, err := Editor("/w")
	if err != nil || !slices.Equal(got, []string{"gvim", "/w"}) {
		t.Errorf("Editor() = %q, %v; want $VISUAL on the worktree", got, err)
	}

	t.Setenv("VISUAL", "")
	got, err = Editor("/w")
	if err != nil || !slices.Equal(got, []string{"vi", "/w"}) {
		t.Errorf("Editor() = %q, %v; want $EDITOR on the worktree", got, err)
	}

	t.Setenv("EDITOR", "")
	if _, err := Editor("/w"); err == nil {
		t.Error("Editor() with neither set: want an error")
	} else if !strings.Contains(err.Error(), "$VISUAL") || !strings.Contains(err.Error(), "$EDITOR") {
		t.Errorf("Editor() = %v; want both variables named", err)
	}
}

// The compiled-in defaults are what work launched before any of this was a
// setting, less the review prompt a pull request used to get.
func TestLaunch(t *testing.T) {
	var e Env
	bead, _ := e.Resolve("bd-1")
	pr, _ := e.Resolve("7")
	tests := []struct {
		name  string
		state State
		opts  Options
		want  []string
	}{
		{
			"a ticket opens on the skill that works it",
			State{Target: bead, Bead: beads.Bead{Title: "a title"}},
			Options{},
			[]string{"claude", "--permission-mode", "auto", "--name=bd-1: a title", "/start bd-1"},
		},
		{
			"a pull request opens on a bare named session",
			State{Target: pr},
			Options{},
			[]string{"claude", "--name=PR #7"},
		},
		{
			"model and effort are placed where the template names them",
			State{Target: pr},
			Options{Model: "opus", Effort: "high"},
			[]string{"claude", "--name=PR #7", "--model=opus", "--effort=high"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.Launch(tt.state, tt.opts)
			if err != nil {
				t.Fatalf("Launch: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Launch() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Returning to a conversation names no session: the agent is run in the
// worktree, and a session is filed by the directory it ran in.
func TestResume(t *testing.T) {
	var e Env
	bead, _ := e.Resolve("bd-1")

	got, err := e.Resume(State{Target: bead}, Options{})
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	want := []string{"claude", "--permission-mode", "auto", "--continue"}
	if !slices.Equal(got, want) {
		t.Errorf("Resume() = %q, want %q", got, want)
	}
}
