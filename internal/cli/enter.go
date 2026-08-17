package cli

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/shim"
	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// switchCommand is the verb that enters a worktree already open. A tab press
// after it offers the worktrees alone, and it declines no action.
func switchCommand(sys work.Systems, run func(o options, target string) (worktree.Handoff, error), list func() ([]work.Candidate, error)) *cobra.Command {
	return opening(&cobra.Command{
		Use:   "switch [<name>|<id>|<pr>|<url>]",
		Short: "Enter the worktree an identifier names",
		Long: `Enter the worktree an identifier names. An identifier is a worktree name, a
ticket id or a pull request; one with no worktree open is refused, work add
being what makes one.

With no identifier, choose among the repository's worktrees. That form needs fzf.

The worktree opens on what action.enter names. Nothing comes into being here, so
no tracker is asked and no action runs.`,
	}, sys, nil, run, list)
}

// enter is the worktree an identifier already has, and a refusal where it has
// none.
func enter(env work.Env, o options, target string) (worktree.Handoff, error) {
	c, err := openWorktree(env, target)
	if err != nil {
		return worktree.Handoff{}, err
	}
	if err := env.Switchable(c); err != nil {
		return worktree.Handoff{}, err
	}
	return open(env, o, c)
}

// open is where every verb that opens something ends: work brings the worktree
// into being if it has to, and says what it opens on.
func open(env work.Env, o options, c work.Candidate) (worktree.Handoff, error) {
	return env.Enter(c, work.Options{Open: o.open, Skip: o.skip})
}

// hand ends the invocation: the worktree goes back to the shell, or the command
// takes the terminal.
func hand(h worktree.Handoff, out io.Writer) error {
	if h.Directory() {
		return shim.Answer(h.Dir, out)
	}
	// Dropped before the exec, so nothing the terminal goes to, and nothing it
	// starts in turn, answers into the shim that called this invocation.
	if err := shim.Forget(); err != nil {
		return err
	}
	return h.Exec()
}
