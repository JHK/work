package shim

import (
	"io"
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
	if err := Answer("/repo/.worktrees/bd-1", &out, io.Discard); err != nil {
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

	var out, note strings.Builder
	if err := Answer("/repo/.worktrees/bd-1", &out, &note); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if want := "/repo/.worktrees/bd-1\n"; out.String() != want {
		t.Errorf("Answer printed %q; want %q", out.String(), want)
	}
	if note.String() != "" {
		t.Errorf("Answer said %q of the integration; want nothing off a terminal", note.String())
	}
}

// device hands back a terminal to read the path off, the case skipped where the
// machine opens none.
func device(t *testing.T) *os.File {
	t.Helper()
	f, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("no pseudo-terminal to open: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// A terminal reading the path is a shell that never sourced the function, which
// is the one install step nothing else says was skipped.
func TestAnswerTellsATerminalTheIntegrationIsMissing(t *testing.T) {
	t.Setenv(CDFile, "")

	var note strings.Builder
	if err := Answer("/repo/.worktrees/bd-1", device(t), &note); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if !strings.Contains(note.String(), "work init") {
		t.Errorf("Answer said %q of the integration; want work init named", note.String())
	}
	if lines := strings.Count(note.String(), "\n"); lines != 1 {
		t.Errorf("Answer said %d line(s); want the one", lines)
	}
}

// The shell that sourced the function is the one this is installed for, so it
// hears nothing of installing it.
func TestAnswerSaysNothingWhereTheShimNamedAFile(t *testing.T) {
	t.Setenv(CDFile, filepath.Join(t.TempDir(), "answer"))

	var note strings.Builder
	if err := Answer("/repo/.worktrees/bd-1", device(t), &note); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if note.String() != "" {
		t.Errorf("Answer said %q of the integration; want nothing", note.String())
	}
}

// A pipe reading the path is a program, not a person to advise.
func TestAnswerSaysNothingIntoAPipe(t *testing.T) {
	t.Setenv(CDFile, "")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer func() { _, _ = r.Close(), w.Close() }()

	var note strings.Builder
	if err := Answer("/repo/.worktrees/bd-1", w, &note); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if note.String() != "" {
		t.Errorf("Answer said %q of the integration; want nothing down a pipe", note.String())
	}
	got := make([]byte, len("/repo/.worktrees/bd-1\n"))
	if _, err := io.ReadFull(r, got); err != nil {
		t.Fatalf("read the pipe: %v", err)
	}
	if want := "/repo/.worktrees/bd-1\n"; string(got) != want {
		t.Errorf("the pipe carried %q; want %q", got, want)
	}
}

// A discarded path is a script's redirect, not a person's: a device is no more a
// terminal here than the pipe is.
func TestAnswerSaysNothingIntoADiscardedPath(t *testing.T) {
	t.Setenv(CDFile, "")
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	defer func() { _ = null.Close() }()

	var note strings.Builder
	if err := Answer("/repo/.worktrees/bd-1", null, &note); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if note.String() != "" {
		t.Errorf("Answer said %q of the integration; want nothing into %s", note.String(), os.DevNull)
	}
}

// A file that cannot be written is a refusal rather than a silent print: the
// shell would otherwise stay where it was with nothing said.
func TestAnswerRefusesAFileItCannotWrite(t *testing.T) {
	t.Setenv(CDFile, filepath.Join(t.TempDir(), "nowhere", "answer"))

	var out strings.Builder
	if err := Answer("/repo/.worktrees/bd-1", &out, io.Discard); err == nil {
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
