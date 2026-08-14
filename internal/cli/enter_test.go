package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/shim"
	"github.com/JHK/work-cli/internal/worktree"
)

// A worktree that opens on no command is handed back rather than run: the shim
// changes into it, and a shell without one reads it off stdout.
func TestHandAnswersWithTheWorktree(t *testing.T) {
	var out strings.Builder
	if err := hand(worktree.Handoff{Dir: "/repo/.worktrees/bd-1"}, &out); err != nil {
		t.Fatalf("hand: %v", err)
	}
	if !strings.Contains(out.String(), "/repo/.worktrees/bd-1") {
		t.Errorf("the worktree came back as %q; want the path", out.String())
	}
}

// The file names one invocation: a command the terminal goes to must not inherit
// it, or a work run inside it answers into the shell still waiting on that one.
func TestHandForgetsTheFileBeforeTheCommandRuns(t *testing.T) {
	t.Setenv(shim.CDFile, filepath.Join(t.TempDir(), "answer"))

	// A command nothing can find refuses before the exec, leaving the test its own
	// process to look at.
	h := worktree.Handoff{Dir: t.TempDir(), Run: []string{"work-cli-nothing-goes-by-this"}}
	if err := hand(h, io.Discard); err == nil {
		t.Fatal("hand ran a command that is not there; want the refusal")
	}
	if file := os.Getenv(shim.CDFile); file != "" {
		t.Errorf("the command was handed the terminal with %s=%q; want it dropped", shim.CDFile, file)
	}
}
