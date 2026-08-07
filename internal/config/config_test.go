package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// What a user spells out, which no other test may derive from the code: the two
// file names, and the branches the defaults name.
const (
	repoFile    = ".work.toml"
	defaultDir  = ".worktrees"
	userRelPath = "work/config.toml"

	ticketBranch      = "bd-42-port-work-to-go"
	untitledBranch    = "bd-42"
	pullRequestBranch = "pr-7"
)

// Both files are layered over the defaults, the repository's on top.
func TestLoadLayers(t *testing.T) {
	tests := []struct {
		name       string
		user, repo string
		want       string
	}{
		{"neither", "", "", defaultDir},
		{"the user alone", "mine", "", "mine"},
		{"the repository alone", "", "ours", "ours"},
		{"the repository over the user", "mine", "ours", "ours"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, home := t.TempDir(), t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", home)
			if tt.user != "" {
				write(t, filepath.Join(home, userRelPath), directory(tt.user))
			}
			if tt.repo != "" {
				write(t, filepath.Join(repo, repoFile), directory(tt.repo))
			}

			got, err := Load(repo)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got.Worktree.Directory != tt.want {
				t.Errorf("worktree.directory = %q, want %q", got.Worktree.Directory, tt.want)
			}
		})
	}
}

// A file that sets one key leaves the rest to the layer below it, so a table
// present but empty changes nothing.
func TestLoadLeavesUnnamedKeys(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	write(t, filepath.Join(repo, repoFile), "[worktree]\n[branch]\n")

	got, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// A pattern is identified by what it matches as, which is derived from the
	// whole of it.
	if !reflect.DeepEqual(got.Worktree, Default().Worktree) ||
		got.Branch.ticket().matcher != defaults.ticket().matcher ||
		got.Branch.pullRequest().matcher != defaults.pullRequest().matcher {
		t.Errorf("Load() = %+v, want the defaults", got)
	}
}

// The compiled-in patterns name the branches work named before they were
// settings, and read those branches back. A zero Branch is those patterns, so an
// Env built without a loaded Config still names one.
func TestDefaultBranches(t *testing.T) {
	var b Branch
	if got := b.Ticket("bd-42", "port-work-to-go"); got != ticketBranch {
		t.Errorf("Ticket() = %q, want %q", got, ticketBranch)
	}
	// A title that slugs to nothing leaves the id alone.
	if got := b.Ticket("bd-42", ""); got != untitledBranch {
		t.Errorf("Ticket() untitled = %q, want %q", got, untitledBranch)
	}
	if got := b.PullRequest("7"); got != pullRequestBranch {
		t.Errorf("PullRequest() = %q, want %q", got, pullRequestBranch)
	}

	for _, branch := range []string{ticketBranch, untitledBranch} {
		if !b.Owns("bd-42", branch) {
			t.Errorf("Owns(bd-42, %q) = false, want true", branch)
		}
	}
	// A shorter id is not a prefix of a longer one's branch.
	if b.Owns("bd-4", ticketBranch) {
		t.Errorf("Owns(bd-4, %q) = true, want false", ticketBranch)
	}
	if got, ok := b.NumberIn(pullRequestBranch); !ok || got != "7" {
		t.Errorf("NumberIn(%q) = %q, %v; want 7", pullRequestBranch, got, ok)
	}
	if got, ok := b.NumberIn(ticketBranch); ok {
		t.Errorf("NumberIn(%q) = %q, want no pull request", ticketBranch, got)
	}
}

// A pattern is the matcher too: the identifier is filled in and every other
// value stands for anything, so a prefix costs nothing and a ticket retitled
// after its worktree was made still finds it.
func TestConfiguredBranchesMatch(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	write(t, filepath.Join(repo, repoFile), branch(`feature/{{.ID}}-{{.Slug}}`, `review/{{.Number}}`))

	c, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := c.Branch.Ticket("bd-42", "port-work-to-go"); got != "feature/bd-42-port-work-to-go" {
		t.Errorf("Ticket() = %q, want the pattern's branch", got)
	}
	for _, b := range []string{"feature/bd-42-port-work-to-go", "feature/bd-42-anything-at-all"} {
		if !c.Branch.Owns("bd-42", b) {
			t.Errorf("Owns(bd-42, %q) = false, want true", b)
		}
	}
	// The prefix is part of the branch, so what lacks it is nobody's.
	if c.Branch.Owns("bd-42", "bd-42-port-work-to-go") {
		t.Error("Owns() = true for a branch without the configured prefix")
	}

	if got := c.Branch.PullRequest("7"); got != "review/7" {
		t.Errorf("PullRequest() = %q, want the pattern's branch", got)
	}
	if got, ok := c.Branch.NumberIn("review/7"); !ok || got != "7" {
		t.Errorf("NumberIn(review/7) = %q, %v; want 7", got, ok)
	}
	if got, ok := c.Branch.NumberIn(pullRequestBranch); ok {
		t.Errorf("NumberIn(%q) = %q, want no pull request", pullRequestBranch, got)
	}
}

