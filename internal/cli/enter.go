package cli

import (
	"io"

	"github.com/spf13/cobra"

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
func enter(env work.Env, l listing, verb, target string) (worktree.Handoff, error) {
	c, err := targeted(env, l, target, env.Resolve)
	if err != nil {
		return worktree.Handoff{}, err
	}
	if err := env.Switchable(c); err != nil {
		return worktree.Handoff{}, err
	}
	return open(env, verb, c)
}

func open(env work.Env, verb string, c work.Candidate) (worktree.Handoff, error) {
	return env.Enter(c, work.Options{Verb: verb})
}

// hand ends the invocation: the worktree goes back to the shell, or the command
// takes the terminal.
func hand(h worktree.Handoff, stdout io.Writer) error {
	if h.Directory() {
		return shim.Answer(h.Dir, stdout)
	}
	// Dropped before the exec, so nothing the terminal goes to, and nothing it
	// starts in turn, answers into the shim that called this invocation.
	if err := shim.Forget(); err != nil {
		return err
	}
	return h.Exec()
}
