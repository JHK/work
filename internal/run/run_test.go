package run

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The stand-ins are the tests' own: internal/testenv reaches the machine through
// this package, so its helpers cannot be used from inside it.

// A tool the machine does not have is refused as the question that reached for
// it, and nothing is said here: the refusal goes to whoever asked, and one of
// them says it. Every later question to that tool is refused with it, unasked.
func TestAToolTheMachineDoesNotHaveIsRefusedRatherThanSaid(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	said := quoting(t)

	_, err := Output(dir, "nowhere", "list")
	if err == nil || err.Error() != "nowhere list: nowhere is not on PATH" {
		t.Errorf("Output = %v; want the question and the tool that is not there", err)
	}
	if _, err := Output(dir, "nowhere", "show"); err == nil {
		t.Error("a second question to a tool that is not there was answered")
	}
	if _, err := Output(dir, "elsewhere", "list"); err == nil {
		t.Error("a second missing tool was answered")
	}

	if said() != "" {
		t.Errorf("work said %q of refusals it handed back; want them left to whoever asked", said())
	}
}

// Say is for the one caller that throws a refusal away rather than handing it
// on, a worktree left untrusted by a mise that is not there being the one.
func TestSayPutsARefusalOnStderr(t *testing.T) {
	said := quoting(t)

	Say(errors.New("mise trust: mise is not on PATH"))

	if want := "work: mise trust: mise is not on PATH\n"; said() != want {
		t.Errorf("work said %q; want %q", said(), want)
	}
}

// A tool the machine does not have is out for the rest of the run, so one that
// turns up on PATH mid-run is still not asked: what a listing already paid for
// is not paid again by every worktree behind it.
func TestAToolAlreadyMissedIsNotTriedAgain(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	quoting(t)

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
	said := quoting(t)

	_, err := Output(t.TempDir(), "git", "not-a-git-command")
	if err == nil {
		t.Fatal("git answered a command it does not have")
	}
	if strings.Contains(err.Error(), "is not on PATH") {
		t.Errorf("Output = %v; want what the tool said of itself", err)
	}
	if said() != "" {
		t.Errorf("work said %q of a tool that ran; want nothing", said())
	}
}

// A refusal names the whole command work put, arguments and all: whoever reads
// it has to be able to run the same thing by hand to find out why it failed.
func TestARefusalNamesTheWholeCommand(t *testing.T) {
	quoting(t)

	_, err := Output(t.TempDir(), "git", "not-a-git-command", "--and-a-flag")
	if err == nil {
		t.Fatal("git answered a command it does not have")
	}
	if !strings.HasPrefix(err.Error(), "git not-a-git-command --and-a-flag: ") {
		t.Errorf("Output = %v; want the command as it was run", err)
	}
}

// An answer that is not the JSON work asked for is a refusal like any other, and
// names the command that gave it: whoever reads it can go and see what the tool
// answered with instead. Nothing is said here, the refusal being the caller's.
func TestAnAnswerThatIsNotJSONNamesTheCommand(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	said := quoting(t)
	if err := os.WriteFile(filepath.Join(dir, "teller"), []byte("#!/bin/sh\necho not json\n"), 0o755); err != nil {
		t.Fatalf("write teller: %v", err)
	}

	rows, err := JSON[[]string](dir, "teller", "rows", "--json")
	if err == nil {
		t.Fatalf("JSON = %q; want an answer that is not JSON refused", rows)
	}
	if want := "teller rows --json: "; !strings.HasPrefix(err.Error(), want) {
		t.Errorf("JSON = %v; want the refusal to open %q", err, want)
	}
	if said() != "" {
		t.Errorf("work said %q of a refusal it handed back; want it left to the caller", said())
	}
}

// quoting hands back what work has said of the machine, keeping it off the
// terminal, and forgets which tools earlier cases found missing.
func quoting(t *testing.T) func() string {
	t.Helper()
	var said strings.Builder
	was := Warnings
	Warnings = &said
	gone.Clear()
	t.Cleanup(func() { Warnings = was; gone.Clear() })
	return said.String
}
