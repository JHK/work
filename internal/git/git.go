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

// Base reports the commit the worktree at dir forked from: the merge-base of
// what it has checked out and what the main checkout has. What has landed on the
// main checkout since is on the far side of it, so a diff against it is the
// worktree's own work and nothing else.
func Base(dir string) (string, error) {
	// git names the main checkout's head from any worktree of the repository,
	// whether or not that checkout is on a branch.
	return git(dir, "merge-base", "HEAD", "main-worktree/HEAD")
}

// SameDir reports whether two paths name the same directory.
func SameDir(a, b string) bool {
	return resolve(a) == resolve(b)
}

// Inside reports whether path is dir or sits below it.
func Inside(path, dir string) bool {
	rel, err := filepath.Rel(resolve(dir), resolve(path))
	return err == nil && filepath.IsLocal(rel)
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

// Merged reports whether a branch has landed on the main checkout's HEAD. git
// branch -d asks a different question, weighing the branch against its upstream
// wherever it has one, so this gate is work's own and not git's.
func Merged(repo, branch string) bool {
	_, err := git(repo, "merge-base", "--is-ancestor", "refs/heads/"+branch, "HEAD")
	return err == nil
}

// origin is the one remote work reads: a pull request is fetched from it, so it
// is the repository whose pull requests are worth offering.
const origin = "origin"

// OriginURL reports where origin points, or "" where the repository has no such
// remote.
func OriginURL(repo string) string {
	url, err := git(repo, "remote", "get-url", origin)
	if err != nil {
		return ""
	}
	return url
}

// Fetch fetches a single refspec from origin.
func Fetch(repo, refspec string) error {
	_, err := git(repo, "fetch", origin, refspec)
	return err
}

// AddWorktree checks an existing branch out into a new worktree.
func AddWorktree(repo, path, branch string) error {
	return add(repo, path, branch)
}

// NewWorktree checks a branch of its own out into a new worktree, forked from
// what the main checkout has at HEAD. git refuses a branch that already exists,
// which is what asserts the name is free.
func NewWorktree(repo, path, branch string) error {
	return add(repo, path, "-b", branch)
}

// RemoveWorktree unregisters a worktree and deletes its directory. git refuses
// one with modified or untracked files unless force.
func RemoveWorktree(repo, path string, force bool) error {
	args := []string{"worktree", "remove", path}
	if force {
		args = append(args, "--force")
	}
	_, err := git(repo, args...)
	return err
}

// DeleteBranch deletes a local branch, whether or not it is merged: the caller
// weighs that with [Merged], -d weighing something else. git refuses while a
// worktree still has the branch checked out.
func DeleteBranch(repo, branch string) error {
	_, err := git(repo, "branch", "-D", branch)
	return err
}

// add makes the directory the worktree goes in, then adds it. git takes its
// options after the path as readily as before it.
func add(repo, path string, args ...string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	_, err := git(repo, append([]string{"worktree", "add", path}, args...)...)
	return err
}

func git(dir string, args ...string) (string, error) {
	return run.Output(dir, "git", args...)
}
