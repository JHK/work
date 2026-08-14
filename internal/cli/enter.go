package cli

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/shim"
	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// switchCommand is the verb that enters a worktree. It is what the bare form is
// a shortcut for, so it takes the same argument, carries the same flags and
// completes to the same rows; naming it reaches the worktrees the verbs shadow.
func switchCommand(sys work.Systems, run func(o options, target string) (worktree.Handoff, error), list func() ([]work.Candidate, error)) *cobra.Command {
	var o options

	cmd := &cobra.Command{
		Use:   "switch [<id>|<pr>|<url>]",
		Short: "Enter the worktree an identifier names, creating it if there is none",
		Long: `Enter the worktree an identifier names, creating it if there is none. An
identifier is a worktree name, a ticket id or a pull request.

With no identifier, choose among the repository's worktrees, its ready tickets
and its open pull requests. That form needs fzf.

An existing worktree opens on what action.enter names, a new one on what
action.create names.

Creating a worktree for a ticket vets that ticket and claims it.

work <identifier> is this same command without the verb: work switch add enters
the worktree add.`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: suggest(list),
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := run(o, arg(args, 0))
			if err != nil {
				return err
			}
			return hand(h, cmd.OutOrStdout())
		},
	}
	openOn(cmd, &o, sys.Openers)
	// switch alone declines an action; add creates a worktree no tracker knows.
	decline(cmd, &o, sys.Actions)

	return cmd
}

// enter resolves the target and asks work to bring its worktree into being.
func enter(env work.Env, o options, target string) (worktree.Handoff, error) {
	c, err := candidate(env, target)
	if err != nil {
		return worktree.Handoff{}, err
	}
	return open(env, o, c)
}

// candidate is the place to work: an identifier names one outright, and without
// one the picker hands one over.
func candidate(env work.Env, target string) (work.Candidate, error) {
	if target == "" {
		return pickFrom(env.Candidates)
	}
	return env.Resolve(target)
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
