package cli

import (
	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// addCommand is the verb that brings a worktree into being. A tab press after it
// offers the places with no worktree yet.
func addCommand(sys work.Systems, verb offering[opens]) *cobra.Command {
	return opening(&cobra.Command{
		Use:   "add [<name>|<id>|<pr>|<url>]",
		Short: "Create the worktree an identifier names and open it",
		Long: `Create the worktree an identifier names, forked from the checkout you are
standing in. A ticket is vetted and claimed, a pull request is checked out, and
a name no system answers for becomes a branch spelled exactly as it is.

An identifier that already has a worktree is refused, work switch being what
enters one.

A checkout carrying changes hands them over: it is left clean on the branch it
was already on, and the new worktree carries them, what was staged still staged.
Untracked files travel; ignored files stay put.

With no identifier, choose among the repository's ready tickets and open pull
requests that have no worktree yet. That form needs fzf.

The worktree opens on what action.create names.`,
	}, sys, sys.Actions, verb)
}

// add makes the worktree the identifier asks for and says what it opens on,
// carrying what the checkout was working on into it.
func add(env work.Env, l listing, o options, id string) (worktree.Handoff, error) {
	c, err := offered(env, l, id, env.Add)
	if err != nil {
		return worktree.Handoff{}, err
	}
	o.park = true
	return open(env, o, c)
}
