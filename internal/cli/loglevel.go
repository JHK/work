package cli

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

// LogLevel is the level work says from before --log-level raises it.
func LogLevel() *slog.LevelVar {
	level := new(slog.LevelVar)
	level.Set(slog.LevelWarn)
	return level
}

// logging declares the flag that sets logLevel, on the root and so on every verb
// below it, and answers a tab press after it with the levels.
func logging(cmd *cobra.Command, logLevel *slog.LevelVar) {
	cmd.PersistentFlags().Var(level{of: logLevel}, "log-level", "say what work reached for")
	_ = cmd.RegisterFlagCompletionFunc("log-level", cobra.FixedCompletions(levels, cobra.ShellCompDirectiveNoFileComp))
}

// levels are the words --log-level takes, in the order --help spells them.
var levels = []string{"warn", "info", "debug"}

// level is --log-level's value: a word becomes a level here and nowhere else.
type level struct{ of *slog.LevelVar }

func (l level) Set(word string) error {
	var at slog.Level
	// slog reads words work does not take, error and warn+2 among them.
	if at.UnmarshalText([]byte(word)) != nil || !slices.Contains(levels, word) {
		return fmt.Errorf("work says at %s", strings.Join(levels, ", "))
	}
	l.of.Set(at)
	return nil
}

// String is the level the log stands at, which pflag prints as the default.
func (l level) String() string { return strings.ToLower(l.of.Level().String()) }

// Type is the set the flag takes, which pflag prints where a placeholder goes.
func (l level) Type() string { return strings.Join(levels, "|") }
