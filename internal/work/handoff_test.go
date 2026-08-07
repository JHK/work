package work

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/beads"
	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/sessions"
)

// execHandoff is the environment variable the re-exec helper below keys on.
// Exec replaces the process, so the only way to observe it is from a child.
const execHandoff = "WORK_TEST_HANDOFF_DIR"

func TestMain(m *testing.M) {
	if dir := os.Getenv(execHandoff); dir != "" {
		if err := (Handoff{Dir: dir, Run: []string{"pwd"}}).Exec(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	os.Exit(m.Run())
}

func TestHandoffExec(t *testing.T) {
	dir := t.TempDir()

	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), execHandoff+"="+dir)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("re-exec: %v", err)
	}
	// pwd resolves symlinks the temp dir may carry; compare what the shell sees.
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != want {
		t.Errorf("ran in %q, want %q", strings.TrimSpace(string(out)), want)
	}
}

func TestHandoffExecRejects(t *testing.T) {
	if err := (Handoff{Dir: "/tmp"}).Exec(); err == nil {
		t.Error("Exec with no command: want an error")
	}
	if err := (Handoff{Dir: "/no/such/dir", Run: []string{"pwd"}}).Exec(); err == nil {
		t.Error("Exec into a missing directory: want an error")
	}
	if err := (Handoff{Dir: "/tmp", Run: []string{"no-such-binary-xyz"}}).Exec(); err == nil {
		t.Error("Exec of a missing binary: want an error")
	}
}

// The environment is where the default comes from: it reaches open.shell as a
// value, not as a key it overrides.
func TestShell(t *testing.T) {
	var e Env
	s := State{Path: "/w"}

	t.Setenv("SHELL", "/usr/bin/fish")
	if got, err := e.Shell(s, Options{}); err != nil || !slices.Equal(got, []string{"/usr/bin/fish"}) {
		t.Errorf("Shell() = %q, %v; want the login shell", got, err)
	}

	t.Setenv("SHELL", "")
	if got, err := e.Shell(s, Options{}); err != nil || !slices.Equal(got, []string{"/bin/sh"}) {
		t.Errorf("Shell() = %q, %v; want the fallback", got, err)
	}
}

func TestEditor(t *testing.T) {
	var e Env
	s := State{Path: "/w"}

	t.Setenv("EDITOR", "vi")
	t.Setenv("VISUAL", "gvim")
	got, err := e.Editor(s, Options{})
	if err != nil || !slices.Equal(got, []string{"gvim", "/w"}) {
		t.Errorf("Editor() = %q, %v; want $VISUAL on the worktree", got, err)
	}

	t.Setenv("VISUAL", "")
	got, err = e.Editor(s, Options{})
	if err != nil || !slices.Equal(got, []string{"vi", "/w"}) {
		t.Errorf("Editor() = %q, %v; want $EDITOR on the worktree", got, err)
	}

	// The default command is left with nothing to run, and the refusal names the
	// key that would have to say otherwise.
	t.Setenv("EDITOR", "")
	if _, err := e.Editor(s, Options{}); err == nil {
		t.Error("Editor() with neither set: want an error")
	} else if !strings.Contains(err.Error(), "open.editor") {
		t.Errorf("Editor() = %v; want it to name open.editor", err)
	}
}

// A configured command is what the worktree is handed to, and the environment
// is only the value its template places.
func TestOpenCommandsComeFromTheConfig(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	body := "[open]\nshell = [\"{{.Shell}}\", \"--login\", \"{{.Name}}\"]\neditor = [\"{{.Editor}}\", \"--wait\", \"{{.Dir}}\"]\n"
	if err := os.WriteFile(filepath.Join(repo, config.RepoFile), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", config.RepoFile, err)
	}
	cfg, err := config.Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	e := Env{Config: cfg}
	s := State{Target: Target{Name: "bd-42"}, Path: "/w"}
	t.Setenv("SHELL", "/usr/bin/fish")
	t.Setenv("VISUAL", "gvim")

	got, err := e.Shell(s, Options{})
	if want := []string{"/usr/bin/fish", "--login", "bd-42"}; err != nil || !slices.Equal(got, want) {
		t.Errorf("Shell() = %q, %v; want %q", got, err, want)
	}
	got, err = e.Editor(s, Options{})
	if want := []string{"gvim", "--wait", "/w"}; err != nil || !slices.Equal(got, want) {
		t.Errorf("Editor() = %q, %v; want %q", got, err, want)
	}
}

// The compiled-in defaults are what work launched before any of this was a
// setting, less the review prompt a pull request used to get.
func TestLaunch(t *testing.T) {
	var e Env
	bead, _ := e.Resolve("bd-1")
	pr, _ := e.Resolve("7")
	tests := []struct {
		name  string
		state State
		opts  Options
		want  []string
	}{
		{
			"a ticket opens on the skill that works it",
			State{Target: bead, Bead: beads.Bead{Title: "a title"}},
			Options{},
			[]string{"claude", "--permission-mode", "auto", "--name=bd-1: a title", "/start bd-1"},
		},
		{
			"a pull request opens on a bare named session",
			State{Target: pr},
			Options{},
			[]string{"claude", "--name=PR #7"},
		},
		{
			"model and effort are placed where the template names them",
			State{Target: pr},
			Options{Model: "opus", Effort: "high"},
			[]string{"claude", "--name=PR #7", "--model=opus", "--effort=high"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.Launch(tt.state, tt.opts)
			if err != nil {
				t.Fatalf("Launch: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Launch() = %q, want %q", got, tt.want)
			}
		})
	}
}

// What a worktree already carries decides, and no session id is ever asked of a
// person: the one conversation is named outright, and several reach the agent's
// own list by naming none.
func TestAgent(t *testing.T) {
	tests := []struct {
		name string
		has  []sessions.Session
		want []string
	}{
		{"none starts one", nil, []string{"claude", "--permission-mode", "auto", "--name=bd-1"}},
		{"one is returned to", []sessions.Session{{ID: "s1"}}, []string{"claude", "--resume", "s1"}},
		{"several reach the list", []sessions.Session{{ID: "s1"}, {ID: "s2"}}, []string{"claude", "--resume"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Env{Conversations: stubConversations{list: tt.has}}
			bead, _ := e.Resolve("bd-1")

			got, err := e.Agent(State{Target: bead, Path: "/wt", Exists: true}, Options{})
			if err != nil {
				t.Fatalf("Agent: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Agent() = %q, want %q", got, tt.want)
			}
		})
	}
}

// An agent that cannot say what a worktree carries is not asked to guess.
func TestAgentRefusesAnUnreadableWorktree(t *testing.T) {
	e := Env{Conversations: stubConversations{err: errors.New("no transcript store")}}
	if _, err := e.Agent(State{Path: "/wt", Exists: true}, Options{}); err == nil {
		t.Error("Agent() with an unreadable store: want an error")
	}
}

type stubConversations struct {
	list []sessions.Session
	err  error
}

func (s stubConversations) List(string) ([]sessions.Session, error) { return s.list, s.err }
