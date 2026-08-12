package open

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// The environment is where the key that names it gets its value: it reaches
// open.shell as a value, not as a key it overrides.
func TestSupplyReadsTheShell(t *testing.T) {
	for _, tt := range []struct{ shell, want string }{
		{"/usr/bin/fish", "/usr/bin/fish"},
		{"", "/bin/sh"}, // a fallback every machine has
	} {
		t.Setenv("SHELL", tt.shell)
		vals, err := Values{}.Supply(worktree.Tree{})
		if err != nil {
			t.Fatalf("Supply: %v", err)
		}
		if vals["Shell"] != tt.want {
			t.Errorf("$SHELL=%q gave shell %q; want %q", tt.shell, vals["Shell"], tt.want)
		}
	}
}

// The action goes by the name the flag reaches it by and the key spells it as.
func TestTheActionGoesByItsKeysName(t *testing.T) {
	if got := Shell(config.Default().Open).Name(); got != string(config.ActionShell) {
		t.Errorf("the action goes by %q; want %q", got, config.ActionShell)
	}
}

// The configured command is what the worktree is handed to, and the environment
// is only the value its template places.
func TestOpenRendersTheConfiguredCommand(t *testing.T) {
	repo := t.TempDir()
	testenv.Write(t, filepath.Join(repo, config.RepoFile), "[open]\nshell = [\"{{.Shell}}\", \"--login\", \"{{.Name}}\"]\n")
	cfg, err := config.Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	t.Setenv("SHELL", "/usr/bin/fish")

	tree := worktree.Tree{Place: worktree.Place{Name: "bd-42"}, Path: "/wt"}
	vals := worktree.Values{"Name": "bd-42", "Dir": "/wt"}
	supplied, err := Values{}.Supply(tree)
	if err != nil {
		t.Fatalf("Supply: %v", err)
	}
	vals.Merge(supplied)

	got, err := Shell(cfg.Open).Open(tree, vals)
	if want := []string{"/usr/bin/fish", "--login", "bd-42"}; err != nil || !slices.Equal(got.Run, want) {
		t.Errorf("the shell runs %q, %v; want %q", got.Run, err, want)
	}
	if got.Dir != "/wt" {
		t.Errorf("the shell hands over in %q; want the worktree", got.Dir)
	}
}

// A command that will not render is reported rather than handed over half made.
func TestOpenRefusesWhatItCannotRender(t *testing.T) {
	tree := worktree.Tree{Place: worktree.Place{Name: "one"}, Path: "/wt"}
	// The shell's own value is missing entirely, which is a value nothing supplied
	// rather than one supplied empty.
	vals := worktree.Values{"Name": "one", "Dir": "/wt"}

	if _, err := Shell(config.Default().Open).Open(tree, vals); err == nil {
		t.Error("Open with no shell supplied: want the command refused")
	}
}
