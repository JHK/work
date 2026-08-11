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
// Both refusals are git's own. git judges a branch only once no worktree holds
// it, so a branch it will not delete is refused with the worktree already gone —
// and the branch left standing is what makes Create refuse the name later.
func (e Env) Delete(c Candidate, force bool) (Deletion, error) {
	if !c.Open {
		return Deletion{}, fmt.Errorf("%s has no worktree to remove", c.Target.Name)
	}
	// git removes the worktree the process stands in without a word, leaving the
	// shell in a directory that is gone.
	if wd, err := os.Getwd(); err == nil && git.Inside(wd, c.path) {
		return Deletion{}, fmt.Errorf("%s is the worktree you are standing in; run work remove from outside it", c.Target.Name)
	}
	if err := git.RemoveWorktree(e.Repo, c.path, force); err != nil {
		return Deletion{}, err
	}
	if c.branch != "" {
		if err := git.DeleteBranch(e.Repo, c.branch, force); err != nil {
			// Naming a flag already given would be no help.
			hint := "; delete it anyway with --force"
			if force {
				hint = ""
			}
			return Deletion{}, fmt.Errorf("removed worktree %s, but its branch stayed: %w%s", c.path, err, hint)
		}
	}
	return Deletion{Path: c.path, Branch: c.branch}, nil
}
