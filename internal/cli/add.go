package cli

import (
	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
)

// addCommand is the verb that brings a worktree of the user's own name into
// being. It carries the open-on flags, opening something, and a tab press after
// it offers nothing, the name being new.
func addCommand(sys work.Systems, run func(o options, name string) error) *cobra.Command {
	var o options

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a worktree on a new branch of that name and open it",
		Long: `Create a worktree on a new branch spelled exactly as the name is, forked from
the checkout the shell is standing in. No tracker is asked, and a branch already
holding the name is refused.

The worktree opens on what action.create names. Re-enter it later with the name
alone.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return run(o, args[0])
		},
	}
	openOn(cmd, &o, sys.Openers)

	return cmd
}

// add makes the worktree the name asks for and hands the terminal over to it.
func add(env work.Env, o options, name string) error {
	c, err := env.Add(name)
	if err != nil {
		return err
	}
	return open(env, o, c)
}
