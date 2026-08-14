package worktree

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	here, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := (Handoff{Dir: "/tmp"}).Exec(); err == nil {
		t.Error("Exec with no command: want an error")
	}
	if err := (Handoff{Dir: "/no/such/dir", Run: []string{"pwd"}}).Exec(); err == nil {
		t.Error("Exec into a missing directory: want an error")
	}
	if err := (Handoff{Dir: "/tmp", Run: []string{"no-such-binary-xyz"}}).Exec(); err == nil {
		t.Error("Exec of a missing binary: want an error")
	}
	if now, err := os.Getwd(); err != nil || now != here {
		t.Errorf("a refused Exec left the process in %q (%v); want %q", now, err, here)
	}
}

// A handoff naming no command is the worktree itself, which a front end answers
// with rather than running.
func TestHandoffDirectory(t *testing.T) {
	if !(Handoff{Dir: "/wt"}).Directory() {
		t.Error("a handoff naming no command is not read as the worktree itself")
	}
	if (Handoff{Dir: "/wt", Run: []string{"fish"}}).Directory() {
		t.Error("a handoff naming a command is read as the worktree itself")
	}
}

// The first to set a name owns it, which is what keeps the core's account of the
// worktree ahead of what the resolver supplies. A value supplied empty is a value
// all the same.
func TestValuesMergeKeepsTheFirstAnswer(t *testing.T) {
	vals := Values{"Name": "bd-1", "Editor": ""}
	vals.Merge(Values{"Name": "the resolver's", "Editor": "vi", "Shell": "fish"})

	want := Values{"Name": "bd-1", "Editor": "", "Shell": "fish"}
	if !maps.Equal(vals, want) {
		t.Errorf("Merge left %v; want %v", vals, want)
	}
}

// git always names a path, so a resolver shown no path is being asked about an
// identifier alone rather than about a worktree without one.
func TestOpenNone(t *testing.T) {
	if !(Open{}).None() {
		t.Error("an Open with no path is not read as an identifier alone")
	}
	// The path is the whole question: a detached worktree carries no branch and is
	// a worktree all the same.
	if (Open{Path: "/wt"}).None() {
		t.Error("an Open with a path is read as an identifier alone")
	}
}
