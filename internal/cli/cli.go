// Package cli is work's headless front end: it turns flags into one call on
// [work.Env] and prints what came back.
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

// options are what the flags settled, in the names the systems go by: which
// action the worktree opens on, and which of the others were declined.
type options struct {
	open string
	skip []string
}

// front is what the command tree calls once cobra has read the flags: a
// function per verb, and a listing per source for the verbs that complete.
type front struct {
	reach      func(o options, target string) (worktree.Handoff, error)
	enter      func(o options, target string) (worktree.Handoff, error)
	add        func(o options, id string) (worktree.Handoff, error)
	remove     func(force bool, target string) (work.Deletion, error)
	move       func(target, dest string) (work.Move, error)
	dump       func(out io.Writer) error
	edit       func(out io.Writer) error
	candidates func() ([]work.Candidate, error)
	enterable  func() ([]work.Candidate, error)
	removable  func() ([]work.Candidate, error)
	addable    func() ([]work.Candidate, error)
	branches   func() ([]string, error)
}

// verbs answers the front's calls against the wiring Execute was handed.
type verbs struct{ wire work.Wiring }

// repository is the one the shell stands in, with its systems wired.
func (v verbs) repository() (work.Env, error) {
	return work.Open(".", v.wire)
}

// performs puts one of the opening verbs over the repository the shell stands in.
func (v verbs) performs(verb func(env work.Env, o options, target string) (worktree.Handoff, error)) func(options, string) (worktree.Handoff, error) {
	return func(o options, target string) (worktree.Handoff, error) {
		env, err := v.repository()
		if err != nil {
			return worktree.Handoff{}, err
		}
		return verb(env, o, target)
	}
}

// performRemove takes a worktree out of the repository the shell stands in.
func (v verbs) performRemove(force bool, target string) (work.Deletion, error) {
	env, err := v.repository()
	if err != nil {
		return work.Deletion{}, err
	}
	return remove(env, force, target)
}

// performMove moves a worktree of the repository the shell stands in and takes
// its branch with it.
func (v verbs) performMove(target, dest string) (work.Move, error) {
	env, err := v.repository()
	if err != nil {
		return work.Move{}, err
	}
	return move(env, target, dest)
}

// Execute runs work and returns the process exit status.
func Execute(version string, wire work.Wiring) int {
	v := verbs{wire: wire}
	f := front{reach: v.performs(reach), enter: v.performs(enter), add: v.performs(add), remove: v.performRemove, move: v.performMove, dump: dump, edit: edit, candidates: v.listing, enterable: v.enterableListing, removable: v.removableListing, addable: v.addableListing, branches: v.branchListing}
	if err := run(command(version, naming(wire), f), os.Args[1:]); err != nil {
		if !errors.Is(err, errCancelled) {
			fmt.Fprintln(os.Stderr, "work:", err)
		}
		return 1
	}
	return 0
}

// report says the refusals a listing came back short of, nothing else carrying
// them to the reader.
func report(w io.Writer, refused []error) {
	for _, err := range refused {
		_, _ = fmt.Fprintln(w, "work:", err)
	}
}

// run puts args through the tree. The tree is never executed on the words as
// typed: [dispatch] belongs to every way in.
func run(cmd *cobra.Command, args []string) error {
	cmd.SetArgs(dispatch(cmd, args))
	return cmd.Execute()
}

// naming is the systems as the command line spells them, asked of the wiring
// without a repository: only the names and the flags are read off what comes
// back, and every system work ships is named rather than the ones one repository
// enabled.
func naming(wire work.Wiring) work.Systems {
	return wire("", "", config.Shipped())
}

