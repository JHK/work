package work

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/beads"
	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/sessions"
)

// The vetting guards the claim, so it runs wherever a ticket's worktree is
// about to be created, whatever the invocation opens on and whether or not it
// will claim. The repository here is not one, so anything reaching the
// provisioning fails rather than making a worktree.
func TestEnterVetsEveryWay(t *testing.T) {
	t.Setenv("VISUAL", "vi")
	e := Env{Repo: filepath.Join(t.TempDir(), "not-a-repo"), Config: config.Default()}
	s := State{
		Target: Target{Kind: KindBead, ID: "bd-1", Name: "bd-1"},
		Bead:   beads.Bead{ID: "bd-1", Status: "closed", Type: "task", AcceptanceCriteria: "It works"},
	}

	tests := []struct {
		name string
		opts Options
	}{
		{"a session", Options{}},
		{"an agent", Options{Action: ActionAgent}},
		{"a shell", Options{Action: ActionShell}},
		{"an editor", Options{Action: ActionEditor}},
		{"a diff", Options{Action: ActionDiff}},
		{"no claim", Options{NoClaim: true}},
		{"a shell and no claim", Options{Action: ActionShell, NoClaim: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := e.enter(s, tt.opts)
			if err == nil || !strings.Contains(err.Error(), "already closed") {
				t.Errorf("enter(%+v) = %v; want the closed bead refused, saying which rule it broke", tt.opts, err)
			}
		})
	}
}

// A worktree that already exists is re-entered whatever the flags say about
// claiming: there is nothing to vet, nothing to create and nothing to claim.
// What it opens on is action.enter's to name, and a flag wins over it.
func TestEnterOpensOn(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")
	t.Setenv("VISUAL", "vi")
	s := State{Target: Target{Kind: KindBead, ID: "bd-1", Name: "bd-1"}, Path: "/wt", Exists: true}

	tests := []struct {
		name  string
		enter Action // unset where the flags name the action
		opts  Options
		want  []string
	}{
		{"a shell by default", "", Options{}, []string{"/usr/bin/fish"}},
		{"asked for outright", "", Options{Action: ActionShell}, []string{"/usr/bin/fish"}},
		{"no claim changes nothing", "", Options{NoClaim: true}, []string{"/usr/bin/fish"}},
		{"an editor", "", Options{Action: ActionEditor}, []string{"vi", "/wt"}},
		{"the agent, on what the worktree carries", "", Options{Action: ActionAgent}, []string{"claude", "--resume", "s1"}},
		{"the agent action.enter names", ActionAgent, Options{}, []string{"claude", "--resume", "s1"}},
		{"the editor action.enter names", ActionEditor, Options{}, []string{"vi", "/wt"}},
		{"a flag over action.enter", ActionAgent, Options{Action: ActionShell}, []string{"/usr/bin/fish"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Action.EnterName = tt.enter
			e := Env{
				Repo: filepath.Join(t.TempDir(), "not-a-repo"), Config: cfg,
				Conversations: stubConversations{list: []sessions.Session{{ID: "s1"}}},
			}

			got, err := e.enter(s, tt.opts)
			if err != nil {
				t.Fatalf("enter(%+v) on action.enter %q: %v", tt.opts, tt.enter, err)
			}
			if !slices.Equal(got.Handoff.Run, tt.want) {
				t.Errorf("enter(%+v) on action.enter %q runs %q; want %q", tt.opts, tt.enter, got.Handoff.Run, tt.want)
			}
		})
	}
}

// The editor is rendered ahead of the vetting whichever named it, so a key
// naming it on a machine with no editor leaves nothing created or claimed, as
// the flag does.
func TestEnterRefusesAConfiguredEditorFirst(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	cfg := config.Default()
	cfg.Action.CreateName = ActionEditor
	e := Env{Repo: filepath.Join(t.TempDir(), "not-a-repo"), Config: cfg}
	s := State{
		Target: Target{Kind: KindBead, ID: "bd-1", Name: "bd-1"},
		Bead:   beads.Bead{ID: "bd-1", Status: "open", Type: "task", AcceptanceCriteria: "It works"},
	}

	_, err := e.enter(s, Options{})
	if err == nil || !strings.Contains(err.Error(), "open.editor") {
		t.Errorf("enter = %v; want the editor refused before anything is created", err)
	}
}

