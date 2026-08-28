package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// carryCommand is the verb that brings a worktree into being and takes the
// checkout's work along.
func carryCommand(verb opens) *cobra.Command {
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

The worktree is handed back whatever claude.on-creation names: a name of your
own opens on no session.`,
		Args: cobra.ExactArgs(1),
	}, verb)
}

// carrying puts carry over the repository the shell stands in.
func (v verbs) carrying() opens {
	return func(verb, name string) (worktree.Handoff, error) {
		env, err := v.repository()
		if err != nil {
			return worktree.Handoff{}, err
		}
		return carry(env, verb, name)
	}
}

// carry makes the worktree the name asks for and moves the checkout's working
// state into it.
func carry(env work.Env, verb, name string) (worktree.Handoff, error) {
	if err := env.Carryable(); err != nil {
		return worktree.Handoff{}, fmt.Errorf("%w; work add %s makes the worktree and carries nothing", err, name)
	}
	c, err := env.Own(name)
	if err != nil {
		return worktree.Handoff{}, err
	}
	return env.Enter(c, work.Options{Verb: verb, Carry: true})
}
