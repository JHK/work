package work

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/beads"
	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/forge"
	"github.com/JHK/work-cli/internal/git"
)

// defaultDir is where worktrees go with nothing configured.
var defaultDir = config.Default().Worktree.Directory

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
	// traverse, or that git would reject, has to be refused up front.
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

// Discovery is what git reports, so a worktree outside the configured directory
// is offered and entered where it is, and the bead it holds is not offered as
// fresh.
func TestWorktreeDiscovery(t *testing.T) {
	repo := initRepo(t)
	outside := filepath.Join(t.TempDir(), "elsewhere-wt")
	gitCmd(t, repo, "worktree", "add", "-b", "one-oxc-outside", outside)

	e := Env{Repo: repo, Config: config.Default()}
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

	// A target with no worktree is created under the configured directory,
	// whatever the ones already open do.
	fresh := e.Inspect(Target{Kind: KindBead, ID: "one-abc", Name: "one-abc"})
	if want := filepath.Join(repo, defaultDir, "one-abc"); fresh.Exists || fresh.Path != want {
		t.Errorf("Inspect(one-abc) = exists %v at %q, want a fresh path at %q", fresh.Exists, fresh.Path, want)
	}
}

// The setting decides where a new worktree goes, and nothing else: one created
// under an earlier value is still found and entered where it sits.
func TestConfiguredWorktreeDirectory(t *testing.T) {
	repo := initRepo(t)
	body := []byte("[worktree]\ndirectory = \"trees\"\n")
	if err := os.WriteFile(filepath.Join(repo, config.RepoFile), body, 0o644); err != nil {
		t.Fatalf("write %s: %v", config.RepoFile, err)
	}

	e, err := Open(repo)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	target := Target{Kind: KindBead, ID: "one-abc", Name: "one-abc"}
	// From e.Repo, not repo: git reports the repository with symlinks resolved,
	// and a worktree that does not exist yet cannot be resolved to compare.
	trees := filepath.Join(e.Repo, "trees", "one-abc")
	if s := e.Inspect(target); s.Path != trees {
		t.Errorf("Inspect() = %q, want %q", s.Path, trees)
	}

	gitCmd(t, repo, "worktree", "add", "-b", "one-abc", trees)
	e.Config.Worktree.Directory = "elsewhere"
	s := e.Inspect(target)
	if !s.Exists || !git.SameDir(s.Path, trees) {
		t.Errorf("Inspect() = exists %v at %q, want the worktree at %q", s.Exists, s.Path, trees)
	}
}

// Run from inside a linked worktree, work still nests new worktrees under the
// main checkout.
func TestOpenFromLinkedWorktree(t *testing.T) {
	repo := initRepo(t)
	wt := filepath.Join(repo, defaultDir, "one-abc")
	gitCmd(t, repo, "worktree", "add", "-b", "one-abc", wt)

	e, err := Open(wt)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !git.SameDir(e.Repo, repo) {
		t.Errorf("Open(%q).Repo = %q, want %q", wt, e.Repo, repo)
	}
}

// A worktree a listing already found is taken from it. The repository here is
// not one, so a worktree still asked for would not be found at all.
func TestInspectAtTakesTheListedWorktree(t *testing.T) {
	e := Env{Repo: filepath.Join(t.TempDir(), "not-a-repo"), Config: config.Default()}
	target := Target{Kind: KindBead, ID: "one", Name: "one"}

	if s := e.inspectAt(target, "/elsewhere/wt"); !s.Exists || s.Path != "/elsewhere/wt" {
		t.Errorf("inspectAt() = exists %v at %q, want the listed worktree", s.Exists, s.Path)
	}
	// An empty path is a target without one, which is where a fresh worktree goes.
	if s := e.inspectAt(target, ""); s.Exists || s.Path != filepath.Join(e.Repo, defaultDir, "one") {
		t.Errorf("inspectAt() = exists %v at %q, want a fresh path under %s", s.Exists, s.Path, defaultDir)
	}
}

// The picker's three sources meet in one list: a worktree is offered once,
// under whatever title its own adapter gave it, and what has none is offered
// fresh.
func TestCandidates(t *testing.T) {
	worktrees := []git.Worktree{
		{Path: "/wt/one", Branch: "one-a-slug"},
		{Path: "/wt/pr-7", Branch: "pr-7"},
		{Path: "/wt/loose", Branch: "loose"},
	}
	// Every bead names a branch and titles a row; only the ready ones are offered
	// fresh, and "one" is claimed already.
	known := []beads.Bead{
		{ID: "one", Title: "The first bead"},
		{ID: "two", Title: "The second bead"},
		{ID: "three", Title: "The third bead"},
	}
	ready := []beads.Bead{{ID: "one", Title: "The first bead"}, {ID: "two", Title: "The second bead"}}
	pulls := []forge.PR{{Number: 7, Title: "Seventh pull request"}, {Number: 9, Title: "Ninth pull request"}}

	want := []Candidate{
		{Target: Target{Kind: KindBead, ID: "one", Name: "one"}, Label: "The first bead", Open: true},
		{Target: prTarget("7"), Label: "Seventh pull request", Open: true},
		{Target: Target{Kind: KindPlain, ID: "/wt/loose", Name: "loose"}, Open: true},
		{Target: prTarget("9"), Label: "Ninth pull request"},
		{Target: Target{Kind: KindBead, ID: "two", Name: "two"}, Label: "The second bead", ready: true},
	}
	got := candidates(worktrees, known, ready, pulls)
	if len(got) != len(want) {
		t.Fatalf("candidates() = %+v, want %d rows", got, len(want))
	}
	for i := range want {
		got[i].path = "" // where git put it is not what this test is about
		if got[i] != want[i] {
			t.Errorf("row %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	// A forge that would not answer costs the pull request rows and their titles,
	// nothing else.
	got = candidates(worktrees, known, ready, nil)
	if len(got) != 4 {
		t.Fatalf("candidates() without gh = %+v, want the worktrees and the ready bead", got)
	}
	if pr := got[1]; pr.Target != prTarget("7") || pr.Label != "" || !pr.Open {
		t.Errorf("pr row without gh = %+v, want an untitled open worktree", pr)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	// Whoever runs the tests has settings of their own, which Open would read.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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
