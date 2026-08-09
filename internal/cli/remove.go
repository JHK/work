package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
)

// removeCommand is the verb that takes a worktree away. It carries --force and
// nothing else, and a tab press after it offers the worktrees alone.
func removeCommand(run func(force bool, target string) error, list func() ([]work.Candidate, error)) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove [<name>]",
		Short: "Remove a worktree and delete the branch it had checked out",
		Long: `Remove a worktree: git removes it and deletes the branch it had checked out. No
tracker is asked.

A worktree with modified or untracked files and a branch not fully merged are
each refused; --force takes both. The worktree you are standing in is refused
either way.

With no name, choose among the repository's worktrees. That form needs fzf.`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: suggest(list),
		RunE: func(_ *cobra.Command, args []string) error {
			return run(force, firstArg(args))
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "take an unclean worktree or an unmerged branch")

	return cmd
}

// remove takes a worktree away and says what went. It hands over to nothing:
// the invocation does its work and exits.
func remove(env work.Env, force bool, target string) error {
	c, err := toRemove(env, target)
	if err != nil {
		return err
	}
	d, err := env.Delete(c, force)
	if err != nil {
		return err
	}
	fmt.Println("removed worktree", d.Path)
	if d.Branch != "" {
		fmt.Println("deleted branch", d.Branch)
	}
	return nil
}

// toRemove is the worktree to take away. The picker offers the open ones alone,
// removing reaching nothing else.
func toRemove(env work.Env, target string) (work.Candidate, error) {
	if target == "" {
		return pickFrom(env.Worktrees, "no worktrees to remove")
	}
	return env.Resolve(target)
}
