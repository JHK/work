package shim

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/testenv"
)

func TestMain(m *testing.M) { testenv.Main(m) }

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
// command that parses the file without running it. bash and zsh read the one
// file, each row holding it to the words they share.
var functions = []struct {
	shell    string
	function string
	declares string
	parse    string
}{
	{"fish", Fish, "function work", "--no-execute"},
	{"bash", Bash, "work()", "-n"},
	{"zsh", Bash, "work()", "-n"},
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
			cmd := exec.Command(onPath(t, f.shell), f.parse)
			cmd.Stdin = strings.NewReader(f.function)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("%s %s: %v\n%s", f.shell, f.parse, err, out)
			}
		})
	}
}

// onPath hands back the shell, the case skipped where the machine has none.
func onPath(t *testing.T, shell string) string {
	t.Helper()
	path, err := exec.LookPath(shell)
	if err != nil {
		t.Skipf("%s is not on PATH", shell)
	}
	return path
}

// through runs script in the shell with the function sourced, and hands back
// what the shell printed, having found nothing left in the temporary directory.
func through(t *testing.T, shell, function, script string) string {
	t.Helper()
	sourced, tmp := filepath.Join(t.TempDir(), "function"), t.TempDir()
	testenv.Write(t, sourced, function)
	// Named after the directories are made, so t.TempDir puts none of its own here.
	t.Setenv("TMPDIR", tmp)

	out, err := exec.Command(onPath(t, shell), "-c", "source "+sourced+"\n"+script).CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v\n%s", shell, err, out)
	}
	left, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatalf("read the temporary directory: %v", err)
	}
	if len(left) != 0 {
		t.Errorf("%q left %d file(s) behind", script, len(left))
	}
	return string(out)
}

// A tab press hands back no worktree, so the function passes the completion to
// the binary with no file named.
func TestTheFunctionRelaysCompletion(t *testing.T) {
	for _, f := range functions {
		t.Run(f.shell, func(t *testing.T) {
			ran := testenv.Stubs(t, testenv.Stub{Name: "work", Shell: "echo \"file=$WORK_CD_FILE\"\n"})

			asked := "work " + cobra.ShellCompRequestCmd + " switch"
			if out := through(t, f.shell, f.function, asked); out != "file=\n" {
				t.Errorf("the completion printed %q; want %q", out, "file=\n")
			}
			if want := []string{asked}; !slices.Equal(ran(), want) {
				t.Errorf("the binary was asked %q; want %q", ran(), want)
			}
		})
	}
}

// Every other verb still hands the worktree back through the file, the shell
// standing in it once the function returns.
func TestTheFunctionChangesIntoTheWorktree(t *testing.T) {
	for _, f := range functions {
		t.Run(f.shell, func(t *testing.T) {
			worktree := t.TempDir()
			testenv.Stubs(t, testenv.Stub{Name: "work", Shell: "echo " + worktree + " >\"$WORK_CD_FILE\"\n"})

			if out := through(t, f.shell, f.function, "work bd-1\npwd"); out != worktree+"\n" {
				t.Errorf("the shell stands in %q; want %q", out, worktree+"\n")
			}
		})
	}
}
