package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
)

// listCommand is the verb that prints the worktrees, over the same listing
// remove offers. It takes no argument and no flag: a name to filter on is
// switch's.
func listCommand(worktrees func() ([]work.Candidate, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Print the repository's open worktrees",
		Long: `Print the repository's open worktrees, one per line: the name you retype, then
the title of the bead behind it where bd names one.

With none open it prints nothing. No tracker but bd is asked.`,
		Args: cobra.NoArgs,
		// It hands over to nothing: the invocation prints and exits.
		RunE: func(cmd *cobra.Command, _ []string) error {
			candidates, err := worktrees()
			if err != nil {
				return err
			}
			for _, line := range lines(candidates) {
				fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			return nil
		},
	}
}

// lines renders the rows for a reader rather than a chooser: the picker's
// column, without what marks a choice being made.
func lines(candidates []work.Candidate) []string {
	width := column(candidates)
	out := make([]string, len(candidates))
	for i, c := range candidates {
		if c.Label == "" {
			out[i] = c.Target.Name
			continue
		}
		out[i] = fmt.Sprintf("%-*s  %s", width, c.Target.Name, c.Label)
	}
	return out
}
