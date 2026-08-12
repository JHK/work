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
// what the checkout at from has at HEAD. git refuses a branch that already
// exists, which is what asserts the name is free.
func NewWorktree(from, path, branch string) error {
	return add(from, path, "-b", branch)
}

// RemoveWorktree unregisters a worktree and deletes its directory. git refuses
// one with modified or untracked files unless force.
func RemoveWorktree(repo, path string, force bool) error {
	return forced(repo, force, "worktree", "remove", path)
}

// DeleteBranch deletes a local branch. Without force git refuses one whose work
// has not landed, which it judges only once no worktree holds the branch;
// nothing here asks the question first.
func DeleteBranch(repo, branch string, force bool) error {
	return forced(repo, force, "branch", "--delete", branch)
}

// DeleteCheckedOutBranch deletes a branch a worktree still has checked out, which
// git judges only once no worktree holds it: the worktree is detached for the
// question to be asked at all, and put back on the branch where git refuses, so
// its refusal leaves the worktree standing where it stood. It is unforced by
// nature — forced, git judges nothing and the worktree can simply go first.
func DeleteCheckedOutBranch(repo, worktree, branch string) error {
	if err := detach(worktree); err != nil {
		return err
	}
	if err := DeleteBranch(repo, branch, false); err != nil {
		if back := checkout(worktree, branch); back != nil {
			return fmt.Errorf("%w; %s is left detached: %w", err, worktree, back)
		}
		return err
	}
	return nil
}

// detach points a worktree at the commit it is on rather than at the branch. HEAD
// does not move, so the files under it are left exactly as they are, modified and
// untracked ones included. A worktree whose index is unresolved is refused.
func detach(worktree string) error {
	_, err := git(worktree, "checkout", "--detach")
	return err
}

// checkout puts a worktree back on a branch.
func checkout(worktree, branch string) error {
	_, err := git(worktree, "checkout", branch)
	return err
}

// Dirty reports whether a worktree holds modified or untracked files, which is
// the status git's own worktree removal weighs it by. Ignored files are no more
// dirty here than they are there.
func Dirty(worktree string) (bool, error) {
	out, err := git(worktree, "status", "--porcelain", "--ignore-submodules=none")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

// forced runs a git command that takes --force, which git accepts after the
// positional as readily as before it.
func forced(dir string, force bool, args ...string) error {
	if force {
		args = append(args, "--force")
	}
	_, err := git(dir, args...)
	return err
}

// add makes the directory the worktree goes in, then adds it. -q leaves the
// progress line off stderr, where a failure's own message is read from. git
// takes its options after the path as readily as before it.
func add(dir, path string, args ...string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	_, err := git(dir, append([]string{"worktree", "add", "-q", path}, args...)...)
	return err
}

func git(dir string, args ...string) (string, error) {
	return run.Output(dir, "git", args...)
}