// An unset Directory is the default, so an Env built without a loaded Config
// still puts a worktree where one belongs rather than in the repository root.
func TestZeroWorktreeIsTheDefault(t *testing.T) {
	if got := (Worktree{}).Dir(); got != defaultDir {
		t.Errorf("Worktree{}.Dir() = %q, want %q", got, defaultDir)
	}
}

func TestLoadRefusals(t *testing.T) {
	tests := []struct{ name, body, want string }{
		{"an unknown key", "[worktree]\ndirectry = \"trees\"\n", "unknown setting"},
		{"an unknown table", "[nonsense]\nkey = 1\n", "unknown setting"},
		{"a value of the wrong type", "[worktree]\ndirectory = 3\n", "directory"},
		{"a key spelled in another case", "[worktree]\nDirectory = \"trees\"\n", "unknown setting"},
		{"a table spelled in another case", "[Worktree]\ndirectory = \"trees\"\n", "unknown setting"},
		{"a directory outside the repository", directory("../trees"), "not a directory inside"},
		{"an absolute directory", directory("/tmp/trees"), "not a directory inside"},
		{"the repository root, unnamed", directory(""), "not a directory inside"},
		{"the repository root, as a dot", directory("."), "not a directory inside"},
		{"the repository root, with a trailing slash", directory("./"), "not a directory inside"},
		{"the repository root, by traversal", directory("trees/.."), "not a directory inside"},
		{"git's own directory", directory(".git"), "git's own directory"},
		{"a directory under git's own", directory(".git/worktrees"), "git's own directory"},
		{"a pattern that does not parse", "[branch]\nticket = \"{{.ID\"\n", "branch.ticket"},
		{"a pattern that is not a string", "[branch]\nticket = 3\n", "ticket"},
		{"a ticket pattern without its id", "[branch]\nticket = \"feature/{{.Slug}}\"\n", "places no {{.ID}}"},
		// The pattern places the id, but only where a ticket with no slug would not
		// reach, and that ticket's branch would then stand for every ticket.
		{"an id only some tickets reach", "[branch]\nticket = \"{{with .Slug}}{{$.ID}}-{{.}}{{end}}\"\n", "places no {{.ID}}"},
		{"a pull request pattern without its number", "[branch]\npull-request = \"pr-{{.ID}}\"\n", "{{.Number}}"},
		{"a branch opening with a dash", "[branch]\nticket = \"-{{.ID}}\"\n", "dash"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			write(t, filepath.Join(repo, repoFile), tt.body)

			_, err := Load(repo)
			if err == nil {
				t.Fatalf("Load(%q) = no error, want one", tt.body)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Load(%q) = %v, want it to name %q", tt.body, err, tt.want)
			}
		})
	}
}

// The containment check is not lexical: a repository is cloned with its file
// and its symlinks, and a checkout may not land outside the repository.
func TestLoadRefusesADirectorySymlinkedOut(t *testing.T) {
	repo, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(repo, "trees")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	write(t, filepath.Join(repo, repoFile), directory("trees"))

	_, err := Load(repo)
	if err == nil || !strings.Contains(err.Error(), "resolves outside") {
		t.Errorf("Load() = %v, want the escaping symlink refused", err)
	}
}

// A repository reached through a symlink is still its own containing directory.
func TestLoadAllowsASymlinkedRepository(t *testing.T) {
	real, link := t.TempDir(), filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(real, "trees"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	write(t, filepath.Join(real, repoFile), directory("trees"))

	if _, err := Load(link); err != nil {
		t.Errorf("Load() = %v, want a repository behind a symlink to load", err)
	}
}

// A value the repository replaces is never the one work uses, so it is not the
// one judged: only what the merge arrives at has to be usable.
func TestLoadValidatesTheMergedValue(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	write(t, filepath.Join(home, userRelPath), directory("/tmp/trees"))
	write(t, filepath.Join(repo, repoFile), directory("trees"))

	got, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Worktree.Directory != "trees" {
		t.Errorf("worktree.directory = %q, want %q", got.Worktree.Directory, "trees")
	}
}

// Two files can carry the same unusable value, so the message names the one that
// did rather than leaving the reader to guess.
func TestLoadNamesTheFileAtFault(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	user := filepath.Join(home, userRelPath)
	write(t, user, directory("/tmp/trees"))

	_, err := Load(repo)
	if err == nil || !strings.Contains(err.Error(), user) {
		t.Errorf("Load() = %v, want it to name %q", err, user)
	}
}

func directory(dir string) string {
	return fmt.Sprintf("[worktree]\ndirectory = %q\n", dir)
}

func branch(ticket, pullRequest string) string {
	return fmt.Sprintf("[branch]\nticket = %q\npull-request = %q\n", ticket, pullRequest)
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
