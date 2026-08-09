// Package cli is work's headless front end: it turns flags into one call on
// [work.Env] and prints what came back. It is one of two front ends over that
// core; everything it can trigger stays reachable without a screen.
package cli

import (
	"errors"
	"fmt"
	"io"
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
	ask     bool
	noClaim bool
}

// front is what the command tree calls once cobra has read the flags: a
// function per verb, and a listing per source for the verbs that complete.
type front struct {
	enter      func(o options, target string) error
	add        func(o options, name string) error
	remove     func(force bool, target string) error
	dump       func(out io.Writer) error
	candidates func() ([]work.Candidate, error)
	worktrees  func() ([]work.Candidate, error)
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
	case o.ask:
		return work.ActionAsk
	}
	return work.ActionUnnamed
}

// performEnter brings a worktree into being in the repository the shell stands
// in and hands the terminal over to it.
func performEnter(o options, target string) error {
	env, err := work.Open(".")
	if err != nil {
		return err
	}
	return enter(env, o, target)
}

// performAdd brings a worktree of the user's own name into being in the
// repository the shell stands in and hands the terminal over to it.
func performAdd(o options, name string) error {
	env, err := work.Open(".")
	if err != nil {
		return err
	}
	return add(env, o, name)
}

// performRemove takes a worktree out of the repository the shell stands in.
func performRemove(force bool, target string) error {
	env, err := work.Open(".")
	if err != nil {
		return err
	}
	return remove(env, force, target)
}

// Execute runs work and returns the process exit status.
func Execute(version string) int {
	f := front{enter: performEnter, add: performAdd, remove: performRemove, dump: dump, candidates: listing, worktrees: worktreeListing}
	if err := run(command(version, f), os.Args[1:]); err != nil {
		if !errors.Is(err, errCancelled) {
			fmt.Fprintln(os.Stderr, "work:", err)
		}
		return 1
	}
	return 0
}

// run puts args through the tree. The dispatch belongs to every way in, so the
// tree is never executed on the words as typed.
func run(cmd *cobra.Command, args []string) error {
	cmd.SetArgs(dispatch(cmd, args))
	return cmd.Execute()
}

// command builds the command tree, calling the verb front once the flags are
// valid and answering a tab press from its listings. The root runs nothing
// itself: [dispatch] has already sent the bare form to switch.
func command(version string, f front) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "work",
		Short: "A smarter cd for git worktrees",
		Long: `work is a smarter cd for git worktrees. It knows which worktrees a repository
has open and which tickets and pull requests are waiting, and hands the terminal
to what the one you pick opens on: your shell, or a command.

An identifier in the first position, or none at all, is work switch.`,
		Version: version,
		// A failure to enter is one line on stderr, not a wall of usage.
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// One documented door to the shell integration: work init fish.
	cmd.CompletionOptions.DisableDefaultCmd = true
	// Every position cobra would otherwise answer with a file listing, the
	// subcommands' arguments included, answers with nothing instead.
	cmd.CompletionOptions.SetDefaultShellCompDirective(cobra.ShellCompDirectiveNoFileComp)
	cmd.AddCommand(initCommand(), switchCommand(f.enter, f.candidates), addCommand(f.add), removeCommand(f.remove, f.worktrees), listCommand(f.worktrees), configCommand(f.dump))
	// Cobra adds these three as it runs, too late for [dispatch] to read them.
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()
	cmd.InitDefaultVersionFlag()

	return cmd
}

// openOn gives a command the flags that name what a worktree opens on. Every
// verb that opens something carries the same set, so one place declares them and
// one place excludes them against each other.
func openOn(cmd *cobra.Command, o *options) {
	flags := cmd.Flags()
	flags.BoolVar(&o.agent, "agent", false, "hand the worktree to its agent")
	flags.BoolVar(&o.shell, "shell", false, "hand the worktree to open.shell, your login shell by default")
	flags.BoolVar(&o.editor, "editor", false, "hand the worktree to open.editor, $VISUAL else $EDITOR by default")
	flags.BoolVar(&o.diff, "diff", false, "hand the worktree to open.diff, its diff against the fork point by default")
	flags.BoolVar(&o.ask, "ask", false, "choose what the worktree opens on from the actions that apply")
	cmd.MarkFlagsMutuallyExclusive("agent", "shell", "editor", "diff", "ask")
}

// firstArg is the name a verb was given, empty where it was left out for the
// picker. Cobra has already refused a second.
func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
