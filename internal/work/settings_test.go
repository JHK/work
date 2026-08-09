package work

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/config"
)

// The dump is of the repository the directory sits in, so a linked worktree
// reads the main checkout's file rather than looking for one of its own.
func TestSettingsFromALinkedWorktree(t *testing.T) {
	repo := initRepo(t)
	if err := os.WriteFile(filepath.Join(repo, config.RepoFile), []byte("[action]\nenter = \"shell\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wt := filepath.Join(repo, defaultDir, "a")
	gitCmd(t, repo, "worktree", "add", "-b", "one-a", wt)

	got, err := Settings(wt)
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if !strings.Contains(got, `enter = "shell"`) {
		t.Errorf("Settings() = %q; want the main checkout's action.enter", got)
	}
}

// A repository is what the layers are read against, so where git names none
// there is nothing to dump and the refusal is git's own.
func TestSettingsWithNoRepository(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	got, err := Settings(t.TempDir())
	if err == nil {
		t.Fatalf("Settings() = %q; want a refusal", got)
	}
	if got != "" {
		t.Errorf("Settings() printed %q; want nothing", got)
	}
}
