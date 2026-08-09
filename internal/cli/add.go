package cli

import (
	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
)

// addCommand is the verb that brings a worktree of the user's own name into
// being. It carries the open-on flags, opening something, and a tab press after
// it offers nothing, the name being new.
func addCommand(run func(o options, name string) error) *cobra.Command {
	var o options

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a worktree on a new branch of that name and open it",
		Long: `Create a worktree on a new branch spelled exactly as the name is, forked from
the main checkout. Nothing is guessed and no tracker is asked, so the name is
the user's own: a branch already holding it is a worktree to enter rather than
create, and is refused.

The worktree opens on what action.create names, the configured launcher by
default. --agent, --shell, --editor, --diff and --ask name the action to open on
instead, for that invocation.

Re-entering the worktree later is the same name without the verb, or work
switch <name> where a verb holds that name.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return run(o, args[0])
		},
	}
	openOn(cmd, &o)

	return cmd
}

// add makes the worktree the name asks for and hands the terminal over to it.
func add(env work.Env, o options, name string) error {
	c, err := env.Create(name)
	if err != nil {
		return err
	}
	return open(env, o, c)
}
