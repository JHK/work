// Package git is the read-only-by-default adapter over the git worktree
// commands work needs.
package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/JHK/work-cli/internal/run"
	"github.com/JHK/work-cli/internal/worktree"
)

// Root reports the main checkout of the repository containing dir, so a call
// from inside a linked worktree still resolves to the main checkout. A bare
// repository is its own, having no checkout to stand in.
func Root(dir worktree.Path) (worktree.Repo, error) {
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
		return worktree.Repo(commonDir), nil
	}
	// The two differ only inside a linked worktree, where the main checkout is where
	// the common git directory sits.
	if filepath.Clean(gitDir) != commonDir {
		if filepath.Base(commonDir) == ".git" {
			return worktree.Repo(filepath.Dir(commonDir)), nil
		}
		return worktree.Repo(commonDir), nil
	}
	// Asked rather than inferred: a main checkout's git directory need not be the
	// .git beside it, and only git knows where its working tree is.
	top, err := git(dir, "rev-parse", "--show-toplevel")
	return worktree.Repo(top), err
}

// SameDir reports whether two paths name the same directory.
func SameDir(a, b worktree.Path) bool {
	return realPath(a) == realPath(b)
}

// Inside reports whether path is dir or sits below it.
func Inside(path, dir worktree.Path) bool {
	rel, err := filepath.Rel(realPath(dir), realPath(path))
	return err == nil && filepath.IsLocal(rel)
}

// git reports worktrees with symlinks already resolved, so a path is compared by
// the real one.
func realPath(path worktree.Path) string {
	if p, err := filepath.EvalSymlinks(string(path)); err == nil {
		return p
	}
	return filepath.Clean(string(path))
}

// Worktree is one checkout git has registered: where it sits, and what it has
// checked out there.
type Worktree struct {
	Path   worktree.Path
	Branch worktree.Branch // short name; empty when the worktree is detached
	Bare   bool            // no working tree, as a bare repository reports for itself
}

// Worktrees lists every worktree the repository has, in git's order.
func Worktrees(repo worktree.Repo) ([]Worktree, error) {
	out, err := inRepo(repo, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var list []Worktree
	for line := range strings.SplitSeq(out, "\n") {
		if p, ok := strings.CutPrefix(line, "worktree "); ok {
			list = append(list, Worktree{Path: worktree.Path(p)})
			continue
		}
		if len(list) == 0 {
			continue
		}
		at := &list[len(list)-1]
		if ref, ok := strings.CutPrefix(line, "branch "); ok {
			at.Branch = worktree.Branch(strings.TrimPrefix(ref, "refs/heads/"))
		} else if line == "bare" {
			at.Bare = true
		}
	}
	return list, nil
}

// HasBranch reports whether a local branch of that name exists.
func HasBranch(repo worktree.Repo, branch worktree.Branch) bool {
	_, err := inRepo(repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+string(branch))
	return err == nil
}

// origin is the one remote work reads.
const origin = "origin"

// OriginURL reports where origin points, or "" where the repository has no such
// remote.
func OriginURL(repo worktree.Repo) string {
	url, err := inRepo(repo, "remote", "get-url", origin)
	if err != nil {
		return ""
	}
	return url
}

// Fetch fetches a single refspec from origin.
func Fetch(repo worktree.Repo, refspec string) error {
	_, err := inRepo(repo, "fetch", origin, refspec)
	return err
}

// AddWorktree checks an existing branch out into a new worktree.
func AddWorktree(repo worktree.Repo, path worktree.Path, branch worktree.Branch) error {
	return add(worktree.Path(repo), path, string(branch))
}

// NewWorktree checks a branch of its own out into a new worktree, forked from
// what the checkout at from has at HEAD. git refuses a branch that already
// exists, which is what asserts the name is free.
func NewWorktree(from, path worktree.Path, branch worktree.Branch) error {
	return add(from, path, "-b", string(branch))
}

// Vacant refuses a path something already sits at, a worktree needing the whole
// directory to itself.
func Vacant(path worktree.Path) error {
	if _, err := os.Lstat(string(path)); err == nil {
		return fmt.Errorf("%s is already there; take that directory away first", path)
	}
	return nil
}

// MoveWorktree moves a worktree's directory and registers it where it landed.
// git nests the worktree inside a destination already there rather than refusing
// it, so that refusal is this adapter's.
func MoveWorktree(repo worktree.Repo, from, to worktree.Path) error {
	if err := Vacant(to); err != nil {
		return err
	}
	if err := mkParent(to); err != nil {
		return err
	}
	_, err := inRepo(repo, "worktree", "move", string(from), string(to))
	return err
}

// RenameBranch renames a local branch, the worktree that has it checked out
// following it. git refuses a name another branch already holds.
func RenameBranch(repo worktree.Repo, from, to worktree.Branch) error {
	_, err := inRepo(repo, "branch", "--move", string(from), string(to))
	return err
}

// Dirty reports whether a worktree carries modified or untracked files, asked as
// git's own removal asks it. One it cannot read counts as clean.
func Dirty(path worktree.Path) bool {
	out, err := git(path, "status", "--porcelain", "--ignore-submodules=none")
	return err == nil && out != ""
}

// Stash saves the working state of the checkout at dir, untracked files in and
// ignored files left where they are, and says whether there was any to save.
func Stash(dir worktree.Path) (bool, error) {
	before := stashed(dir)
	if _, err := git(dir, "stash", "push", "--include-untracked"); err != nil {
		return false, err
	}
	return stashed(dir) > before, nil
}

// Unstash pops the repository's topmost stash entry into the checkout at dir,
// staged staying staged. One that will not apply keeps it, dir part-written.
func Unstash(dir worktree.Path) error {
	_, err := git(dir, "stash", "pop", "--index")
	return err
}

// stashed is how many entries the repository's stash holds, none where there is
// no stash. Counted rather than read off refs/stash, which two saves of the same
// state in the one second leave pointing at the one commit.
func stashed(dir worktree.Path) int {
	out, err := git(dir, "rev-list", "--walk-reflogs", "--count", "refs/stash")
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0
	}
	return n
}

// RemoveWorktree unregisters a worktree and deletes its directory. git refuses
// one with modified or untracked files unless force, and the main checkout
// either way.
func RemoveWorktree(repo worktree.Repo, path worktree.Path, force bool) error {
	args := []string{"worktree", "remove", string(path)}
	if force {
		args = append(args, "--force")
	}
	_, err := inRepo(repo, args...)
	return err
}

// DeleteBranch deletes a local branch whether or not its work has landed. git
// refuses one a worktree still has checked out.
func DeleteBranch(repo worktree.Repo, branch worktree.Branch) error {
	_, err := inRepo(repo, "branch", "--delete", "--force", string(branch))
	return err
}

// add makes the directory the worktree goes in, then adds it. -q leaves the
// progress line off stderr, where a failure's own message is read from.
func add(dir, path worktree.Path, args ...string) error {
	if err := mkParent(path); err != nil {
		return err
	}
	_, err := git(dir, append([]string{"worktree", "add", "-q", string(path)}, args...)...)
	return err
}

func mkParent(path worktree.Path) error {
	return os.MkdirAll(filepath.Dir(string(path)), 0o755)
}

// git asks in English, git translating its refusals and this adapter reading
// them apart.
func git(dir worktree.Path, args ...string) (string, error) {
	return run.InEnglish(string(dir), "git", args...)
}

// inRepo runs git in the repository's main checkout.
func inRepo(repo worktree.Repo, args ...string) (string, error) {
	return git(worktree.Path(repo), args...)
}
