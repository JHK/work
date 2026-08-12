package open

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// The environment is where the two keys that name it get their value: it
// reaches open.shell and open.editor as a value, not as a key it overrides.
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

// $VISUAL wins over $EDITOR, and an editor nothing names is supplied as nothing
// rather than left out, which is what tells it from a value still coming.
func TestSupplyReadsTheEditor(t *testing.T) {
	for _, tt := range []struct{ visual, editor, want string }{
		{"gvim", "vi", "gvim"},
		{"", "vi", "vi"},
		{"", "", ""},
	} {
		t.Setenv("VISUAL", tt.visual)
		t.Setenv("EDITOR", tt.editor)
		vals, err := Values{}.Supply(worktree.Tree{})
		if err != nil {
			t.Fatalf("Supply: %v", err)
		}
		if got, named := vals["Editor"]; !named || got != tt.want {
			t.Errorf("$VISUAL=%q $EDITOR=%q gave editor %q, named %v; want %q supplied", tt.visual, tt.editor, got, named, tt.want)
		}
	}
}

// The base is a revision handed over for git to resolve rather than one read
// here, so it is the same whatever the worktree and a worktree that does not
// exist yet has one too.
func TestSupplyIsTheSameRevisionWhateverTheWorktree(t *testing.T) {
	for _, tree := range []worktree.Tree{
		{},
		{Place: worktree.Place{Name: "one"}, Path: "/wt"},
	} {
		vals, err := Values{}.Supply(tree)
		if err != nil {
			t.Fatalf("Supply: %v", err)
		}
		if vals["Base"] != base {
			t.Errorf("Supply(%+v) gave base %q; want the revision %q", tree, vals["Base"], base)
		}
	}
}

// What git makes of the default command is the merge-base against the working
// tree: run from the worktree, the diff covers the worktree's own work,
// uncommitted work included, and what has landed on the main checkout since is on
// the far side of it. git is what enforces that rather than work, so the command
// the settings name is the one run here.
func TestTheBaseDiffsTheWorktreesOwnWork(t *testing.T) {
	repo := testenv.InitRepo(t)
	wt := filepath.Join(repo, "trees", "one")
	testenv.Git(t, repo, "worktree", "add", "-b", "one", wt)
	testenv.Write(t, filepath.Join(wt, "own"), "the worktree's own work")
	testenv.Git(t, wt, "add", ".")
	testenv.Git(t, wt, "commit", "-m", "the worktree's own work")
	// Left uncommitted, which a diff between two commits would leave out.
	testenv.Write(t, filepath.Join(wt, "still-going"), "not committed yet")
	testenv.Git(t, wt, "add", ".")
	testenv.Write(t, filepath.Join(repo, "later"), "the main checkout has moved on")
	testenv.Git(t, repo, "add", ".")
	testenv.Git(t, repo, "commit", "-m", "the main checkout has moved on")

	vals, err := Values{}.Supply(worktree.Tree{Path: wt})
	if err != nil {
		t.Fatalf("Supply: %v", err)
	}
	// What the core supplies of every worktree, whichever resolver answered for it.
	vals["Name"], vals["Dir"] = "one", wt

	argv, err := config.Default().Open.Diff().Render(vals)
	if err != nil || argv[0] != "git" {
		t.Fatalf("the default diff renders %q, %v; want a git command", argv, err)
	}

	files := testenv.Git(t, wt, append(argv[1:], "--name-only")...)
	if want := "own\nstill-going"; files != want {
		t.Errorf("the diff covers %q; want %q", files, want)
	}
}

// The values in hand are what decide whether an action is one to offer: an
// editor the environment never named leaves no command to run, while a shell
// nothing names has a fallback and the diff's base is a constant.
func TestAppliesTellsAMissingToolFromAValueStillComing(t *testing.T) {
	cfg := config.Default().Open
	t.Setenv("SHELL", "")
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	vals, err := Values{}.Supply(worktree.Tree{})
	if err != nil {
		t.Fatalf("Supply: %v", err)
	}

	// The shell has a fallback every machine has, and the diff's base is a constant
	// revision rather than something the environment has to name.
	if err := Shell(cfg).Applies(vals); err != nil {
		t.Errorf("the shell = %v; want it on the offer", err)
	}
	if err := Diff(cfg).Applies(vals); err != nil {
		t.Errorf("the diff = %v; want it on the offer", err)
	}
	if err := Editor(cfg).Applies(vals); !errors.Is(err, worktree.ErrAbsent) {
		t.Errorf("the editor = %v; want it left off the offer", err)
	}

	// And an editor the environment does name is on the offer with the rest.
	t.Setenv("VISUAL", "gvim")
	if vals, err = (Values{}).Supply(worktree.Tree{}); err != nil {
		t.Fatalf("Supply: %v", err)
	}
	if err := Editor(cfg).Applies(vals); err != nil {
		t.Errorf("the editor = %v; want it on the offer once the environment names one", err)
	}
}

