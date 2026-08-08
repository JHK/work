// Package cli is work's headless front end: it turns flags into one call on
// [work.Env] and prints what came back. It is one of two front ends over that
// core; everything it can trigger stays reachable without a screen.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
)

// errCancelled reports a choice the user declined to make. It exits non-zero
// without a message: nothing happened, and nothing went wrong.
var errCancelled = errors.New("cancelled")

type options struct {
	agent   bool
	shell   bool
	editor  bool
	diff    bool
	create  bool
	noClaim bool
}

// action is the one the flags named. Cobra has already refused two at once, so
// the order here only settles what nothing can ask for.
func (o options) action() work.Action {
	switch {
	case o.agent:
		return work.ActionAgent
	case o.shell:
		return work.ActionShell
	case o.editor:
		return work.ActionEditor
	case o.diff:
		return work.ActionDiff
	}
	return work.ActionUnnamed
}

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

Entering a worktree that already exists runs open.shell in it, your shell by
default. A target without one has its worktree created and the configured
launcher invoked in it. --agent, --shell, --editor and --diff name the command
to open on instead.

--agent hands the worktree to its agent. It changes nothing for one just
created, which opens on the launcher regardless; an existing one is handed over
by what it carries, no conversation starting one, a single one being returned
to, and several reaching the agent's own list.

Creating a worktree for a ticket vets that ticket and claims it, whatever the
worktree then opens on. A ticket the vetting refuses is refused outright;
--no-claim declines the claim and nothing else.

--create takes the identifier as a name of its own, guessing nothing and asking
no tracker: a worktree on a new branch spelled exactly that way, forked from the
main checkout. Re-entering it later is the same name without the flag.`,
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
	f.BoolVar(&o.agent, "agent", false, "hand the worktree to its agent; an existing one starts, resumes or lists by what it carries")
	f.BoolVar(&o.shell, "shell", false, "hand the worktree to open.shell, your login shell by default, instead of launching a session")
	f.BoolVar(&o.editor, "editor", false, "hand the worktree to open.editor, $VISUAL else $EDITOR by default, instead of a session or a shell")
	f.BoolVar(&o.diff, "diff", false, "hand the worktree to open.diff, git diff against the point its branch forked from by default, instead of a session or a shell")
	f.BoolVar(&o.create, "create", false, "take the identifier as a worktree name of its own, on a new branch spelled the same way")
	f.BoolVar(&o.noClaim, "no-claim", false, "create the worktree without claiming the ticket; the vetting still applies")
	cmd.MarkFlagsMutuallyExclusive("agent", "shell", "editor", "diff")
	// A worktree with no ticket behind it has no claim to decline.
	cmd.MarkFlagsMutuallyExclusive("create", "no-claim")

	return cmd
}
