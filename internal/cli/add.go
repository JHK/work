package cli

import (
	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// addCommand is the verb that brings a worktree into being. A tab press after it
// offers the places with no worktree yet.
func addCommand(sys work.Systems, run func(o options, id string) (worktree.Handoff, error), list func() ([]work.Candidate, error)) *cobra.Command {
	return opening(&cobra.Command{
		Use:   "add [<name>|<id>|<pr>|<url>]",
		Short: "Create the worktree an identifier names and open it",
		Long: `Create the worktree an identifier names, forked from the checkout you are
standing in. A ticket is vetted and claimed, a pull request is checked out, and
a name no system answers for becomes a branch spelled exactly as it is.

An identifier that already has a worktree is refused, work switch being what
enters one.

With no identifier, choose among the repository's ready tickets and open pull
requests that have no worktree yet. That form needs fzf.

The worktree opens on what action.create names.`,
	}, sys, sys.Actions, run, list)
}

// add makes the worktree the identifier asks for and says what it opens on.
func add(env work.Env, o options, id string) (worktree.Handoff, error) {
	c, err := offered(id, env.Addable, env.Add)
	if err != nil {
		return worktree.Handoff{}, err
	}
	return open(env, o, c)
}
