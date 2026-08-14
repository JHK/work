package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
)

// moveCommand is the verb that moves a worktree and takes its branch with it. It
// carries no flag, and a tab press offers the worktrees at the name and nothing
// at the destination, which is a name nothing holds yet.
func moveCommand(run func(target, dest string) (work.Move, error), list func() ([]work.Candidate, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "move [<name>] [<destination>]",
		Short: "Move a worktree and rename its branch to the destination",
		Long: `Move a worktree's directory and rename the branch it has checked out to the
destination's last element. Both land or neither does.

A destination spelled as a bare name lands the worktree beside where it sits;
one carrying a path separator is a path, read from where you are standing.

With no destination you are asked for one, the directory's current name already
in it; with no name either, choose among the repository's worktrees first. Both
forms need fzf.

No tracker is asked and nothing opens. The worktree you are standing in cannot
be moved, and neither can the main checkout.`,
		Args:              cobra.MaximumNArgs(2),
		ValidArgsFunction: suggest(list),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := run(arg(args, 0), arg(args, 1))
			if err != nil {
				return err
			}
			return moved(cmd.OutOrStdout(), m)
		},
	}
}

// move moves a worktree and renames its branch with it. It hands over to
// nothing: the invocation does its work and exits.
func move(env work.Env, target, dest string) (work.Move, error) {
	c, err := openWorktree(env, target)
	if err != nil {
		return work.Move{}, err
	}
	if dest == "" {
		// The directory rather than the name, which for a plain worktree is its branch
		// and may carry a separator the destination would read as a path.
		if dest, err = ask(c.Dir()); err != nil {
			return work.Move{}, err
		}
	}
	return env.Move(c, dest)
}

// moved prints what moved: the worktree, and the branch where the destination
// renamed one.
func moved(out io.Writer, m work.Move) error {
	said := fmt.Sprintf("moved worktree %s to %s\n", m.From, m.To)
	if m.Renamed() {
		said += fmt.Sprintf("renamed branch %s to %s\n", m.Was, m.Now)
	}
	_, err := io.WriteString(out, said)
	return err
}
