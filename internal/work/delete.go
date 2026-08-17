package work

import (
	"fmt"

	"github.com/JHK/work-cli/internal/git"
)

// Deletion is what was taken away: the worktree, and the branch it had checked
// out where it had one.
type Deletion struct {
	Path   string
	Branch string
}

// Delete removes a worktree and the branch it had checked out. No ticket is
// touched, no tracker asked and no action run. A worktree carrying changes is
// refused without force, and where the removal is not refused the branch goes
// with it.
func (e Env) Delete(c Candidate, force bool) (Deletion, error) {
	if err := e.actOn(c, "remove"); err != nil {
		return Deletion{}, err
	}
	if !force && git.Dirty(c.path) {
		return Deletion{}, fmt.Errorf("%s carries changes; take it and them with work remove --force %s", c.Name, c.Name)
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
