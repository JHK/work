package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The two names a user spells out, which no other test may derive from the code.
const (
	repoFile    = ".work.toml"
	defaultDir  = ".worktrees"
	userRelPath = "work/config.toml"
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
	write(t, filepath.Join(repo, repoFile), "[worktree]\n")

	got, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, Default()) {
		t.Errorf("Load() = %+v, want %+v", got, Default())
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

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
