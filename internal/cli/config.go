package cli

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
)

// configCommand is the verb that answers for the settings. It runs nothing
// itself: what it carries is dump, and a second sub-verb would be added here.
func configCommand(run func(out io.Writer) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Answer for the settings work reads",
		Long: `Answer for the settings work reads: the repository's .work.toml and your own
file, merged over the compiled-in defaults.

Nothing here writes a settings file.`,
		Args: cobra.NoArgs,
		// Cobra answers a verb of its own with the help before it ever weighs the
		// arguments, so the sub-verb is asked for here and a typo of one is refused.
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(dumpCommand(run))

	return cmd
}

// dumpCommand prints the configuration work resolved. It takes no argument,
// there being one configuration to print.
func dumpCommand(run func(out io.Writer) error) *cobra.Command {
	return &cobra.Command{
		Use:   "dump",
		Short: "Print the effective configuration as TOML, each key under its source",
		Long: `Print the configuration work resolved here as TOML, every key under a comment
naming the layer that set it: the compiled-in default, your file, or the
repository's .work.toml.

Templates print as they are written; rendering one needs a target.

Outside a repository the printed configuration is your file over the defaults,
and it says .work.toml went unread.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.OutOrStdout())
		},
	}
}

// dump prints the settings of the repository the shell stands in, or the user's
// alone where it stands outside one. Nothing is printed of a configuration work
// would refuse to load.
func dump(out io.Writer) error {
	text, err := work.Settings(".")
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, text)
	return err
}
