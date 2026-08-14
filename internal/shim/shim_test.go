package shim

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The file the shim named is where the worktree goes, as the one line the
// function reads back.
func TestAnswerWritesTheFileTheShimNamed(t *testing.T) {
	file := filepath.Join(t.TempDir(), "answer")
	t.Setenv(CDFile, file)

	var out strings.Builder
	if err := Answer("/repo/.worktrees/bd-1", &out); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read the answer: %v", err)
	}
	if want := "/repo/.worktrees/bd-1\n"; string(got) != want {
		t.Errorf("the answer is %q; want %q", got, want)
	}
	if out.String() != "" {
		t.Errorf("Answer printed %q as well; want the file alone", out.String())
	}
}

// Run without the shim there is nowhere to write, and the worktree is printed
// instead: the invocation still says where the work is.
func TestAnswerPrintsWhereNothingNamedAFile(t *testing.T) {
	t.Setenv(CDFile, "")

	var out strings.Builder
	if err := Answer("/repo/.worktrees/bd-1", &out); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if want := "/repo/.worktrees/bd-1\n"; out.String() != want {
		t.Errorf("Answer printed %q; want %q", out.String(), want)
	}
}

// A file that cannot be written is a refusal rather than a silent print: the
// shell would otherwise stay where it was with nothing said.
func TestAnswerRefusesAFileItCannotWrite(t *testing.T) {
	t.Setenv(CDFile, filepath.Join(t.TempDir(), "nowhere", "answer"))

	var out strings.Builder
	if err := Answer("/repo/.worktrees/bd-1", &out); err == nil {
		t.Error("Answer into a file that cannot be written: want a refusal")
	}
}

// The function and the binary are two halves of one contract, so the variable
// the function names is the one the binary reads.
func TestTheFunctionNamesTheVariable(t *testing.T) {
	if !strings.Contains(Fish, CDFile) {
		t.Errorf("the function names no %s; it is\n%s", CDFile, Fish)
	}
	if !strings.Contains(Fish, "function work") {
		t.Errorf("the function is not work's; it is\n%s", Fish)
	}
}

// The completions are printed straight after the function, so a last line left
// unterminated would swallow the first of them.
func TestTheFunctionEnds(t *testing.T) {
	if !strings.HasSuffix(Fish, "\n") {
		t.Errorf("the function does not end in a newline; it is\n%s", Fish)
	}
}

// What init prints is sourced by a shell, so it has to parse as one.
func TestTheFunctionParses(t *testing.T) {
	fish, err := exec.LookPath("fish")
	if err != nil {
		t.Skip("fish is not on PATH")
	}
	cmd := exec.Command(fish, "--no-execute")
	cmd.Stdin = strings.NewReader(Fish)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("fish --no-execute: %v\n%s", err, out)
	}
}
