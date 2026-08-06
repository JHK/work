// Package cli is work's headless front end: it turns flags into one call on
// [work.Env] and prints what came back. It is one of two front ends over that
// core; everything it can trigger stays reachable without a screen.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// errCancelled reports a choice the user declined to make. It exits non-zero
// without a message: nothing happened, and nothing went wrong.
var errCancelled = errors.New("cancelled")

type options struct {
	shell  bool
	model  string
	effort string
}

// Execute runs work and returns the process exit status.
func Execute(version string) int {
	if err := command(version, enter).Execute(); err != nil {
		if !errors.Is(err, errCancelled) {
			fmt.Fprintln(os.Stderr, "work:", err)
		}
		return 1
	}
	return 0
}

// command builds the command tree, calling run once the flags are valid.
func command(version string, run func(o options, target string) error) *cobra.Command {
	var o options

	cmd := &cobra.Command{
		Use:   "work [<id>|<pr>|<url>]",
		Short: "A smarter cd for git worktrees",
		Long: `work is a smarter cd for git worktrees. It knows which worktrees a repository
has open and which tickets and pull requests are waiting, and hands the terminal
to what the one you pick opens on: your shell, or a command.

With no identifier, choose among the repository's worktrees, its ready tickets
and its open pull requests. That form needs fzf.

Entering a worktree that already exists opens a shell in it and prints the lines
that resume the sessions it carries. A target without one has its worktree
created, its ticket claimed, and the launcher invoked in it.`,
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
	}

	f := cmd.Flags()
	f.BoolVar(&o.shell, "shell", false, "create the worktree without claiming the ticket or launching a session")
	f.StringVar(&o.model, "model", "", "model for the launched session")
	f.StringVar(&o.effort, "effort", "", "effort for the launched session (low|medium|high|xhigh|max)")

	return cmd
}
