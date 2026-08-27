package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// listCommand is the verb that prints the worktrees open, asking git and no
// system beyond it.
func listCommand(branches func() ([]string, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Print the repository's open worktrees",
		Long: `Print the repository's open worktrees, one per line: the branch each has checked
out, or its directory where it has none.`,
		Args: cobra.NoArgs,
		// It hands over to nothing: the invocation prints and exits.
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

// branchListing is what list prints: the repository's worktrees as git reports them.
func (v verbs) branchListing() ([]string, error) {
	env, err := v.repository()
	if err != nil {
		return nil, err
	}
	return env.Branches()
}
