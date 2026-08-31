package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// go is what the bare form is a shortcut for.
func goCommand(verb offering[opens]) *cobra.Command {
	return opening(&cobra.Command{
		Use:   "go [<name>|<id>|<pr>|<url>]",
		Short: "Reach the worktree an identifier names, creating it if there is none",
		Long: `Reach the worktree an identifier names, creating it if there is none. An
identifier is a worktree name, a ticket id or a pull request.

With no identifier, choose among the repository's worktrees, less the one you are
standing in, its ready tickets and its open pull requests. That form needs fzf.

A worktree already open is handed back. One this run creates opens on a Claude
session where claude.on-creation names go, which it does by default.

Creating a worktree for a ticket vets that ticket and claims it. A name no
system answers for is work add's.

work <identifier> is work go <identifier>: work go add reaches the worktree add.`,
	}, verb)
}

// reach resolves the target and asks work to bring its worktree into being, and
// is the one verb whose refusal names add.
func reach(env work.Env, l listing, verb, target string) (worktree.Handoff, error) {
	c, err := targeted(env, l, target, env.Resolve)
	// A spelling add would refuse too is advice that goes nowhere.
	if work.Unanswered(err) && work.Nameable(target) {
		return worktree.Handoff{}, fmt.Errorf("%w; work add %s makes a worktree of it", err, target)
	}
	if err != nil {
		return worktree.Handoff{}, err
	}
	return open(env, verb, c)
}
