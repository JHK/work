package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/worktree"
)

// configCommand is the verb that answers for the settings. It runs nothing
// itself: what it carries is dump and edit.
func configCommand(dumping func(out io.Writer) error, editing func(out io.Writer) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Answer for the settings work reads",
		Long: `Answer for the settings work reads: your own file, over the compiled-in
defaults.`,
		Args: cobra.NoArgs,
		// Cobra answers a verb of its own with the help before it ever weighs the
		// arguments, so the sub-verb is asked for here and a typo of one is refused.
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(dumpCommand(dumping), editCommand(editing))

	return cmd
}

func dumpCommand(run func(out io.Writer) error) *cobra.Command {
	return &cobra.Command{
		Use:   "dump",
		Short: "Print the effective configuration as TOML",
		Long: `Print the configuration work resolved, as the TOML work loads back.

Templates print as they are written; rendering one needs a target.

The settings are the same wherever the shell stands, so this verb runs anywhere.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.OutOrStdout())
		},
	}
}

func (v verbs) dumping(out io.Writer) error {
	if v.refusal != nil {
		return v.refusal
	}
	_, err := io.WriteString(out, v.cfg.Dump())
	return err
}

func editCommand(run func(out io.Writer) error) *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open your settings file in $VISUAL, else $EDITOR",
		Long: `Open ~/.config/work/config.toml, the settings that follow you from repository
to repository, in $VISUAL else $EDITOR. The file and the directory it sits in
are created where neither is there yet, so an editor that creates neither still
opens.

Where neither variable names an editor, the invocation is refused before
anything is created.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.OutOrStdout())
		},
	}
}

// edit opens the settings file in an editor, $VISUAL else $EDITOR, creating the
// file and its directory. Every refusal comes first. It asks git nothing.
func edit(stdout io.Writer) error {
	editor, err := namedEditor()
	if err != nil {
		return err
	}
	path := config.UserFile()
	if path == "" {
		return errors.New("this machine names neither $XDG_CONFIG_HOME nor a home directory, so there is nowhere to keep your settings")
	}
	if err := ensureSettingsFile(path); err != nil {
		return err
	}
	return hand(worktree.Handoff{Dir: worktree.Path(filepath.Dir(path)), Run: append(editor, path)}, stdout)
}

// namedEditor is the editor the environment names, with the flags it carries.
func namedEditor() ([]string, error) {
	editor := strings.Fields(os.Getenv("VISUAL"))
	if len(editor) == 0 {
		editor = strings.Fields(os.Getenv("EDITOR"))
	}
	if len(editor) == 0 {
		return nil, errors.New("neither $VISUAL nor $EDITOR names an editor to open your settings in")
	}
	return editor, nil
}

// ensureSettingsFile brings an empty settings file into being.
func ensureSettingsFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}
