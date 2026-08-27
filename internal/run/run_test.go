package run

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// What a run says goes nowhere: this package says on the process's own log, and
// a case reads what came back rather than what was said. internal/testenv, which
// puts every other package's log away, imports this one and so is out of reach.
func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.DiscardHandler))
	m.Run()
}

// A tool the machine does not have is out for the rest of the run, which is the
// run's own bookkeeping: no command shows a reader the question went unasked.
func TestAToolAlreadyMissedIsNotAskedAgain(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	Forget()
	t.Cleanup(Forget)
	_, err := Output(dir, "later", "list")
	require.Error(t, err, "a tool that is not there answered")

	// It has since arrived, so only the memo can keep it unasked.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "later"), []byte("#!/bin/sh\necho here\n"), 0o755))

	got, err := Output(dir, "later", "show")

	require.Error(t, err, "the tool answered %q; want it left unasked for the rest of the run", got)
}
