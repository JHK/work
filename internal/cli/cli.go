// Package cli is work's headless front end: it turns flags into a decision and
// reports what it found. It is one of two front ends over [work.Env];
// everything it can trigger stays reachable without a screen.
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
	start  bool
	shell  bool
	model  string
	effort string
}

// Execute runs work and returns the process exit status.
func Execute() int {
	if err := command(enter).Execute(); err != nil {
		if !errors.Is(err, errCancelled) {
			fmt.Fprintln(os.Stderr, "work:", err)
		}
		return 1
	}
	return 0
}

// command builds the command tree, calling run once the flags are valid.
func command(run func(o options, target string) error) *cobra.Command {
	var o options

	cmd := &cobra.Command{
		Use:   "work [<id>|<pr>|<url>]",
		Short: "Enter a bead or pull request worktree, creating it if needed",
		Long: `Turn a ticket, a pull request, or an open worktree into a git worktree
with a coding-agent session inside it.

With no identifier, pick from the repository's worktrees and ready beads.
Nothing is created until you confirm it.`,
		Args: cobra.MaximumNArgs(1),
		// A failure to enter is one line on stderr, not a wall of usage.
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			if err := validate(o); err != nil {
				return err
			}
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			return run(o, target)
		},
	}

	f := cmd.Flags()
	f.BoolVar(&o.start, "start", false, "vet the bead, claim it, and launch a session on /start")
	f.BoolVar(&o.shell, "shell", false, "enter the worktree without claiming the bead")
	f.StringVar(&o.model, "model", "", "model for the launched session")
	f.StringVar(&o.effort, "effort", "", "effort for the launched session (low|medium|high|xhigh|max)")
	cmd.MarkFlagsMutuallyExclusive("start", "shell")

	return cmd
}

func validate(o options) error {
	if !o.start && (o.model != "" || o.effort != "") {
		return errors.New("--model and --effort apply to --start")
	}
	return nil
}
