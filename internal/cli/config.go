package cli

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/settings"
)

// configCommand is the verb that answers for the settings. It runs nothing
// itself: what it carries is dump and edit.
func configCommand(run, open func(out io.Writer) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Answer for the settings work reads",
		Long: `Answer for the settings work reads: the repository's .work.toml and your own
file, merged over the compiled-in defaults.

dump prints what they resolved to here; edit opens your own file.`,
		Args: cobra.NoArgs,
		// Cobra answers a verb of its own with the help before it ever weighs the
		// arguments, so the sub-verb is asked for here and a typo of one is refused.
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(dumpCommand(run), editCommand(open))

	return cmd
}

// dumpCommand prints the configuration work resolved.
func dumpCommand(run func(out io.Writer) error) *cobra.Command {
	return &cobra.Command{
		Use:   "dump",
		Short: "Print the effective configuration as TOML, each key under its source",
		Long: `Print the configuration work resolved here as TOML, every key under a comment
naming the layer that set it: the compiled-in default, your file, or the
repository's .work.toml.

Templates print as they are written; rendering one needs a target.

A repository is what the layers are read against, so outside one there is nothing
to print and the refusal is git's.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.OutOrStdout())
		},
	}
}

// dump prints the settings of the repository the shell stands in. Nothing is
// printed where git names no repository, or of a configuration work would refuse
// to load.
func dump(out io.Writer) error {
	text, err := settings.Dump(".")
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, text)
	return err
}

// editCommand opens the user's settings file. The repository's own is already
// where the shell stands.
func editCommand(run func(out io.Writer) error) *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open your own settings file in $VISUAL, else $EDITOR",
		Long: `Open ~/.config/work/config.toml, the settings that follow you from repository
to repository, in $VISUAL else $EDITOR. The file and the directory it sits in
are created where neither is there yet, so an editor that creates neither still
opens. The repository's own .work.toml is not this file.

Nothing is created where neither variable names an editor.

Nothing runs after: the terminal is the editor's from here.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.OutOrStdout())
		},
	}
}

// edit opens the user's settings file and hands the terminal over to the editor
// the environment names. It asks git nothing.
func edit(out io.Writer) error {
	h, err := settings.Edit()
	if err != nil {
		return err
	}
	// The editor always takes the terminal, so nothing is ever handed back to advise on.
	return hand(h, out, io.Discard)
}
