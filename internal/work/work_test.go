package work

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/git"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		arg  string
		kind Kind
		id   string
		name string
	}{
		{"bd-42", KindBead, "bd-42", "bd-42"},
		{"7", KindPR, "7", "pr-7"},
		{"https://github.com/o/r/pull/91", KindPR, "91", "pr-91"},
		{"https://github.com/o/r/pull/91/files", KindPR, "91", "pr-91"},
		{"pull-request-1", KindBead, "pull-request-1", "pull-request-1"},
		{"007", KindPR, "7", "pr-7"},
	}
	for _, tt := range tests {
		got, err := Resolve(tt.arg)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", tt.arg, err)
		}
		if got.Kind != tt.kind || got.ID != tt.id || got.Name != tt.name {
			t.Errorf("Resolve(%q) = %+v, want kind %v id %q name %q", tt.arg, got, tt.kind, tt.id, tt.name)
		}
	}

	// An identifier becomes a directory name and a refspec; anything that would
	// leave .worktrees, or that git would reject, has to be refused up front.
	for _, arg := range []string{"", "..", "../../etc", "a/b", "/", ".", "-5", "--yes", "docs/pull/99-notes.md"} {
		if got, err := Resolve(arg); err == nil {
			t.Errorf("Resolve(%q) = %+v, want an error", arg, got)
		}
	}
}

// A worktree is read off the branch it has checked out, whatever directory it
// sits in.
func TestTargetAt(t *testing.T) {
	ids := []string{"one", "one-two", "1234"}
	tests := []struct {
		name   string
		branch string
		kind   Kind
		id     string
		label  string
	}{
		{"a bare bead id", "one", KindBead, "one", "one"},
		{"an id and a title slug", "one-and-a-slug", KindBead, "one", "one"},
		{"the longest id wins", "one-two-three", KindBead, "one-two", "one-two"},
		// A numeric branch stays a bead: the PR heuristic belongs to command-line
		// input, not to a worktree that already exists.
		{"a numeric id", "1234", KindBead, "1234", "1234"},
		{"a pull request", "pr-7", KindPR, "7", "pr-7"},
		{"a non-canonical pr", "pr-007", KindPlain, "/wt", "pr-007"},
		{"an unknown branch", "some-branch", KindPlain, "/wt", "some-branch"},
		{"detached", "", KindPlain, "/wt", "wt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := targetAt(git.Worktree{Path: "/wt", Branch: tt.branch}, ids)
			want := Target{Kind: tt.kind, ID: tt.id, Name: tt.label}
			if got != want {
				t.Errorf("targetAt(%q) = %+v, want %+v", tt.branch, got, want)
			}
		})
	}

	// Without the tracker every worktree still lists, as a plain one.
	if got := targetAt(git.Worktree{Path: "/wt", Branch: "one"}, nil); got.Kind != KindPlain {
		t.Errorf("targetAt without ids = %+v, want a plain worktree", got)
	}
}

// A named target finds its worktree by branch, wherever that worktree sits.
func TestMatches(t *testing.T) {
	tests := []struct {
		arg, branch string
		want        bool
	}{
		{"bd-42", "bd-42", true},
		{"bd-42", "bd-42-port-work-to-go", true},
		{"bd-42", "bd-42.1-a-child", false},
		{"bd-4", "bd-42", false},
		{"bd-42", "", false},
		{"7", "pr-7", true},
		{"7", "pr-70", false},
	}
	for _, tt := range tests {
		target, err := Resolve(tt.arg)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", tt.arg, err)
		}
		w := git.Worktree{Path: "/elsewhere/wt", Branch: tt.branch}
		if got := target.matches(w); got != tt.want {
			t.Errorf("Resolve(%q).matches(%q) = %v, want %v", tt.arg, tt.branch, got, tt.want)
		}
	}

	// A plain worktree is only itself, so the path is what identifies it.
	plain := targetAt(git.Worktree{Path: "/elsewhere/wt", Branch: "some-branch"}, nil)
	if !plain.matches(git.Worktree{Path: "/elsewhere/wt", Branch: "some-branch"}) {
		t.Error("a plain target does not match its own worktree")
	}
	if plain.matches(git.Worktree{Path: "/other/wt", Branch: "some-branch"}) {
		t.Error("a plain target matches another worktree on the same branch")
	}
}

// Discovery is what git reports, so a worktree outside .worktrees is offered
// and entered where it is, and the bead it holds is not offered as fresh.
func TestWorktreeDiscovery(t *testing.T) {
	repo := initRepo(t)
	outside := filepath.Join(t.TempDir(), "elsewhere-wt")
	gitCmd(t, repo, "worktree", "add", "-b", "one-oxc-outside", outside)

	e := Env{Repo: repo}
	// The main checkout is git's first entry and never a place to work.
	list, err := git.Linked(repo)
	if err != nil {
		t.Fatalf("Linked: %v", err)
	}
	if len(list) != 1 || !git.SameDir(list[0].Path, outside) {
		t.Fatalf("Linked = %+v, want only %q", list, outside)
	}

	target, err := Resolve("one-oxc")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	s := e.Inspect(target)
	if !s.Exists || !git.SameDir(s.Path, outside) {
		t.Errorf("Inspect(one-oxc) = exists %v at %q, want the worktree at %q", s.Exists, s.Path, outside)
	}

	// A target with no worktree is created under .worktrees, whatever the ones
	// already open do.
	fresh := e.Inspect(Target{Kind: KindBead, ID: "one-abc", Name: "one-abc"})
	if fresh.Exists || fresh.Path != filepath.Join(repo, worktreesDir, "one-abc") {
		t.Errorf("Inspect(one-abc) = exists %v at %q, want a fresh path under %s", fresh.Exists, fresh.Path, worktreesDir)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitCmd(t, dir, "init", "-b", "main")
	gitCmd(t, dir, "commit", "--allow-empty", "-m", "root")
	return dir
}

func gitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// The user's own config would otherwise reach in: commit.gpgsign alone makes
	// the empty commit depend on a working key.
	cmd.Env = append(cmd.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func TestSlug(t *testing.T) {
	tests := []struct{ title, want string }{
		{"Port work to Go", "port-work-to-go"},
		{"Rate-limit /api/upload", "rate-limit-api-upload"},
		{"  Trailing punctuation!!  ", "trailing-punctuation"},
		{strings.Repeat("ab ", 30), strings.Repeat("ab-", 13) + "a"},
		{"—", ""},
	}
	for _, tt := range tests {
		if got := slug(tt.title); got != tt.want {
			t.Errorf("slug(%q) = %q, want %q", tt.title, got, tt.want)
		}
		if len(slug(tt.title)) > slugLen {
			t.Errorf("slug(%q) is longer than %d", tt.title, slugLen)
		}
	}
}

func TestBranch(t *testing.T) {
	pr, _ := Resolve("7")
	if got, err := (State{Target: pr}).Branch(); err != nil || got != "pr-7" {
		t.Errorf("PR branch = %q, %v", got, err)
	}

	bead, _ := Resolve("bd-42")
	s := State{Target: bead}
	s.Bead.Title = "Port work to Go"
	if got, err := s.Branch(); err != nil || got != "bd-42-port-work-to-go" {
		t.Errorf("bead branch = %q, %v", got, err)
	}
}
