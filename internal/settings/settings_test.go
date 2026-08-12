package settings

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/action/open"
	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// Where the user's settings sit under $XDG_CONFIG_HOME, spelled out rather than
// derived, so the path the editor is opened on is the documented one.
const userRelPath = "work/config.toml"

// defaultDir is where worktrees go with nothing configured.
var defaultDir = config.Default().Worktree.Directory

// editing wires the editor action over a source naming the editor, which is
// where the name comes from at this verb as at every other: what open.editor
// renders with is supplied rather than read here.
func editing(editor string) work.Wiring {
	return func(_, _ string, cfg config.Config) work.Systems {
		return work.Systems{
			Openers: []work.Opener{open.Editor(cfg.Open)},
			Sources: []worktree.Source{source{worktree.Values{"Editor": editor}}},
		}
	}
}

// source supplies what it was made with, whatever it is asked about.
type source struct{ vals worktree.Values }

func (s source) Supply(worktree.Tree) (worktree.Values, error) { return s.vals, nil }

// The dump is of the repository the directory sits in, so a linked worktree
// reads the main checkout's file rather than looking for one of its own.
func TestDumpFromALinkedWorktree(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Write(t, filepath.Join(repo, config.RepoFile), "[action]\nenter = \"diff\"\n")
	wt := filepath.Join(repo, defaultDir, "a")
	testenv.Git(t, repo, "worktree", "add", "-b", "one-a", wt)

	got, err := Dump(wt)
	if err != nil {
		t.Fatalf("Dump: %v", err)
	}
	if !strings.Contains(got, `enter = "diff"`) {
		t.Errorf("Dump() = %q; want the main checkout's action.enter", got)
	}
}

// A repository is what the layers are read against, so where git names none
// there is nothing to dump and the refusal is git's own.
func TestDumpWithNoRepository(t *testing.T) {
	got, err := Dump(t.TempDir())
	if err == nil {
		t.Fatalf("Dump() = %q; want a refusal", got)
	}
	if got != "" {
		t.Errorf("Dump() printed %q; want nothing", got)
	}
}

// The editor is handed the settings file itself, and the file and its directory
// are brought into being for it.
func TestEditCreatesTheFile(t *testing.T) {
	repo, home := testenv.InitRepo(t), testenv.Home(t)
	path := filepath.Join(home, userRelPath)

	h, err := Edit(repo, editing("gvim"))
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if want := []string{"gvim", path}; !slices.Equal(h.Run, want) {
		t.Errorf("Edit() runs %q; want %q", h.Run, want)
	}
	if want := filepath.Dir(path); h.Dir != want {
		t.Errorf("Edit() changes into %q; want %q", h.Dir, want)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the settings file was not created: %v", err)
	}
}

// A file already there is the one being edited, so bringing it into being
// leaves what it holds alone.
func TestEditKeepsAFileThatIsThere(t *testing.T) {
	repo, home := testenv.InitRepo(t), testenv.Home(t)
	path := filepath.Join(home, userRelPath)
	body := "[action]\nenter = \"shell\"\n"
	testenv.Write(t, path, body)

	if _, err := Edit(repo, editing("gvim")); err != nil {
		t.Fatalf("Edit: %v", err)
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
// first, so the default with nothing naming an editor leaves no file behind.
func TestEditWithNoEditor(t *testing.T) {
	refusedEdit(t, testenv.InitRepo(t), editing(""), "open.editor")
}

// A repository is what the settings are read against, so where git names none
// the refusal is dump's, and nothing is created on the way to it.
func TestEditWithNoRepository(t *testing.T) {
	refusedEdit(t, t.TempDir(), editing("gvim"), "git rev-parse")
}

// The editor is named by the settings, so a configuration work will not load
// names nothing, and nothing is created.
func TestEditWithASettingItRefuses(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Write(t, filepath.Join(repo, config.RepoFile), "[open]\nbogus = 1\n")

	refusedEdit(t, repo, editing("gvim"), "unknown setting")
}

// The action is reached through the seam like any other, so a wiring with no
// editor in it is refused as any unknown action is.
func TestEditWithNoEditorAction(t *testing.T) {
	bare := func(string, string, config.Config) work.Systems { return work.Systems{} }

	refusedEdit(t, testenv.InitRepo(t), bare, "editor")
}

// refusedEdit asserts the verb refused with nothing created, naming what the
// case says it should. The home it asserts about is its own.
func refusedEdit(t *testing.T, dir string, wire work.Wiring, names string) {
	t.Helper()
	home := testenv.Home(t)
	h, err := Edit(dir, wire)
	if err == nil {
		t.Fatalf("Edit() = %+v; want a refusal", h)
	}
	if !strings.Contains(err.Error(), names) {
		t.Errorf("Edit() = %v; want it to name %q", err, names)
	}
	if _, err := os.Stat(filepath.Join(home, "work")); err == nil {
		t.Error("Edit() created the settings directory")
	}
}
