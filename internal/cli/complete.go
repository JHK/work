package cli

import (
	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
)

// listing is where a tab press on the identifier gets its rows: the same
// candidates the picker renders.
func listing() ([]work.Candidate, error) {
	env, err := work.Open(".")
	if err != nil {
		return nil, err
	}
	return env.Candidates()
}

// worktreeListing is what a tab press after remove gets: the repository's
// worktrees alone, there being nothing else to remove.
func worktreeListing() ([]work.Candidate, error) {
	env, err := work.Open(".")
	if err != nil {
		return nil, err
	}
	return env.Worktrees()
}

// suggest answers a tab press on a verb's argument. There is only one, so a
// second word completes nothing, and a repository that will not answer
// completes nothing rather than spilling an error into the shell.
func suggest(list func() ([]work.Candidate, error)) cobra.CompletionFunc {
	return func(_ *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
		var candidates []work.Candidate
		if len(args) == 0 {
			candidates, _ = list()
		}
		return completions(candidates), cobra.ShellCompDirectiveNoFileComp
	}
}

// completions renders candidates the way the shell reads them: the name to
// complete, then its title in the description column.
func completions(candidates []work.Candidate) []cobra.Completion {
	out := make([]cobra.Completion, len(candidates))
	for i, c := range candidates {
		if c.Label == "" {
			out[i] = c.Target.Name
			continue
		}
		out[i] = cobra.CompletionWithDesc(c.Target.Name, c.Label)
	}
	return out
}

// initCommand prints the shell integration. It runs at every shell start, so it
// reaches for nothing.
func initCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init fish",
		Short: "Print the shell integration to source",
		Long: `Print the shell integration for fish. Source it from config.fish:

    work init fish | source

It completes the identifier with the repository's worktrees, its open pull
requests and its ready tickets, each with its title beside it.`,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []cobra.Completion{cobra.CompletionWithDesc("fish", "fish shell integration")},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
		},
	}
}
