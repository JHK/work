// Package cli is work's headless front end: it turns the words typed into one
// call on [work.Env] and prints what came back.
package cli

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// errCancelled reports a choice the user declined to make. It exits non-zero
// without a message: nothing happened, and nothing went wrong.
var errCancelled = errors.New("cancelled")

// The shape each verb takes once the words are read: an opening verb hands a
// worktree over, the two that act on one hand back what they did.
type (
	opens   = func(verb, target string) (worktree.Handoff, error)
	removes = func(force bool, target string) (work.Deletion, error)
	moves   = func(target, dest string) (work.Move, error)
)

// offering is a verb and the listing it has to offer.
type offering[T any] struct {
	run  T
	list func() ([]work.Candidate, error)
}

// front is what the command tree calls once cobra has read the words: an
// offering per verb, and the verbs that offer nothing.
type front struct {
	reach    offering[opens]
	enter    offering[opens]
	add      offering[opens]
	carry    opens
	remove   offering[removes]
	move     offering[moves]
	dump     func(out io.Writer) error
	edit     func(out io.Writer) error
	branches func() ([]string, error)
}

// verbs answers the front's calls against the settings and the wiring Execute
// was handed.
type verbs struct {
	cfg  config.Config
	read error // the settings would not load, which every verb that reads them refuses with
	wire work.Wiring
}

// repository is the one the shell stands in, with its systems wired.
func (v verbs) repository() (work.Env, error) {
	if v.read != nil {
		return work.Env{}, v.read
	}
	return work.Open(".", v.cfg, v.wire)
}

// performs puts a verb over the repository the shell stands in, and puts the
// listing it was wired with up for the shell.
func performs[A, B, R any](v verbs, l listing, verb func(work.Env, listing, A, B) (R, error)) offering[func(A, B) (R, error)] {
	return offering[func(A, B) (R, error)]{
		run: func(a A, b B) (R, error) {
			env, err := v.repository()
			if err != nil {
				var none R
				return none, err
			}
			return verb(env, reporting(l), a, b)
		},
		list: v.offers(l),
	}
}

// reporting is l with every system it came back short of said once.
func reporting(l listing) listing {
	rows := l.rows
	l.rows = func(env work.Env) ([]work.Candidate, []error, error) {
		found, refused, err := rows(env)
		for _, was := range refused {
			slog.Warn(was.Error())
		}
		return found, refused, err
	}
	return l
}

// fronting wires each verb to the listing it offers, which its picker and its
// completion are both put up from.
func fronting(v verbs) front {
	return front{
		reach:    performs(v, workable, reach),
		enter:    performs(v, enterable, enter),
		add:      performs(v, addable, add),
		carry:    v.carrying(),
		remove:   performs(v, removable, remove),
		move:     performs(v, movable, move),
		dump:     dump,
		edit:     edit,
		branches: v.branchListing,
	}
}

// Execute runs work over the words it was given and returns the process exit
// status. A run answers on stdout and says on the log the caller stood the
// process on. A settings file that will not load is carried to the verbs that
// read it, not refused here.
func Execute(version string, wire work.Wiring, args []string, stdout io.Writer, logLevel *slog.LevelVar) int {
	cfg, read := config.Load()

	cmd := command(version, logLevel, fronting(verbs{cfg: cfg, read: read, wire: wire}))
	if read != nil {
		// The settings are what the reader has to hear: a file that would not load is
		// the reason a verb refuses, whatever else the words were wrong about.
		cmd.SetFlagErrorFunc(func(*cobra.Command, error) error { return read })
	}
	if err := through(cmd, args, stdout); err != nil {
		if !errors.Is(err, errCancelled) {
			slog.Error(err.Error())
		}
		return 1
	}
	return 0
}

// through puts args through the tree on the stream work answers on, cobra's own
// writes going nowhere. The tree is never executed on the words as typed:
// [dispatch] belongs to every way in.
func through(cmd *cobra.Command, args []string, stdout io.Writer) error {
	cmd.SetOut(stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(dispatch(cmd, args))
	return cmd.Execute()
}

// command builds the command tree, calling the verb front once the flags are
// valid and answering a tab press from its listings. The root runs nothing
// itself: [dispatch] has already sent the bare form to go.
func command(version string, logLevel *slog.LevelVar, f front) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "work",
		Short: "A smarter cd for git worktrees",
		Long: `work is a smarter cd for git worktrees. It knows which worktrees a repository
has open and which tickets and pull requests are waiting, and hands you the one
you pick: your shell stands in it, or a command takes the terminal.`,
		Version: fmt.Sprintf("%s (%s)", version, runtime.Version()),
		// A failure to enter is one line on stderr, not a wall of usage.
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// One documented door to the shell integration: work init <shell>.
	cmd.CompletionOptions.DisableDefaultCmd = true
	// Every position cobra would otherwise answer with a file listing, the
	// subcommands' arguments included, answers with nothing instead.
	cmd.CompletionOptions.SetDefaultShellCompDirective(cobra.ShellCompDirectiveNoFileComp)
	cmd.AddCommand(initCommand(), goCommand(f.reach), switchCommand(f.enter), addCommand(f.add), carryCommand(f.carry), removeCommand(f.remove), moveCommand(f.move), listCommand(f.branches), configCommand(f.dump, f.edit))
	logging(cmd, logLevel)
	// Declared here so that cobra does not give it the -v below leaves free.
	cmd.Flags().Bool("version", false, "print the version and the Go toolchain")
	// Cobra adds what it is still missing as it runs, too late for [dispatch] to
	// read them.
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()

	return cmd
}

// opening builds a verb that reaches a worktree and hands it to something, with
// the listing it offers behind both its picker and its completion.
func opening(help *cobra.Command, verb offering[opens]) *cobra.Command {
	help.Args = cobra.MaximumNArgs(1)
	help.ValidArgsFunction = suggest(verb.list)
	return handing(help, verb.run)
}

// handing wires the one handoff a verb that opens something ends on. The verb is
// read off the command that ran, which is what a creation's opener follows from.
func handing(help *cobra.Command, run opens) *cobra.Command {
	help.RunE = func(cmd *cobra.Command, args []string) error {
		h, err := run(cmd.Name(), arg(args, 0))
		if err != nil {
			return err
		}
		return hand(h, cmd.OutOrStdout())
	}
	return help
}

// arg is the word a verb was given at a position, empty where it was left out
// for the picker or the prompt. Cobra has already refused one past the last.
func arg(args []string, at int) string {
	if at >= len(args) {
		return ""
	}
	return args[at]
}
