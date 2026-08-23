package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// goCommand is the verb that reaches a place whatever state it is in: it enters
// the worktree an identifier already has and creates the one it does not find.
// It is what the bare form is a shortcut for.
func goCommand(sys work.Systems, verb offering[opens]) *cobra.Command {
	return opening(&cobra.Command{
		Use:   "go [<name>|<id>|<pr>|<url>]",
		Short: "Reach the worktree an identifier names, creating it if there is none",
		Long: `Reach the worktree an identifier names, creating it if there is none. An
identifier is a worktree name, a ticket id or a pull request.

With no identifier, choose among the repository's worktrees, less the one you are
standing in, its ready tickets and its open pull requests. That form needs fzf.

An existing worktree opens on what action.enter names, a new one on what
action.create names.

Creating a worktree for a ticket vets that ticket and claims it. A name no
system answers for is work add's, this verb reaching what its picker offers and
nothing else.

work <identifier> is this same command without the verb: work go add reaches the
worktree add.`,
	}, sys, sys.Actions, verb)
}

// reach resolves the target and asks work to bring its worktree into being, and
// is the one verb whose refusal names add.
func reach(env work.Env, l listing, o options, target string) (worktree.Handoff, error) {
	c, err := offered(env, l, target, env.Resolve)
	// A spelling add would refuse too is advice that goes nowhere.
	if work.Unanswered(err) && work.Nameable(target) {
		return worktree.Handoff{}, fmt.Errorf("%w; work add %s makes a worktree of it", err, target)
	}
	if err != nil {
		return worktree.Handoff{}, err
	}
	return open(env, o, c)
}
