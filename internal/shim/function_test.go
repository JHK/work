package shim

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/JHK/work-cli/internal/testenv"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// No command reaches the function work init prints; work init only prints it. It
// reaches the binary with the words typed and stands the shell where it answered.
func TestTheFunctionStandsTheShellWhereTheBinaryAnswered(t *testing.T) {
	for _, f := range []struct{ shell, function string }{
		{"fish", Fish},
		{"bash", Bash},
		{"zsh", Bash},
	} {
		t.Run(f.shell, func(t *testing.T) {
			worktree := t.TempDir()
			ran := testenv.Stubs(t, testenv.Stub{Name: "work", Shell: "echo " + worktree + " >\"$" + CDFile + "\"\n"})

			got := runSourced(t, f.shell, f.function, "work bd-1\npwd")

			require.Equal(t, worktree+"\n", got, "the shell stands somewhere else")
			testenv.Equal(t, []string{"work bd-1"}, ran(), "the binary was asked the wrong thing")
		})
	}
}

// runSourced runs script in the shell with the function sourced, and hands back
// what the shell printed, having found nothing left in the temporary directory.
func runSourced(t *testing.T, shell, function, script string) string {
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
