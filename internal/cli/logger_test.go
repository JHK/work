package cli

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// These are the only cases that read a rendered line. Everywhere else a case
// reads what work said, so a spelling that changes moves here and nowhere else.
// What tint does with a record is tint's to test; these hold the options work
// hands it.

// A line is the level at a glance and the sentence work said, with the time
// dropped and nothing wrapped around either.
func TestALineIsTheLevelTagAndTheMessage(t *testing.T) {
	var said strings.Builder

	log := Logger(&said, slog.LevelDebug)
	log.Error("scratch carries changes")
	log.Warn("gh pr list: exit status 1")
	log.Info("git worktree list --porcelain")
	log.Debug("the settings in force")

	want := "ERR scratch carries changes\n" +
		"WRN gh pr list: exit status 1\n" +
		"INF git worktree list --porcelain\n" +
		"DBG the settings in force\n"
	require.Equal(t, want, said.String(), "the lines a reader is given")
}

// Colour reaches a terminal only, so what a reader redirected carries the
// sentence and no escape sequence with it.
func TestColourReachesATerminalOnly(t *testing.T) {
	to, read := redirected(t)

	Logger(to, slog.LevelError).Error("scratch carries changes")

	said := read()
	require.NotEmpty(t, said, "work said nothing to the stream")
	require.NotContains(t, said, "\x1b", "colour reached what is not a terminal")
}

// redirected is a file a reader sent work's diagnostics to, and what that file
// was left holding.
func redirected(t *testing.T) (*os.File, func() string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "said")
	to, err := os.Create(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = to.Close() })
	return to, func() string {
		said, err := os.ReadFile(path)
		require.NoError(t, err)
		return string(said)
	}
}
