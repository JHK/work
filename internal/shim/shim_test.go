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

// functions are the shells', each with the words that declare work's and the
// command that parses the file without running it.
var functions = []struct {
	shell    string
	function string
	declares string
	parse    string
}{
	{"fish", Fish, "function work", "--no-execute"},
	{"bash", Bash, "work()", "-n"},
}

// The function and the binary are two halves of one contract, so the variable
// the function names is the one the binary reads.
func TestTheFunctionNamesTheVariable(t *testing.T) {
	for _, f := range functions {
		t.Run(f.shell, func(t *testing.T) {
			if !strings.Contains(f.function, CDFile) {
				t.Errorf("the function names no %s; it is\n%s", CDFile, f.function)
			}
			if !strings.Contains(f.function, f.declares) {
				t.Errorf("the function is not work's; it is\n%s", f.function)
			}
		})
	}
}

// The completions are printed straight after the function, so a last line left
// unterminated would swallow the first of them.
func TestTheFunctionEnds(t *testing.T) {
	for _, f := range functions {
		t.Run(f.shell, func(t *testing.T) {
			if !strings.HasSuffix(f.function, "\n") {
				t.Errorf("the function does not end in a newline; it is\n%s", f.function)
			}
		})
	}
}

// What init prints is sourced by a shell, so it has to parse as one.
func TestTheFunctionParses(t *testing.T) {
	for _, f := range functions {
		t.Run(f.shell, func(t *testing.T) {
			shell, err := exec.LookPath(f.shell)
			if err != nil {
				t.Skipf("%s is not on PATH", f.shell)
			}
			cmd := exec.Command(shell, f.parse)
			cmd.Stdin = strings.NewReader(f.function)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("%s %s: %v\n%s", f.shell, f.parse, err, out)
			}
		})
	}
}
