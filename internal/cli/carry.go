package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// carryCommand is the verb that brings a worktree into being and takes the
// checkout's work along. It takes its identifier on the command line and offers
// no listing behind it.
func carryCommand(sys work.Systems, verb opens) *cobra.Command {
	return handing(&cobra.Command{
		Use:   "carry <name>|<id>|<pr>|<url>",
		Short: "Create the worktree an identifier names and take the checkout's changes into it",
		Long: `Create the worktree an identifier names, forked from the checkout you are
standing in, and move that checkout's working state into it: the checkout is
left clean on the branch it was already on, and the new worktree carries the
changes, what was staged still staged. Untracked files travel; ignored files
stay put.

A ticket is vetted and claimed, a pull request is checked out, and a name no
system answers for becomes a branch spelled exactly as it is.

An identifier that already has a worktree is refused, and so is a checkout
carrying nothing: work add is what only creates.

The identifier is named on the command line; there is no listing to pick from.

The worktree opens on what action.create names.`,
		Args: cobra.ExactArgs(1),
	}, sys, sys.Actions, verb)
}

// carrying puts carry over the repository the shell stands in.
func (v verbs) carrying() opens {
	return func(o options, id string) (worktree.Handoff, error) {
		env, err := v.repository()
		if err != nil {
			return worktree.Handoff{}, err
		}
		return carry(env, o, id)
	}
}

// carry makes the worktree the identifier asks for and moves the checkout's
// working state into it.
func carry(env work.Env, o options, id string) (worktree.Handoff, error) {
	// Ahead of the resolvers, so a checkout with nothing to hand over asks no
	// system anything.
	if err := env.Carryable(); err != nil {
		return worktree.Handoff{}, fmt.Errorf("%w; work add %s makes the worktree and carries nothing", err, id)
	}
	c, err := env.Add(id)
	if err != nil {
		return worktree.Handoff{}, err
	}
	o.carry = true
	return open(env, o, c)
}
