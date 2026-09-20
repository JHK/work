package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

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

The worktree is handed back: the default claude.command opens no session on a
name of your own.`,
		Args: cobra.ExactArgs(1),
	}, verb)
}

// carrying puts carry over the repository the shell stands in.
func (v verbs) carrying() opens {
	return func(name string) (worktree.Handoff, error) {
		return within(v, func(env work.Env) (worktree.Handoff, error) { return carry(env, name) })
	}
}

// carry makes the worktree the name asks for and moves the checkout's working
// state into it.
func carry(env work.Env, name string) (worktree.Handoff, error) {
	if err := env.Carryable(); err != nil {
		return worktree.Handoff{}, fmt.Errorf("%w; work add %s makes the worktree and carries nothing", err, name)
	}
	c, err := env.Own(worktree.Name(name))
	if err != nil {
		return worktree.Handoff{}, err
	}
	return env.Enter(c, work.Options{Carry: true})
}