// command builds the command tree, calling the verb front once the flags are
// valid and answering a tab press from its listings. The root runs nothing
// itself: [dispatch] has already sent the bare form to go.
func command(version string, sys work.Systems, f front) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "work",
		Short: "A smarter cd for git worktrees",
		Long: `work is a smarter cd for git worktrees. It knows which worktrees a repository
has open and which tickets and pull requests are waiting, and hands you the one
you pick: your shell stands in it, or a command takes the terminal.

An identifier in the first position, or none at all, is work go.`,
		Version: version,
		// A failure to enter is one line on stderr, not a wall of usage.
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// One documented door to the shell integration: work init <shell>.
	cmd.CompletionOptions.DisableDefaultCmd = true
	// Every position cobra would otherwise answer with a file listing, the
	// subcommands' arguments included, answers with nothing instead.
	cmd.CompletionOptions.SetDefaultShellCompDirective(cobra.ShellCompDirectiveNoFileComp)
	cmd.AddCommand(initCommand(), goCommand(sys, f.reach, f.candidates), switchCommand(sys, f.enter, f.enterable), addCommand(sys, f.add, f.addable), removeCommand(f.remove, f.removable), moveCommand(f.move, f.removable), listCommand(f.branches), configCommand(f.dump, f.edit))
	// Cobra adds these three as it runs, too late for [dispatch] to read them.
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()
	cmd.InitDefaultVersionFlag()

	return cmd
}

// opening builds a verb that reaches a worktree and hands it to something: the
// one identifier, the open-on flags and the one handoff. declines are the
// actions it may be told not to run.
func opening(help *cobra.Command, sys work.Systems, declines []work.Action, run func(o options, target string) (worktree.Handoff, error), list func() ([]work.Candidate, error)) *cobra.Command {
	var o options

	help.Args = cobra.MaximumNArgs(1)
	help.ValidArgsFunction = suggest(list)
	help.RunE = func(cmd *cobra.Command, args []string) error {
		h, err := run(o, arg(args, 0))
		if err != nil {
			return err
		}
		return hand(h, cmd.OutOrStdout(), cmd.ErrOrStderr())
	}
	openOn(help, &o, sys.Openers)
	decline(help, &o, declines)

	return help
}

// openOn gives a command the flags that name what a worktree opens on: one per
// action wired, under the spelling that action answers to, mutually exclusive.
func openOn(cmd *cobra.Command, o *options, openers []work.Opener) {
	exclusive := make([]string, 0, len(openers))
	for _, op := range openers {
		name, usage := spelling(op)
		exclusive = append(exclusive, name)
		// The name rather than the opener: cobra holds the callback for the life of the
		// process, and a system captured here is one nothing can let go of.
		action := op.Name()
		boolFlag(cmd, name, usage, func() { o.open = action })
		// What was renamed is the action; the refusal names the flag that reaches it today.
		for _, was := range config.Renamed(config.ActionName(op.Name())) {
			renamedFlag(cmd, string(was), name)
		}
	}
	cmd.MarkFlagsMutuallyExclusive(exclusive...)
}

// decline gives a command the flags that call off an action that would otherwise
// run. An action spelling no flag runs whenever a worktree comes into being.
func decline(cmd *cobra.Command, o *options, actions []work.Action) {
	for _, a := range actions {
		f, ok := a.(worktree.Flagged)
		if !ok {
			continue
		}
		name, usage := f.Flag()
		action := a.Name()
		boolFlag(cmd, name, usage, func() { o.skip = append(o.skip, action) })
	}
}

// spelling is the flag an opener answers to and the line --help shows for it. An
// opener that spells neither is named by the flag its own name spells.
func spelling(op work.Opener) (name, usage string) {
	if f, ok := op.(worktree.Flagged); ok {
		return f.Flag()
	}
	return op.Name(), ""
}

// boolFlag declares a flag that stands for itself and calls set where it was
// given. A value spelled out is read, so --flag=false settles nothing.
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

// renamedFlag keeps the spelling an action used to answer to, hidden and refused
// with the one it answers to now.
func renamedFlag(cmd *cobra.Command, was, now string) {
	cmd.Flags().BoolFunc(was, "", func(string) error {
		return fmt.Errorf("this flag is now --%s", now)
	})
	// The error is a flag that was not declared, and one just was.
	_ = cmd.Flags().MarkHidden(was)
}

// arg is the word a verb was given at a position, empty where it was left out
// for the picker or the prompt. Cobra has already refused one past the last.
func arg(args []string, at int) string {
	if at >= len(args) {
		return ""
	}
	return args[at]
}
