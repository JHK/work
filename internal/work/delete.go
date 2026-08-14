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

// Delete removes a worktree and the branch it had checked out. No ticket is
// touched, no tracker asked and no action run. The worktree is git's to refuse,
// and where it is not refused the branch goes with it.
func (e Env) Delete(c Candidate, force bool) (Deletion, error) {
	if !c.Open {
		return Deletion{}, fmt.Errorf("%s has no worktree to remove", c.Name)
	}
	// git removes the worktree the process stands in without a word. Not the main
	// checkout, which every other worktree sits under and which git refuses outright.
	if wd, err := os.Getwd(); err == nil && !git.SameDir(c.path, e.Repo) && git.Inside(wd, c.path) {
		return Deletion{}, fmt.Errorf("%s is the worktree you are standing in; run work remove from outside it", c.Name)
	}
	if err := git.RemoveWorktree(e.Repo, c.path, force); err != nil {
		return Deletion{}, err
	}
	if c.branch != "" {
		if err := git.DeleteBranch(e.Repo, c.branch); err != nil {
			return Deletion{}, fmt.Errorf("removed worktree %s, but %w", c.path, err)
		}
	}
	return Deletion{Path: c.path, Branch: c.branch}, nil
}
