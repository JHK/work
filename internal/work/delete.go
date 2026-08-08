package work

import (
	"fmt"
	"os"

	"github.com/JHK/work-cli/internal/git"
)

// Deletion is what was taken away: the worktree, and the branch it had checked
// out where it had one.
type Deletion struct {
	Path   string
	Branch string
}

// Delete removes a worktree and the branch it had checked out, git's two
// commands and nothing besides: no ticket is touched and no tracker asked, so a
// bead's worktree, a pull request's and a bare one all go the same way.
//
// Both of git's gates are weighed before anything is removed, so a refusal
// leaves the worktree standing rather than half a deletion: a branch left behind
// is what makes --create refuse the name later. A dirty worktree is git's own
// refusal, made by the removal before it touches anything; a branch not fully
// merged is one only git branch -d makes, by which time the worktree would be
// gone, so it is weighed here instead and the deletion itself left unguarded:
// -d weighs the branch against its upstream where it has one, which would refuse
// after the removal what the gate here had already allowed. force overrides both.
func (e Env) Delete(c Candidate, force bool) (Deletion, error) {
	if !c.Open {
		return Deletion{}, fmt.Errorf("%s has no worktree to remove", c.Target.Name)
	}
	// git removes the worktree the process stands in without a word, leaving the
	// shell in a directory that is gone.
	if wd, err := os.Getwd(); err == nil && git.Inside(wd, c.path) {
		return Deletion{}, fmt.Errorf("%s is the worktree you are standing in; run work remove from outside it", c.Target.Name)
	}
	if !force && c.branch != "" && !git.Merged(e.Repo, c.branch) {
		return Deletion{}, fmt.Errorf("branch %s is not fully merged; delete it anyway with --force", c.branch)
	}

	if err := git.RemoveWorktree(e.Repo, c.path, force); err != nil {
		return Deletion{}, err
	}
	if c.branch != "" {
		if err := git.DeleteBranch(e.Repo, c.branch); err != nil {
			return Deletion{}, fmt.Errorf("removed worktree %s, but its branch stayed: %w", c.path, err)
		}
	}
	return Deletion{Path: c.path, Branch: c.branch}, nil
}