// A refusal says which key would have to name the tool, so the reason is
// actionable rather than a bare "no editor".
func TestARefusalNamesTheKeyBehindIt(t *testing.T) {
	err := refusedEditor(t)
	if !strings.Contains(err.Error(), editorKey) {
		t.Errorf("the editor = %v; want it to name %s", err, editorKey)
	}
}

// The sentinel is what the core reads with errors.Is; what a person reads is the
// reason, so the reason is what the refusal leads with.
func TestARefusalLeadsWithItsReason(t *testing.T) {
	err := refusedEditor(t)
	if !strings.HasPrefix(err.Error(), editorKey) {
		t.Errorf("the editor = %v; want the reason to lead rather than the sentinel", err)
	}
	// Still the bit the core reads, whatever the prose says.
	if !errors.Is(err, worktree.ErrAbsent) {
		t.Errorf("the editor = %v; want it left off the offer", err)
	}
}

// refusedEditor is the refusal an environment naming no editor produces.
func refusedEditor(t *testing.T) error {
	t.Helper()
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	vals, err := Values{}.Supply(worktree.Tree{})
	if err != nil {
		t.Fatalf("Supply: %v", err)
	}
	err = Editor(config.Default().Open).Applies(vals)
	if err == nil {
		t.Fatal("the editor applies with nothing naming one; want it refused")
	}
	return err
}

// editorKey is what the refusal above has to name, spelled as the settings spell
// it.
const editorKey = "open.editor"

// Each of the three is one command the [open] table names, under the name the
// flag reaches it by and the key spells it as.
func TestEachActionGoesByItsKeysName(t *testing.T) {
	cfg := config.Default().Open
	tests := []struct {
		action Action
		want   config.ActionName
	}{
		{Shell(cfg), config.ActionShell},
		{Editor(cfg), config.ActionEditor},
		{Diff(cfg), config.ActionDiff},
	}
	for _, tt := range tests {
		if tt.action.Name() != string(tt.want) {
			t.Errorf("the action goes by %q; want %q", tt.action.Name(), tt.want)
		}
	}
}

// The configured command is what the worktree is handed to, and the environment
// is only the value its template places.
func TestOpenRendersTheConfiguredCommand(t *testing.T) {
	repo := t.TempDir()
	body := "[open]\nshell = [\"{{.Shell}}\", \"--login\", \"{{.Name}}\"]\neditor = [\"{{.Editor}}\", \"--wait\", \"{{.Dir}}\"]\n"
	testenv.Write(t, filepath.Join(repo, config.RepoFile), body)
	cfg, err := config.Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	t.Setenv("SHELL", "/usr/bin/fish")
	t.Setenv("VISUAL", "gvim")

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
	got, err = Editor(cfg.Open).Open(tree, vals)
	if want := []string{"gvim", "--wait", "/wt"}; err != nil || !slices.Equal(got.Run, want) {
		t.Errorf("the editor runs %q, %v; want %q", got.Run, err, want)
	}
	if got.Dir != "/wt" {
		t.Errorf("the editor hands over in %q; want the worktree", got.Dir)
	}
}

// A command that will not render is reported rather than handed over half made.
func TestOpenRefusesWhatItCannotRender(t *testing.T) {
	tree := worktree.Tree{Place: worktree.Place{Name: "one"}, Path: "/wt"}
	// The editor's own value is missing entirely, which is a value nothing supplied
	// rather than one supplied empty.
	vals := worktree.Values{"Name": "one", "Dir": "/wt"}

	if _, err := Editor(config.Default().Open).Open(tree, vals); err == nil {
		t.Error("Open with no editor supplied: want the command refused")
	}
}
