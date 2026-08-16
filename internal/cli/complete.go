package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/shim"
	"github.com/JHK/work-cli/internal/work"
)

// listing is where a tab press on the identifier gets its rows: the same
// candidates the picker renders.
func (v verbs) listing() ([]work.Candidate, error) {
	env, err := v.repository()
	if err != nil {
		return nil, err
	}
	// The rows alone: a refusal on stderr would land in the middle of the shell
	// drawing its completions.
	list, _, err := env.Candidates()
	return list, err
}

// worktreeListing is what a tab press after remove gets: the repository's
// worktrees alone, there being nothing else to remove.
func (v verbs) worktreeListing() ([]work.Candidate, error) {
	env, err := v.repository()
	if err != nil {
		return nil, err
	}
	return env.Worktrees()
}

// suggest answers a tab press on a verb's argument. A second word completes
// nothing, and so does a repository that will not answer.
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
			out[i] = c.Name
			continue
		}
		out[i] = cobra.CompletionWithDesc(c.Name, c.Label)
	}
	return out
}

// integration is a shell work prints for.
type integration struct {
	shell      string
	function   string
	completion func(root *cobra.Command, out io.Writer, desc bool) error
}

// integrations are the shells [work init] answers to.
var integrations = []integration{
	{"bash", shim.Bash, (*cobra.Command).GenBashCompletionV2},
	{"fish", shim.Fish, (*cobra.Command).GenFishCompletion},
}

// initCommand prints the shell integration. It runs at every shell start, so it
// reaches for nothing.
func initCommand() *cobra.Command {
	valid := make([]cobra.Completion, len(integrations))
	for i, in := range integrations {
		valid[i] = cobra.CompletionWithDesc(in.shell, in.shell+" shell integration")
	}
	return &cobra.Command{
		Use:   "init <shell>",
		Short: "Print the shell integration to source",
		Long: `Print the shell integration for bash or fish. Source it from the shell's own
startup file:

    source <(work init bash)    # .bashrc
    work init fish | source     # config.fish

It puts a work function in front of the binary, which is what changes the shell
into the worktree, and completes the commands and each verb's argument.`,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: valid,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			for _, in := range integrations {
				if in.shell != args[0] {
					continue
				}
				if _, err := io.WriteString(out, in.function); err != nil {
					return err
				}
				return in.completion(cmd.Root(), out, true)
			}
			return fmt.Errorf("no %s integration", args[0])
		},
	}
}
