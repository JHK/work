// Package cli is work's headless front end: it turns flags into one call on
// [work.Env] and prints what came back. It is one of two front ends over that
// core; everything it can trigger stays reachable without a screen.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// errCancelled reports a choice the user declined to make. It exits non-zero
// without a message: nothing happened, and nothing went wrong.
var errCancelled = errors.New("cancelled")

// options are what the flags settled, in the names the systems go by rather than
// a field per system: which action the worktree opens on, and which of the ones
// that would otherwise run were declined.
type options struct {
	open string
	skip []string
}

// front is what the command tree calls once cobra has read the flags: a
// function per verb, and a listing per source for the verbs that complete.
type front struct {
	enter      func(o options, target string) error
	add        func(o options, name string) error
	remove     func(force bool, target string) error
	dump       func(out io.Writer) error
	edit       func() error
	candidates func() ([]work.Candidate, error)
	worktrees  func() ([]work.Candidate, error)
}

// verbs answers the front's calls against the wiring Execute was handed, so that
// the systems reach the core through the call that needs them and cmd/work stays
// the only place they are named.
type verbs struct{ wire work.Wiring }

// repository is the one the shell stands in, with its systems wired.
func (v verbs) repository() (work.Env, error) {
	return work.Open(".", v.wire)
}

// performEnter brings a worktree into being in the repository the shell stands
// in and hands the terminal over to it.
func (v verbs) performEnter(o options, target string) error {
	env, err := v.repository()
	if err != nil {
		return err
	}
	return enter(env, o, target)
}

// performAdd brings a worktree of the user's own name into being in the
// repository the shell stands in and hands the terminal over to it.
func (v verbs) performAdd(o options, name string) error {
	env, err := v.repository()
	if err != nil {
		return err
	}
	return add(env, o, name)
}

// performRemove takes a worktree out of the repository the shell stands in.
func (v verbs) performRemove(force bool, target string) error {
	env, err := v.repository()
	if err != nil {
		return err
	}
	return remove(env, force, target)
}

// Execute runs work and returns the process exit status.
func Execute(version string, wire work.Wiring) int {
	v := verbs{wire: wire}
	f := front{enter: v.performEnter, add: v.performAdd, remove: v.performRemove, dump: dump, edit: v.edit, candidates: v.listing, worktrees: v.worktreeListing}
	if err := run(command(version, naming(wire), f), os.Args[1:]); err != nil {
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

// naming is the systems as the command line spells them. A flag set is settled
// before there is a repository to read, so the wiring is asked without one, which
// [work.Wiring] is what holds it to: nothing is taken from what comes back but
// the names the systems go by and the flags they answer to, which no repository
// decides.
//
// It asks for every system work ships rather than the ones a repository enabled,
// so --help reads the same everywhere and a flag whose system is off is refused
// with the key that puts it back rather than read as a flag work never had.
func naming(wire work.Wiring) work.Systems {
	return wire("", "", config.Shipped())
}

// command builds the command tree, calling the verb front once the flags are
// valid and answering a tab press from its listings. The root runs nothing
// itself: [dispatch] has already sent the bare form to switch.
func command(version string, sys work.Systems, f front) *cobra.Command {
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
	cmd.AddCommand(initCommand(), switchCommand(sys, f.enter, f.candidates), addCommand(sys, f.add), removeCommand(f.remove, f.worktrees), listCommand(f.worktrees), configCommand(f.dump, f.edit))
	// Cobra adds these three as it runs, too late for [dispatch] to read them.
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()
	cmd.InitDefaultVersionFlag()

	return cmd
}

// openOn gives a command the flags that name what a worktree opens on: one per
// action wired, under the spelling that action answers to, and one more for the
// screen, whose name no action goes by. Every verb that opens something carries
// the same set, so one place declares them and one place excludes them against
// each other.
func openOn(cmd *cobra.Command, o *options, openers []work.Opener) {
	exclusive := make([]string, 0, len(openers)+1)
	for _, op := range openers {
		name, usage := spelling(op)
		exclusive = append(exclusive, name)
		boolFlag(cmd, name, usage, func() { o.open = op.Name() })
		// What was renamed is the action; the refusal names the flag that reaches it
		// today, which is the spelling it answers to rather than the name behind it.
		for _, was := range config.Renamed(config.ActionName(op.Name())) {
			renamedFlag(cmd, string(was), name)
		}
	}
	screen := string(config.ActionAsk)
	exclusive = append(exclusive, screen)
	boolFlag(cmd, screen, "choose what the worktree opens on from the actions that apply", func() { o.open = screen })
	cmd.MarkFlagsMutuallyExclusive(exclusive...)
}

// decline gives a command the flags that call off an action that would otherwise
// run. Every action runs, so what an action's own flag can say is that this
// worktree is not its business, which is the other way round from an opener's.
// An action spelling no flag runs whenever a worktree comes into being.
func decline(cmd *cobra.Command, o *options, actions []work.Action) {
	for _, a := range actions {
		f, ok := a.(worktree.Flagged)
		if !ok {
			continue
		}
		name, usage := f.Flag()
		boolFlag(cmd, name, usage, func() { o.skip = append(o.skip, a.Name()) })
	}
}

// spelling is the flag an opener answers to and the line --help shows for it. An
// opener that spells neither is named by the flag its own name spells, there
// being nothing else it is called.
func spelling(op work.Opener) (name, usage string) {
	if f, ok := op.(worktree.Flagged); ok {
		return f.Flag()
	}
	return op.Name(), ""
}

// boolFlag declares a flag that stands for itself and calls set where it was
// given. A value spelled out is read, so a flag given false settles nothing, as
// a plain bool flag does.
func boolFlag(cmd *cobra.Command, name, usage string, set func()) {
	cmd.Flags().BoolFunc(name, usage, func(v string) error {
		on, err := strconv.ParseBool(v)
		if err != nil {
			return err
		}
		if on {
			set()
		}
		return nil
	})
}

// renamedFlag keeps the spelling an action used to answer to, refused with the
// one it answers to now rather than left to read as a flag work never had. It is
// hidden, so nothing offers a name that only fails.
func renamedFlag(cmd *cobra.Command, was, now string) {
	cmd.Flags().BoolFunc(was, "", func(string) error {
		return fmt.Errorf("this flag is now --%s", now)
	})
	// The error is a flag that was not declared, and one just was.
	_ = cmd.Flags().MarkHidden(was)
}

// firstArg is the name a verb was given, empty where it was left out for the
// picker. Cobra has already refused a second.
func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
