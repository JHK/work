package shim

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/JHK/work-cli/internal/testenv"
)

// This case sources the function work init prints, which no command reaches:
// work init only prints it. What the binary answers it with is internal/cli's.

func TestMain(m *testing.M) { testenv.Main(m) }

// The function reaches the binary with the words that were typed, and stands the
// shell in the worktree the binary wrote back. bash and zsh read the one file.
func TestTheFunctionStandsTheShellWhereTheBinaryAnswered(t *testing.T) {
	for _, f := range []struct{ shell, function string }{
		{"fish", Fish},
		{"bash", Bash},
		{"zsh", Bash},
	} {
		t.Run(f.shell, func(t *testing.T) {
			worktree := t.TempDir()
			ran := testenv.Stubs(t, testenv.Stub{Name: "work", Shell: "echo " + worktree + " >\"$" + CDFile + "\"\n"})

			got := through(t, f.shell, f.function, "work bd-1\npwd")

			require.Equal(t, worktree+"\n", got, "the shell stands somewhere else")
			testenv.Equal(t, []string{"work bd-1"}, ran(), "the binary was asked the wrong thing")
		})
	}
}

// through runs script in the shell with the function sourced, and hands back
// what the shell printed, having found nothing left in the temporary directory.
func through(t *testing.T, shell, function, script string) string {
	t.Helper()
	path, err := exec.LookPath(shell)
	require.NoErrorf(t, err, "%s is not on PATH", shell)
	sourced, tmp := filepath.Join(t.TempDir(), "function"), t.TempDir()
	testenv.Write(t, sourced, function)
	// Named after the directories are made, so t.TempDir puts none of its own here.
	t.Setenv("TMPDIR", tmp)

	out, err := exec.Command(path, "-c", "source "+sourced+"\n"+script).CombinedOutput()
	require.NoErrorf(t, err, "%s: %s", shell, out)
	left, err := os.ReadDir(tmp)
	require.NoError(t, err)
	require.Empty(t, left, "%q left files behind", script)
	return string(out)
}
