package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
)

// removeCommand is the verb that takes a worktree away. It carries --force and
// nothing else, and a tab press after it offers the worktrees alone.
func removeCommand(run func(force bool, target string) (work.Deletion, error), list func() ([]work.Candidate, error)) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove [<name>]",
		Short: "Remove a worktree and delete the branch it had checked out",
		Long: `Remove a worktree: git removes it and deletes the branch it had checked out. No
tracker is asked.

The worktree and its branch go together or neither goes. A worktree with modified
or untracked files is refused, and so is a branch whose work has not landed;
--force takes both. The worktree you are standing in is refused either way.

With no name, choose among the repository's worktrees. That form needs fzf.`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: suggest(list),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := run(force, firstArg(args))
			if err != nil {
				return err
			}
			return removed(cmd.OutOrStdout(), d)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "take an unclean worktree or an unmerged branch")

	return cmd
}

// remove takes a worktree away and returns what went. It hands over to nothing:
// the invocation does its work and exits.
func remove(env work.Env, force bool, target string) (work.Deletion, error) {
	c, err := toRemove(env, target)
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

// toRemove is the worktree to take away. The picker offers the open ones alone,
// removing reaching nothing else.
func toRemove(env work.Env, target string) (work.Candidate, error) {
	if target == "" {
		return pickFrom(env.Worktrees, "no worktrees to remove")
	}
	return env.Resolve(target)
}
