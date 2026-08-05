// Package git is the read-only-by-default adapter over the git worktree
// commands work needs.
package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JHK/work-cli/internal/run"
)

// Root reports the main checkout of the repository containing dir, so that
// running work from inside a linked worktree still nests new worktrees under the
// main checkout rather than under the current one.
func Root(dir string) (string, error) {
	out, err := git(dir, "rev-parse", "--path-format=absolute", "--git-dir", "--git-common-dir")
	if err != nil {
		return "", err
	}
	gitDir, commonDir, _ := strings.Cut(out, "\n")

	// The two differ only inside a linked worktree, where the main checkout is
	// the first worktree git lists. Everywhere else --show-toplevel is right
	// whatever layout the git directory has.
	if filepath.Clean(gitDir) == filepath.Clean(commonDir) {
		return git(dir, "rev-parse", "--path-format=absolute", "--show-toplevel")
	}
	paths, err := Worktrees(dir)
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("git worktree list: no worktrees")
	}
	return paths[0], nil
}

// IsWorktree reports whether path is a worktree git has registered, as opposed
// to a directory that merely sits where one belongs.
func IsWorktree(repo, path string) bool {
	paths, err := Worktrees(repo)
	if err != nil {
		return false
	}
	want := resolve(path)
	for _, p := range paths {
		if resolve(p) == want {
			return true
		}
	}
	return false
}

// resolve normalises a path for comparison; git reports worktrees with symlinks
// already resolved.
func resolve(path string) string {
	if p, err := filepath.EvalSymlinks(path); err == nil {
		return p
	}
	return filepath.Clean(path)
}

// Worktrees lists the absolute paths of every worktree the repository has, the
// main checkout first.
func Worktrees(repo string) ([]string, error) {
	out, err := git(repo, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var paths []string
	for line := range strings.SplitSeq(out, "\n") {
		if p, ok := strings.CutPrefix(line, "worktree "); ok {
			paths = append(paths, p)
		}
	}
	return paths, nil
}

// HasBranch reports whether a local branch of that name exists.
func HasBranch(repo, branch string) bool {
	_, err := git(repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

// Fetch fetches a single refspec from origin.
func Fetch(repo, refspec string) error {
	_, err := git(repo, "fetch", "origin", refspec)
	return err
}

// AddWorktree checks an existing branch out into a new worktree.
func AddWorktree(repo, path, branch string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	_, err := git(repo, "worktree", "add", path, branch)
	return err
}

func git(dir string, args ...string) (string, error) {
	return run.Output(dir, "git", args...)
}
