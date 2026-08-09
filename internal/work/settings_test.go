package work

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/config"
)

// Where the user's settings sit under $XDG_CONFIG_HOME, spelled out rather than
// derived, so the path the editor is opened on is the documented one.
const userRelPath = "work/config.toml"

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

// The editor is handed the settings file itself, and the file and its directory
// are brought into being for it.
func TestEditSettingsCreatesTheFile(t *testing.T) {
	repo := initRepo(t)
	home := settingsHome(t)
	t.Setenv("VISUAL", "gvim")
	path := filepath.Join(home, userRelPath)

	h, err := EditSettings(repo)
	if err != nil {
		t.Fatalf("EditSettings: %v", err)
	}
	if want := []string{"gvim", path}; !slices.Equal(h.Run, want) {
		t.Errorf("EditSettings() runs %q; want %q", h.Run, want)
	}
	if want := filepath.Dir(path); h.Dir != want {
		t.Errorf("EditSettings() changes into %q; want %q", h.Dir, want)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the settings file was not created: %v", err)
	}
}

// A file already there is the one being edited, so bringing it into being
// leaves what it holds alone.
func TestEditSettingsKeepsAFileThatIsThere(t *testing.T) {
	repo := initRepo(t)
	home := settingsHome(t)
	t.Setenv("VISUAL", "gvim")
	path := filepath.Join(home, userRelPath)
	body := "[action]\nenter = \"shell\"\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, body)

	if _, err := EditSettings(repo); err != nil {
		t.Fatalf("EditSettings: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Errorf("the settings file holds %q; want %q", got, body)
	}
}

// Nothing is created by a verb that goes on to refuse: the editor is rendered
// first, so the default with neither variable set leaves no file behind.
func TestEditSettingsWithNoEditor(t *testing.T) {
	repo := initRepo(t)
	home := settingsHome(t)
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")

	refusedEdit(t, repo, home, "open.editor")
}

// A repository is what the settings are read against, so where git names none
// the refusal is dump's, and nothing is created on the way to it.
func TestEditSettingsWithNoRepository(t *testing.T) {
	home := settingsHome(t)
	t.Setenv("VISUAL", "gvim")

	refusedEdit(t, t.TempDir(), home, "git rev-parse")
}

// The editor is named by the settings, so a configuration work will not load
// names nothing, and nothing is created.
func TestEditSettingsWithASettingItRefuses(t *testing.T) {
	repo := initRepo(t)
	home := settingsHome(t)
	t.Setenv("VISUAL", "gvim")
	if err := os.WriteFile(filepath.Join(repo, config.RepoFile), []byte("[open]\nbogus = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	refusedEdit(t, repo, home, "unknown setting")
}

// refusedEdit asserts the verb refused with nothing created, naming what the
// case says it should.
func refusedEdit(t *testing.T, dir, home, names string) {
	t.Helper()
	h, err := EditSettings(dir)
	if err == nil {
		t.Fatalf("EditSettings() = %+v; want a refusal", h)
	}
	if !strings.Contains(err.Error(), names) {
		t.Errorf("EditSettings() = %v; want it to name %q", err, names)
	}
	if _, err := os.Stat(filepath.Join(home, "work")); err == nil {
		t.Error("EditSettings() created the settings directory")
	}
}

// settingsHome puts the user's settings somewhere the test owns, whatever the
// machine running it keeps its own in.
func settingsHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	return home
}
