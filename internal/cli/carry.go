package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// carryCommand is the verb that brings a worktree into being and takes the
// checkout's work along.
func carryCommand(sys work.Systems, verb opens) *cobra.Command {
	return handing(&cobra.Command{
		Use:   "carry <name>",
		Short: "Create the worktree and take the checkout's changes into it",
		Long: `Create a worktree under a name of your own, forked from the checkout you are
standing in, and move that checkout's working state into it: the checkout is
left clean on the branch it was already on, and the new worktree carries the
changes, what was staged still staged. Untracked files travel; ignored files
stay put.

A name that already has a worktree is refused, and so is a checkout carrying
nothing: work add is what only creates.

The worktree opens on what action.create names.`,
		Args: cobra.ExactArgs(1),
	}, sys, sys.Actions, verb)
}

// carrying puts carry over the repository the shell stands in.
func (v verbs) carrying() opens {
	return func(o options, name string) (worktree.Handoff, error) {
		env, err := v.repository()
		if err != nil {
			return worktree.Handoff{}, err
		}
		return carry(env, o, name)
	}
}

// carry makes the worktree the name asks for and moves the checkout's working
// state into it.
func carry(env work.Env, o options, name string) (worktree.Handoff, error) {
	if err := env.Carryable(); err != nil {
		return worktree.Handoff{}, fmt.Errorf("%w; work add %s makes the worktree and carries nothing", err, name)
	}
	c, err := env.Own(name)
	if err != nil {
		return worktree.Handoff{}, err
	}
	o.carry = true
	return open(env, o, c)
}
