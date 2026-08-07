// Package cli is work's headless front end: it turns flags into one call on
// [work.Env] and prints what came back. It is one of two front ends over that
// core; everything it can trigger stays reachable without a screen.
package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
)

// errCancelled reports a choice the user declined to make. It exits non-zero
// without a message: nothing happened, and nothing went wrong.
var errCancelled = errors.New("cancelled")

type options struct {
	shell  bool
	editor bool
	model  string
	effort string
}

// efforts is what --effort offers, and writes its usage line, so a tab press
// and --help cannot drift. Nothing checks the flag against it.
var efforts = []string{"low", "medium", "high", "xhigh", "max"}

// Execute runs work and returns the process exit status.
func Execute(version string) int {
	if err := command(version, enter, listing).Execute(); err != nil {
		if !errors.Is(err, errCancelled) {
			fmt.Fprintln(os.Stderr, "work:", err)
		}
		return 1
	}
	return 0
}

// command builds the command tree, calling run once the flags are valid and
// answering a tab press from list.
func command(version string, run func(o options, target string) error, list func() ([]work.Candidate, error)) *cobra.Command {
	var o options

	cmd := &cobra.Command{
		Use:   "work [<id>|<pr>|<url>]",
		Short: "A smarter cd for git worktrees",
		Long: `work is a smarter cd for git worktrees. It knows which worktrees a repository
has open and which tickets and pull requests are waiting, and hands the terminal
to what the one you pick opens on: your shell, or a command.

With no identifier, choose among the repository's worktrees, its ready tickets
and its open pull requests. That form needs fzf.

Entering a worktree that already exists opens a shell in it, naming the
conversations it carries, with one line to return to them. A target without one
has its worktree created, its ticket claimed, and the configured launcher
invoked in it.

--editor takes over from whichever of those two the target was headed for: an
existing worktree opens in the editor rather than the shell, and one that does
not exist yet is still vetted, created and claimed, then opened there rather
than launched into a session.`,
		Version: version,
		Args:    cobra.MaximumNArgs(1),
		// A failure to enter is one line on stderr, not a wall of usage.
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			return run(o, target)
		},
		ValidArgsFunction: suggest(list),
	}
	// One documented door to the shell integration: work init fish.
	cmd.CompletionOptions.DisableDefaultCmd = true
	// Every position cobra would otherwise answer with a file listing, the
	// subcommands' arguments included, answers with nothing instead.
	cmd.CompletionOptions.SetDefaultShellCompDirective(cobra.ShellCompDirectiveNoFileComp)
	cmd.AddCommand(initCommand())

	f := cmd.Flags()
	f.BoolVar(&o.shell, "shell", false, "create the worktree without claiming the ticket or launching a session")
	f.BoolVar(&o.editor, "editor", false, "open the worktree in $VISUAL, else $EDITOR, instead of a session or a shell")
	f.StringVar(&o.model, "model", "", "model for the launched session")
	f.StringVar(&o.effort, "effort", "", "effort for the launched session ("+strings.Join(efforts, "|")+")")
	cmd.MarkFlagsMutuallyExclusive("shell", "editor")

	// The agent behind --model is about to be configurable, so a fixed list would
	// rot. Registration fails only on a flag this function did not just declare.
	_ = cmd.RegisterFlagCompletionFunc("model", cobra.NoFileCompletions)
	_ = cmd.RegisterFlagCompletionFunc("effort", cobra.FixedCompletions(efforts, cobra.ShellCompDirectiveNoFileComp))

	return cmd
}
