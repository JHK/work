package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// listCommand asks git and no integration beyond it.
func listCommand(branches func() ([]worktree.Name, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Print the repository's open worktrees",
		Long: `Print the repository's open worktrees, one per line: the branch each has checked
out, or its directory where it has none.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			list, err := branches()
			if err != nil {
				return err
			}
			for _, branch := range list {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), branch); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func (v verbs) branches() ([]worktree.Name, error) {
	return within(v, work.Env.Branches)
}
