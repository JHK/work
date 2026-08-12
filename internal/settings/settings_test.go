package settings

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// Where the user's settings sit under $XDG_CONFIG_HOME, spelled out rather than
// derived, so the path the editor is opened on is the documented one.
const userRelPath = "work/config.toml"

// defaultDir is where worktrees go with nothing configured.
var defaultDir = config.Default().Worktree.Directory

// The dump is of the repository the directory sits in, so a linked worktree
// reads the main checkout's file rather than looking for one of its own.
func TestDumpFromALinkedWorktree(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Write(t, filepath.Join(repo, config.RepoFile), "[action]\nenter = \"claude\"\n")
	wt := filepath.Join(repo, defaultDir, "a")
	testenv.Git(t, repo, "worktree", "add", "-b", "one-a", wt)

	got, err := Dump(wt)
	if err != nil {
		t.Fatalf("Dump: %v", err)
	}
	if !strings.Contains(got, `enter = "claude"`) {
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
// are brought into being for it. $VISUAL wins over $EDITOR.
func TestEditCreatesTheFile(t *testing.T) {
	home := testenv.Home(t)
	path := filepath.Join(home, userRelPath)
	t.Setenv("VISUAL", "gvim")
	t.Setenv("EDITOR", "vi")

	h, err := Edit()
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

// $EDITOR is what an environment naming no $VISUAL is opened in.
func TestEditFallsBackToEditor(t *testing.T) {
	testenv.Home(t)
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "vi")

	h, err := Edit()
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if h.Run[0] != "vi" {
		t.Errorf("Edit() runs %q; want $EDITOR where $VISUAL names nothing", h.Run)
	}
}

// A variable is the words it holds: the first names the editor and the rest
// reach it ahead of the path. Whitespace names nothing, so a $VISUAL holding
// only that is read past as an unset one is.
func TestEditSplitsTheVariable(t *testing.T) {
	for _, tc := range []struct {
		name   string
		visual string
		editor string
		want   []string
	}{
		{"$EDITOR carrying flags", "", "  code   --wait ", []string{"code", "--wait"}},
		{"$VISUAL carrying flags", "emacsclient -nw", "vi", []string{"emacsclient", "-nw"}},
		{"$VISUAL holding whitespace", "   ", "vi", []string{"vi"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := testenv.Home(t)
			t.Setenv("VISUAL", tc.visual)
			t.Setenv("EDITOR", tc.editor)

			h, err := Edit()
			if err != nil {
				t.Fatalf("Edit: %v", err)
			}
			if want := append(tc.want, filepath.Join(home, userRelPath)); !slices.Equal(h.Run, want) {
				t.Errorf("Edit() runs %q; want %q", h.Run, want)
			}
		})
	}
}

// A file already there is the one being edited, so bringing it into being
// leaves what it holds alone.
func TestEditKeepsAFileThatIsThere(t *testing.T) {
	home := testenv.Home(t)
	path := filepath.Join(home, userRelPath)
	body := "[action]\nenter = \"shell\"\n"
	testenv.Write(t, path, body)
	t.Setenv("VISUAL", "gvim")

	if _, err := Edit(); err != nil {
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

// Nothing is created by a verb that goes on to refuse: the editor is resolved
// first, so an environment naming none leaves no file behind.
func TestEditWithNoEditor(t *testing.T) {
	for _, tc := range []struct {
		name   string
		visual string
		editor string
	}{
		{"unset", "", ""},
		{"whitespace", "   ", "\t"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := testenv.Home(t)
			t.Setenv("VISUAL", tc.visual)
			t.Setenv("EDITOR", tc.editor)

			h, err := Edit()
			if err == nil {
				t.Fatalf("Edit() = %+v; want a refusal", h)
			}
			if !strings.Contains(err.Error(), "$EDITOR") {
				t.Errorf("Edit() = %v; want it to name the variables it read", err)
			}
			if _, err := os.Stat(filepath.Join(home, "work")); err == nil {
				t.Error("Edit() created the settings directory")
			}
		})
	}
}
