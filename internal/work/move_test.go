package work

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/testenv"
)

// opened puts a worktree in a repository, on a branch spelled as its name, and
// hands back the candidate for it.
func opened(t *testing.T, e Env, name string) Candidate {
	t.Helper()
	testenv.Git(t, e.Repo, "worktree", "add", "-b", name, filepath.Join(e.Repo, defaultDir, name))
	c, err := e.Resolve(name)
	if err != nil {
		t.Fatalf("Resolve(%q): %v", name, err)
	}
	return c
}

// Moving is git's two commands and nothing besides: the directory moves and the
// branch takes the destination's last element with it.
func TestMoveTakesTheWorktreeAndItsBranch(t *testing.T) {
	repo := testenv.InitRepo(t)
	e, s := bare(t, repo)
	c := opened(t, e, "scratch")

	m, err := e.Move(c, "settled")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}

	want := filepath.Join(repo, defaultDir, "settled")
	if !git.SameDir(m.To, want) || m.Was != "scratch" || m.Now != "settled" {
		t.Errorf("Move = %+v; want it at %q on branch settled", m, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("%s is not there: %v", want, err)
	}
	if _, err := os.Stat(c.path); !os.IsNotExist(err) {
		t.Errorf("%s is still there: %v", c.path, err)
	}
	if list := testenv.Git(t, repo, "worktree", "list"); !strings.Contains(list, "settled") {
		t.Errorf("git does not list the worktree where it landed:\n%s", list)
	}
	if branches := testenv.Git(t, repo, "branch", "--list", "scratch"); branches != "" {
		t.Errorf("branch scratch is still there: %q", branches)
	}
	// The worktree is reached under the name it took, the branch being the address.
	if _, err := e.Resolve("settled"); err != nil {
		t.Errorf("Resolve(settled): %v", err)
	}
	// No resolver was asked to prepare or create anything and no action ran: a
	// worktree moving reaches neither seam.
	if len(s.seen) != 0 {
		t.Errorf("moving ran %q; want git's two commands alone", s.seen)
	}
}

// A bare name lands the worktree beside where it sits, whatever directory that
// is; one carrying a separator is a path, read from where work was invoked.
func TestMoveReadsTheDestination(t *testing.T) {
	elsewhere := t.TempDir()
	tests := []struct {
		name string
		dest string
		want func(repo string) string
	}{
		{
			"a bare name",
			"settled",
			func(repo string) string { return filepath.Join(repo, defaultDir, "settled") },
		},
		{
			"a path of its own",
			filepath.Join(elsewhere, "settled"),
			func(string) string { return filepath.Join(elsewhere, "settled") },
		},
		{
			"a relative path",
			filepath.Join("..", "beside"),
			func(repo string) string { return filepath.Join(repo, "beside") },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testenv.InitRepo(t)
			e, _ := bare(t, repo)
			c := opened(t, e, "scratch")
			// A relative destination is read from here, never from the worktree.
			t.Chdir(filepath.Join(repo, defaultDir))

			m, err := e.Move(c, tt.dest)
			if err != nil {
				t.Fatalf("Move(%q): %v", tt.dest, err)
			}
			if want := tt.want(repo); !git.SameDir(m.To, want) {
				t.Errorf("Move(%q) landed at %q; want %q", tt.dest, m.To, want)
			}
			// The last element names the branch wherever the worktree went.
			if want := filepath.Base(m.To); m.Now != want {
				t.Errorf("Move(%q) left the branch %q; want %q", tt.dest, m.Now, want)
			}
		})
	}
}

// A destination already occupied and a branch name already taken are each
// refused before anything moves, so neither half of the move lands.
func TestMoveRefusesAnOccupiedDestination(t *testing.T) {
	repo := testenv.InitRepo(t)
	e, _ := bare(t, repo)
	c := opened(t, e, "scratch")
	// git moves a worktree inside a directory already there rather than refusing it,
	// so a plain one is the case the refusal is for.
	if err := os.Mkdir(filepath.Join(repo, defaultDir, "occupied"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	testenv.Git(t, repo, "branch", "spoken-for")

	tests := []struct {
		name, dest, want string
	}{
		{"a directory already there", "occupied", "occupied already exists"},
		{"a branch already there", "spoken-for", "branch spoken-for already exists"},
		{"where it already sits", "scratch", "already exists"},
		{"a name no worktree could be made for", "..", "not a usable worktree name"},
		{"nothing at all", "", "no destination given"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := e.Move(c, tt.dest); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Move(%q) = %v; want a refusal naming %q", tt.dest, err, tt.want)
			}
			if _, err := os.Stat(c.path); err != nil {
				t.Errorf("the worktree moved despite the refusal: %v", err)
			}
			if branches := testenv.Git(t, repo, "branch", "--list", "scratch"); branches == "" {
				t.Error("the branch was renamed despite the refusal")
			}
		})
	}
}

// Moving the main checkout is git's to refuse, and the refusal has to survive
// being asked for from inside a worktree that sits under it.
func TestGitRefusesToMoveTheMainCheckout(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Git(t, repo, "branch", "--move", "trunk")
	e, _ := bare(t, repo)
	opened(t, e, "scratch")

	c, err := e.Resolve("trunk")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	t.Chdir(filepath.Join(repo, defaultDir, "scratch"))
	if _, err := e.Move(c, filepath.Join(t.TempDir(), "elsewhere")); err == nil ||
		!strings.Contains(err.Error(), "main working tree") {
		t.Fatalf("Move = %v; want git's own refusal", err)
	}
	if head := testenv.Git(t, repo, "rev-parse", "--abbrev-ref", "HEAD"); head != "trunk" {
		t.Errorf("the main checkout is on %q; want it left standing on trunk", head)
	}
}

// A detached worktree has no branch to take with it, so the directory moves
// alone and what moved names none.
func TestMoveADetachedWorktreeTakesNoBranch(t *testing.T) {
	repo := testenv.InitRepo(t)
	e, _ := bare(t, repo)
	path := filepath.Join(repo, defaultDir, "adrift")
	testenv.Git(t, repo, "worktree", "add", "--detach", path)
	c, err := e.Resolve("adrift")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	m, err := e.Move(c, "settled")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if m.Was != "" || m.Now != "" || m.Renamed() {
		t.Errorf("Move = %+v; want no branch renamed", m)
	}
	if _, err := os.Stat(filepath.Join(repo, defaultDir, "settled")); err != nil {
		t.Errorf("the worktree did not land: %v", err)
	}
}
