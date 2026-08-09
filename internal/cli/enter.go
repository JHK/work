package cli

import (
	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
)

// switchCommand is the verb that enters a worktree. It is what the bare form is
// a shortcut for, so it takes the same argument, carries the same flags and
// completes to the same rows; naming it reaches the worktrees the verbs shadow.
func switchCommand(run func(o options, target string) error, list func() ([]work.Candidate, error)) *cobra.Command {
	var o options

	cmd := &cobra.Command{
		Use:   "switch [<id>|<pr>|<url>]",
		Short: "Enter the worktree an identifier names, creating it if there is none",
		Long: `Enter the worktree an identifier names, creating it if there is none.

With no identifier, choose among the repository's worktrees, its ready tickets
and its open pull requests. That form needs fzf.

Entering a worktree that already exists hands it to what action.enter names,
which by default asks. A target without one has its worktree created and handed
to what action.create names, the configured launcher by default. --agent,
--shell, --editor, --diff and --ask name the action to open on instead, for that
invocation.

--ask offers the actions that apply and opens on the one picked; dismissing that
list creates nothing and claims nothing. That form needs fzf.

--agent hands the worktree to its agent. It changes nothing for one just
created, which opens on the launcher regardless; an existing one is handed over
by what it carries, no conversation starting one, a single one being returned
to, and several reaching the agent's own list.

Creating a worktree for a ticket vets that ticket and claims it, whatever the
worktree then opens on. A ticket the vetting refuses is refused outright;
--no-claim declines the claim and nothing else.

work <identifier>, without the verb, is this same command. The verb is how a
worktree named for one is reached: work switch add enters the worktree named
add, where work add creates one.`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: suggest(list),
		RunE: func(_ *cobra.Command, args []string) error {
			return run(o, firstArg(args))
		},
	}
	openOn(cmd, &o)
	// switch alone declines a claim; add has no ticket to decline.
	cmd.Flags().BoolVar(&o.noClaim, "no-claim", false, "create the worktree without claiming the ticket; the vetting still applies")

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
	e, err := env.Enter(c, work.Options{Action: o.action(), Ask: ask, NoClaim: o.noClaim})
	if err != nil {
		return err
	}
	return e.Handoff.Exec()
}
