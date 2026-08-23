package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
)

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
	body := "[action]\nenter = \"shell\"\n"
	path := testenv.Settings(t, body)
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
