// Package git is the read-only-by-default adapter over the git worktree
// commands work needs.
package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JHK/work-cli/internal/run"
)

// Root reports the main checkout of the repository containing dir, so a call
// from inside a linked worktree still resolves to the main checkout. A bare
// repository is its own, having no checkout to stand in.
func Root(dir string) (string, error) {
	out, err := git(dir, "rev-parse", "--path-format=absolute", "--git-dir", "--git-common-dir", "--is-bare-repository")
	if err != nil {
		// Only a directory with no repository is work's own; a git that cannot read
		// the one that is there keeps its own words, which name the way out.
		if !strings.Contains(err.Error(), "not a git repository") {
			return "", err
		}
		return "", errors.New("no git repository here")
	}
	gitDir, rest, _ := strings.Cut(out, "\n")
	commonDir, bare, _ := strings.Cut(rest, "\n")
	commonDir = filepath.Clean(commonDir)

	if bare == "true" {
		return commonDir, nil
	}
	// The two differ only inside a linked worktree, where the main checkout is where
	// the common git directory sits.
	if filepath.Clean(gitDir) != commonDir {
		if filepath.Base(commonDir) == ".git" {
			return filepath.Dir(commonDir), nil
		}
		return commonDir, nil
	}
	// Asked rather than inferred: a main checkout's git directory need not be the
	// .git beside it, and only git knows where its working tree is.
	return git(dir, "rev-parse", "--show-toplevel")
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
	Bare   bool   // no working tree, as a bare repository reports for itself
}

// Worktrees lists every worktree the repository has, in git's order.
func Worktrees(repo string) ([]Worktree, error) {
	out, err := git(repo, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var list []Worktree
	for line := range strings.SplitSeq(out, "\n") {
		if p, ok := strings.CutPrefix(line, "worktree "); ok {
			list = append(list, Worktree{Path: p})
			continue
		}
		if len(list) == 0 {
			continue
		}
		at := &list[len(list)-1]
		if ref, ok := strings.CutPrefix(line, "branch "); ok {
			at.Branch = strings.TrimPrefix(ref, "refs/heads/")
		} else if line == "bare" {
			at.Bare = true
		}
	}
	return list, nil
}

// HasBranch reports whether a local branch of that name exists.
func HasBranch(repo, branch string) bool {
	_, err := git(repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

// origin is the one remote work reads.
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

// Vacant refuses a path something already sits at, a worktree needing the whole
// directory to itself.
func Vacant(path string) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("%s is already there; take that directory away first", path)
	}
	return nil
}

// MoveWorktree moves a worktree's directory and registers it where it landed.
// git nests the worktree inside a destination already there rather than refusing
// it, so that refusal is this adapter's.
func MoveWorktree(repo, from, to string) error {
	if err := Vacant(to); err != nil {
		return err
	}
	if err := mkParent(to); err != nil {
		return err
	}
	_, err := git(repo, "worktree", "move", from, to)
	return err
}

// RenameBranch renames a local branch, the worktree that has it checked out
// following it. git refuses a name another branch already holds.
func RenameBranch(repo, from, to string) error {
	_, err := git(repo, "branch", "--move", from, to)
	return err
}

// Dirty reports whether a worktree carries modified or untracked files, asked as
// git's own removal asks it. One it cannot read counts as clean.
func Dirty(path string) bool {
	out, err := git(path, "status", "--porcelain", "--ignore-submodules=none")
	return err == nil && out != ""
}

// Stash saves the working state of the checkout at dir, untracked files in and
// ignored files left where they are, and says whether there was any to save.
func Stash(dir string) (bool, error) {
	before := stashed(dir)
	if _, err := git(dir, "stash", "push", "--include-untracked"); err != nil {
		return false, err
	}
	return stashed(dir) != before, nil
}

// Unstash pops the repository's topmost stash entry into the checkout at dir,
// staged staying staged. One that will not apply keeps it, dir part-written.
func Unstash(dir string) error {
	_, err := git(dir, "stash", "pop", "--index")
	return err
}

// stashed is what refs/stash points at, empty where there is no entry.
func stashed(dir string) string {
	out, err := git(dir, "rev-parse", "--verify", "--quiet", "refs/stash")
	if err != nil {
		return ""
	}
	return out
}

// RemoveWorktree unregisters a worktree and deletes its directory. git refuses
// one with modified or untracked files unless force, and the main checkout
// either way.
func RemoveWorktree(repo, path string, force bool) error {
	args := []string{"worktree", "remove", path}
	if force {
		args = append(args, "--force")
	}
	_, err := git(repo, args...)
	return err
}

// DeleteBranch deletes a local branch whether or not its work has landed. git
// refuses one a worktree still has checked out.
func DeleteBranch(repo, branch string) error {
	_, err := git(repo, "branch", "--delete", "--force", branch)
	return err
}

// add makes the directory the worktree goes in, then adds it. -q leaves the
// progress line off stderr, where a failure's own message is read from.
func add(dir, path string, args ...string) error {
	if err := mkParent(path); err != nil {
		return err
	}
	_, err := git(dir, append([]string{"worktree", "add", "-q", path}, args...)...)
	return err
}

// mkParent makes the directory a worktree is about to sit in.
func mkParent(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

// git asks in English, git translating its refusals and this adapter reading
// them apart.
func git(dir string, args ...string) (string, error) {
	return run.InEnglish(dir, "git", args...)
}
