package cli

import (
	"io"
	"log/slog"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/run"
	"github.com/JHK/work-cli/internal/shim"
	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// A tab press after switch offers what its picker offers.
func switchCommand(verb offering[opens]) *cobra.Command {
	return opening(&cobra.Command{
		Use:   "switch [<name>|<id>|<pr>|<url>]",
		Short: "Enter the worktree an identifier names",
		Long: `Enter the worktree an identifier names. An identifier is a worktree name, a
ticket id or a pull request; one with no worktree open is refused, work add
being what makes one.

With no identifier, choose among the repository's worktrees, less the one you are
standing in. That form needs fzf.

The worktree is handed back: your shell stands in it.`,
	}, verb)
}

// enter is the worktree an identifier already has, and a refusal where it has
// none.
func enter(env work.Env, l listing, target string) (worktree.Handoff, error) {
	c, err := targeted(env, l, target, env.Resolve)
	if err != nil {
		return worktree.Handoff{}, err
	}
	if err := env.Switchable(c); err != nil {
		return worktree.Handoff{}, err
	}
	return env.Enter(c, work.Options{})
}

const shellIntegrationAdvice = "the shell integration is not sourced, so your shell stays where it is; see work init --help"

// hand ends the invocation: the worktree goes back to the shell, or the command
// takes the terminal.
func hand(h worktree.Handoff, stdout io.Writer) error {
	if h.Directory() {
		return answer(h.Dir, stdout)
	}
	// Dropped before the exec, so nothing the terminal goes to, and nothing it
	// starts in turn, answers into the shim that called this invocation.
	if err := shim.Forget(); err != nil {
		return err
	}
	return takeTerminal(h)
}

// answer hands the worktree back, with one warning naming work init where a
// terminal rather than the shell function is reading the path.
func answer(dir worktree.Path, stdout io.Writer) error {
	shellRead, err := shim.Answer(string(dir), stdout)
	if err != nil {
		return err
	}
	if !shellRead && isTerminal(stdout) {
		slog.Warn(shellIntegrationAdvice)
	}
	return nil
}

// takeTerminal replaces work with the command, run inside the worktree. It
// returns only on failure.
func takeTerminal(h worktree.Handoff) error {
	slog.Info(run.CommandLine(h.Run[0], h.Run[1:]...))
	// Resolved before the chdir, so a failure leaves the process where it started.
	bin, err := exec.LookPath(h.Run[0])
	if err != nil {
		return err
	}
	if err := os.Chdir(string(h.Dir)); err != nil {
		return err
	}
	return syscall.Exec(bin, h.Run, os.Environ())
}
