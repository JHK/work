package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
)

// removeCommand is the verb that takes a worktree away. It carries --force and
// nothing else, and a tab press after it offers what its picker offers.
func removeCommand(run func(force bool, target string) (work.Deletion, error), list func() ([]work.Candidate, error)) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove [<name>]",
		Short: "Remove a worktree and delete the branch it had checked out",
		Long: `Remove a worktree and delete the branch it had checked out.

Only a clean worktree can be removed, an unclean one with --force. The main
checkout cannot be removed, and neither can the worktree you are standing in.

With no name, choose among the repository's worktrees, less the one you are
standing in and the main checkout. That form needs fzf.`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: suggest(list),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := run(force, arg(args, 0))
			if err != nil {
				return err
			}
			return removed(cmd.OutOrStdout(), d)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "take an unclean worktree")

	return cmd
}

// remove takes a worktree away and returns what went. It hands over to nothing:
// the invocation does its work and exits.
func remove(env work.Env, force bool, target string) (work.Deletion, error) {
	c, err := openWorktree(env, target, "no worktree to remove", env.Removable)
	if err != nil {
		return work.Deletion{}, err
	}
	return env.Delete(c, force)
}

// removed prints what went: the worktree, and the branch where it had one.
func removed(out io.Writer, d work.Deletion) error {
	said := fmt.Sprintln("removed worktree", d.Path)
	if d.Branch != "" {
		said += fmt.Sprintln("deleted branch", d.Branch)
	}
	_, err := io.WriteString(out, said)
	return err
}
