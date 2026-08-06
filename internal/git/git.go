// Package git is the read-only-by-default adapter over the git worktree
// commands work needs.
package git

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/JHK/work-cli/internal/run"
)

// Root reports the main checkout of the repository containing dir, so that
// running work from inside a linked worktree still nests new worktrees under the
// main checkout rather than under the current one.
func Root(dir string) (string, error) {
	out, err := git(dir, "rev-parse", "--path-format=absolute", "--git-dir", "--git-common-dir", "--show-toplevel")
	if err != nil {
		return "", err
	}
	gitDir, rest, _ := strings.Cut(out, "\n")
	commonDir, toplevel, _ := strings.Cut(rest, "\n")
	commonDir = filepath.Clean(commonDir)

	// The two directories differ only inside a linked worktree. Everywhere else
	// the top level is right whatever layout the git directory has.
	if filepath.Clean(gitDir) == commonDir {
		return toplevel, nil
	}
	// Inside one, the main checkout is where the common git directory sits: the
	// rule git itself applies to report it as the first worktree of the list.
	if filepath.Base(commonDir) == ".git" {
		return filepath.Dir(commonDir), nil
	}
	return commonDir, nil
}

// SameDir reports whether two paths name the same directory.
func SameDir(a, b string) bool {
	return resolve(a) == resolve(b)
}

// resolve normalises a path for comparison; git reports worktrees with symlinks
// already resolved.
func resolve(path string) string {
	if p, err := filepath.EvalSymlinks(path); err == nil {
		return p
	}
	return filepath.Clean(path)
}

// Worktree is one checkout git has registered: where it sits, and what it has
// checked out there.
type Worktree struct {
	Path   string
	Branch string // short name; empty when the worktree is detached
}

// Linked lists every worktree but the main checkout, which git reports first.
func Linked(repo string) ([]Worktree, error) {
	list, err := worktrees(repo)
	if err != nil {
		return nil, err
	}
	if len(list) > 0 {
		list = list[1:]
	}
	return list, nil
}

// worktrees lists every worktree the repository has, the main checkout first.
func worktrees(repo string) ([]Worktree, error) {
	out, err := git(repo, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var list []Worktree
	for line := range strings.SplitSeq(out, "\n") {
		if p, ok := strings.CutPrefix(line, "worktree "); ok {
			list = append(list, Worktree{Path: p})
		} else if ref, ok := strings.CutPrefix(line, "branch "); ok && len(list) > 0 {
			list[len(list)-1].Branch = strings.TrimPrefix(ref, "refs/heads/")
		}
	}
	return list, nil
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
