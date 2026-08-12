package run

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The stand-ins are the tests' own: internal/testenv reaches the machine through
// this package, so its helpers cannot be used from inside it.

// A tool the machine does not have is said where work reached for it, once
// however many questions that one invocation put to it, and the caller is handed
// the same sentence. Once, because a listing asks a tracker for more than one
// thing; said at all, because the listing swallows what it was told.
func TestAToolTheMachineDoesNotHaveIsSaidOnce(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	var out strings.Builder
	quoting(t, &out)

	if _, err := Output(dir, "nowhere", "list"); err == nil || !strings.Contains(err.Error(), "nowhere is not on PATH") {
		t.Errorf("Output = %v; want the caller told which command is missing", err)
	}
	if _, err := Output(dir, "nowhere", "show"); err == nil {
		t.Error("a second question to a tool that is not there was answered")
	}
	if _, err := Output(dir, "elsewhere", "list"); err == nil {
		t.Error("a second missing tool was answered")
	}

	want := "work: nowhere is not on PATH\nwork: elsewhere is not on PATH\n"
	if out.String() != want {
		t.Errorf("work said %q; want %q", out.String(), want)
	}
}

// A tool the machine does not have is out for the rest of the run, so one that
// turns up on PATH mid-run is still not asked: what a listing already paid for
// is not paid again by every worktree behind it.
func TestAToolAlreadyMissedIsNotTriedAgain(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	var out strings.Builder
	quoting(t, &out)

	if _, err := Output(dir, "later", "list"); err == nil {
		t.Fatal("a tool that is not there answered")
	}
	if err := os.WriteFile(filepath.Join(dir, "later"), []byte("#!/bin/sh\necho here\n"), 0o755); err != nil {
		t.Fatalf("write later: %v", err)
	}

	if got, err := Output(dir, "later", "list"); err == nil {
		t.Errorf("Output = %q; want the tool left unasked for the rest of the run", got)
	}
}

// A tool that is there and fails is a tool that ran: what it said of itself is
// the message, and nothing claims the machine does not have it. git is the one
// every test here already rests on.
func TestAToolThatRanAndFailedIsNotReportedMissing(t *testing.T) {
	var out strings.Builder
	quoting(t, &out)

	_, err := Output(t.TempDir(), "git", "not-a-git-command")
	if err == nil {
		t.Fatal("git answered a command it does not have")
	}
	if strings.Contains(err.Error(), "is not on PATH") {
		t.Errorf("Output = %v; want what the tool said of itself", err)
	}
	if out.String() != "" {
		t.Errorf("work said %q of a tool that ran; want nothing", out.String())
	}
}

// quoting sends what work says about the machine to the test rather than to the
// terminal, and forgets which tools earlier cases found missing.
func quoting(t *testing.T, w *strings.Builder) {
	t.Helper()
	was := warnings
	warnings = w
	gone.Clear()
	t.Cleanup(func() { warnings = was; gone.Clear() })
}
