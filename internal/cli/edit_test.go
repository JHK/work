package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// userRelPath is where the settings file sits under a settings home, as a user
// spells it out and no case derives it from the code.
const userRelPath = "work/config.toml"

// editing stands a shell in a repository whose editors all stand in, in a
// settings home of the case's own, and hands back where the file belongs.
func editing(t *testing.T, editors ...string) (*session, string) {
	t.Helper()
	stubs := make([]testenv.Stub, len(editors))
	for i, name := range editors {
		stubs[i] = testenv.Stub{Name: name}
	}
	s := repository(t, stubs...)
	return s, filepath.Join(testenv.Home(t), userRelPath)
}

// The editor is handed the settings file itself, and the file and the directory
// it sits in are brought into being for it.
func TestConfigEditOpensTheSettingsFile(t *testing.T) {
	s, path := editing(t, "gvim")
	t.Setenv("VISUAL", "gvim")

	r := s.hands("config", "edit")

	r.came(t, result{Asked: []string{"gvim " + path}})
	require.FileExists(t, path, "the settings file was not created")
}

// A variable is the words it holds: the first names the editor and the rest
// reach it ahead of the path. Whitespace names nothing, so a $VISUAL holding
// only that is read past as an unset one is.
func TestConfigEditReadsTheEditorOutOfTheEnvironment(t *testing.T) {
	tests := []struct {
		name, visual, editor, asked string
	}{
		{"$VISUAL over $EDITOR, and its flags", "emacsclient -nw", "vi", "emacsclient -nw"},
		{"$VISUAL unset, so $EDITOR and its flags", "", "  code   --wait ", "code --wait"},
		{"$VISUAL naming nothing, so $EDITOR", "   ", "vi", "vi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, path := editing(t, "emacsclient", "code", "vi")
			t.Setenv("VISUAL", tt.visual)
			t.Setenv("EDITOR", tt.editor)

			r := s.hands("config", "edit")

			r.came(t, result{Asked: []string{tt.asked + " " + path}})
		})
	}
}

// Bringing the file into being leaves what it already holds alone.
func TestConfigEditKeepsAFileThatIsThere(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "gvim"})
	body := "[action]\nenter = \"shell\"\n"
	path := s.settings(body)
	t.Setenv("VISUAL", "gvim")

	r := s.hands("config", "edit")

	r.came(t, result{Asked: []string{"gvim " + path}})
	held, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, body, string(held), "the settings file was written over")
}

// Nothing is created by a verb that goes on to refuse: the editor is resolved
// first, so an environment naming none leaves no file behind.
func TestConfigEditWithNoEditor(t *testing.T) {
	for _, tt := range []struct{ name, visual, editor string }{
		{"unset", "", ""},
		{"whitespace", "   ", "\t"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, path := editing(t)
			t.Setenv("VISUAL", tt.visual)
			t.Setenv("EDITOR", tt.editor)

			r := s.run("config", "edit")

			r.refused(t, "$EDITOR")
			require.NoDirExists(t, filepath.Dir(path), "the settings directory was created for a verb that refused")
		})
	}
}

// A file work will not read is the one reason to reach for the editor, so this
// is the one verb such a file does not stop: docs/references/cli.md#config.
func TestConfigEditOpensAFileWorkWillNotRead(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "gvim"})
	path := s.settings("[worktree]\ndirectry = \"oops\"\n")
	t.Setenv("VISUAL", "gvim")

	r := s.hands("config", "edit")

	r.came(t, result{Asked: []string{"gvim " + path}})
}

// The editor is left standing in the directory the settings file sits in:
// docs/references/cli.md#handoff.
func TestConfigEditHandsTheTerminalToTheEditorInTheSettingsDirectory(t *testing.T) {
	// pwd -P rather than pwd, which the stand-in's shell answers out of a $PWD
	// work's own chdir left stale.
	s := repository(t, testenv.Stub{Name: "gvim", Shell: "pwd -P"})
	path := filepath.Join(testenv.Home(t), userRelPath)
	t.Setenv("VISUAL", "gvim")

	r := s.hands("config", "edit")

	r.came(t, result{Out: resolved(t, filepath.Dir(path)) + "\n", Asked: []string{"gvim " + path}})
}