// A worktree only just created opens on what action.create names, the launcher
// by default rather than the shell an existing one gets, and --agent leaves
// that alone: there is no conversation to return to yet. A pull request is the
// target here, its branch already local, so provisioning is git's alone and
// neither bd nor gh is asked anything.
func TestEnterCreatesAndOpensOn(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")
	t.Setenv("VISUAL", "vi")
	repo := initRepo(t)
	base := gitCmd(t, repo, "rev-parse", "HEAD")
	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// An agent that would be consulted fails the case rather than answering it.
	e.Conversations = stubConversations{err: errors.New("a fresh worktree carries none")}

	launcher := []string{"claude", "--name=PR #7"}
	tests := []struct {
		branch string
		create Action // unset where the flags name the action
		opts   Options
		want   []string
	}{
		{"pr-7-unnamed", "", Options{}, launcher},
		{"pr-7-agent", "", Options{Action: ActionAgent}, launcher},
		{"pr-7-shell", "", Options{Action: ActionShell}, []string{"/usr/bin/fish"}},
		{"pr-7-editor", "", Options{Action: ActionEditor}, []string{"vi", filepath.Join(repo, defaultDir, "pr-7-editor")}},
		{"pr-7-diff", "", Options{Action: ActionDiff}, []string{"git", "diff", base}},
		{"pr-7-created-shell", ActionShell, Options{}, []string{"/usr/bin/fish"}},
		{"pr-7-created-diff", ActionDiff, Options{}, []string{"git", "diff", base}},
		{"pr-7-flagged-over-created", ActionShell, Options{Action: ActionAgent}, launcher},
	}
	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			// One branch per case: each worktree is provisioned for real, and a branch
			// is checked out once.
			gitCmd(t, repo, "branch", tt.branch)
			e.Config.Action.CreateName = tt.create
			path := filepath.Join(repo, defaultDir, tt.branch)
			s := State{Target: Target{Kind: KindPR, ID: "7", Name: tt.branch}, Path: path}

			got, err := e.enter(s, tt.opts)
			if err != nil {
				t.Fatalf("enter(%+v) on action.create %q: %v", tt.opts, tt.create, err)
			}
			if !slices.Equal(got.Handoff.Run, tt.want) {
				t.Errorf("enter(%+v) on action.create %q runs %q; want %q", tt.opts, tt.create, got.Handoff.Run, tt.want)
			}
		})
	}
}

// The diff is against the point the worktree's branch forked from, so what has
// landed on the main checkout since is not in it and the worktree's own work is,
// committed and uncommitted alike.
func TestEnterDiffsAgainstTheBase(t *testing.T) {
	repo := initRepo(t)
	base := gitCmd(t, repo, "rev-parse", "HEAD")
	wt := filepath.Join(repo, defaultDir, "bd-1")
	gitCmd(t, repo, "worktree", "add", "-b", "bd-1", wt)
	gitCmd(t, wt, "commit", "--allow-empty", "-m", "the worktree's own work")
	gitCmd(t, repo, "commit", "--allow-empty", "-m", "the main checkout has moved on")

	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	s := State{Target: Target{Kind: KindBead, ID: "bd-1", Name: "bd-1"}, Path: wt, Exists: true}

	got, err := e.enter(s, Options{Action: ActionDiff})
	if err != nil {
		t.Fatalf("enter with a diff: %v", err)
	}
	if want := []string{"git", "diff", base}; !slices.Equal(got.Handoff.Run, want) {
		t.Errorf("enter with a diff runs %q; want %q", got.Handoff.Run, want)
	}
}
