package cli

import (
	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
)

// switchCommand is the verb that enters a worktree. It is what the bare form is
// a shortcut for, so it takes the same argument, carries the same flags and
// completes to the same rows; naming it reaches the worktrees the verbs shadow.
func switchCommand(sys work.Systems, run func(o options, target string) error, list func() ([]work.Candidate, error)) *cobra.Command {
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
		RunE: func(_ *cobra.Command, args []string) error {
			return run(o, firstArg(args))
		},
	}
	openOn(cmd, &o, sys.Openers)
	// switch alone declines an action; add creates a worktree no tracker knows.
	decline(cmd, &o, sys.Actions)

	return cmd
}

// enter resolves the target, asks work to bring its worktree into being, and
// hands the terminal over to what came back.
func enter(env work.Env, o options, target string) error {
	c, err := candidate(env, target)
	if err != nil {
		return err
	}
	return open(env, o, c)
}

// candidate is the place to work: an identifier names one outright, and without
// one the picker hands one over.
func candidate(env work.Env, target string) (work.Candidate, error) {
	if target == "" {
		return pickFrom(env.Candidates, "nothing to work on")
	}
	return env.Resolve(target)
}

// open is where every verb that opens something ends: work brings the worktree
// into being if it has to, and the terminal goes to what came back.
func open(env work.Env, o options, c work.Candidate) error {
	h, err := env.Enter(c, work.Options{Open: o.open, Ask: ask, Skip: o.skip})
	if err != nil {
		return err
	}
	return h.Exec()
}
