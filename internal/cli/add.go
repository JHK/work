package cli

import (
	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// A tab press after add offers the places with no worktree yet.
func addCommand(verb offering[opens]) *cobra.Command {
	return opening(&cobra.Command{
		Use:   "add [<name>|<id>|<pr>|<url>]",
		Short: "Create the worktree an identifier names and open it",
		Long: `Create the worktree an identifier names, forked from the checkout you are
standing in. A ticket is vetted and claimed, a pull request is checked out, and
a name no system answers for becomes a branch spelled exactly as it is.

An identifier that already has a worktree is refused, work switch being what
enters one. The checkout is left as it stands, work carry being what takes its
changes along.

With no identifier, choose among the repository's ready tickets and open pull
requests that have no worktree yet. That form needs fzf.

The worktree opens on a Claude session where claude.on-creation names add, which
it does by default. One under a name of your own is handed back, the default
claude.command naming nothing to run for it.`,
	}, verb)
}

// add makes the worktree the identifier asks for and says what it opens on.
func add(env work.Env, l listing, verb, target string) (worktree.Handoff, error) {
	c, err := targeted(env, l, target, env.Add)
	if err != nil {
		return worktree.Handoff{}, err
	}
	return open(env, verb, c)
}
