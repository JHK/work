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

// Delete removes a worktree and the branch it had checked out, git's own commands
// and nothing besides: no ticket is touched, no tracker asked and no action run,
// so a bead's worktree, a pull request's and a bare one all go the same way.
//
// The two go together or neither goes, so no refusal leaves a branch behind that
// git alone could clear and that makes Create refuse the name later.
func (e Env) Delete(c Candidate, force bool) (Deletion, error) {
	if !c.Open {
		return Deletion{}, fmt.Errorf("%s has no worktree to remove", c.Name)
	}
	// git removes the worktree the process stands in without a word, leaving the
	// shell in a directory that is gone.
	if wd, err := os.Getwd(); err == nil && git.Inside(wd, c.path) {
		return Deletion{}, fmt.Errorf("%s is the worktree you are standing in; run work remove from outside it", c.Name)
	}
	var err error
	if force {
		err = e.take(c)
	} else {
		err = e.weigh(c)
	}
	if err != nil {
		return Deletion{}, err
	}
	return Deletion{Path: c.path, Branch: c.branch}, nil
}

// take is the forced pair: git refuses neither half, so the worktree goes and the
// branch follows it. That order is also the only one open to a worktree git will
// not detach, one whose index is unresolved among them, which is what --force is
// reached for in the first place.
func (e Env) take(c Candidate) error {
	if err := git.RemoveWorktree(e.Repo, c.path, true); err != nil {
		return err
	}
	if c.branch == "" {
		return nil
	}
	if err := git.DeleteBranch(e.Repo, c.branch, true); err != nil {
		return fmt.Errorf("removed worktree %s, but branch %s stayed: %w", c.path, c.branch, err)
	}
	return nil
}

// weigh is the unforced pair, where either half can be refused and neither may go
// without the other, so both of git's gates are weighed before anything is
// removed: the status git weighs a worktree by, and the deletion git judges a
// branch by, which is asked of a branch whose worktree is still standing.
func (e Env) weigh(c Candidate) error {
	dirty, err := git.Dirty(c.path)
	if err != nil {
		return err
	}
	if dirty {
		return fmt.Errorf("%s contains modified or untracked files; use --force to remove it and its branch", c.Name)
	}
	branched := c.branch != ""
	if branched {
		if err := git.DeleteCheckedOutBranch(e.Repo, c.path, c.branch); err != nil {
			return fmt.Errorf("%w; %s stayed with it, and --force takes both", err, c.Name)
		}
	}
	// git found no modified or untracked files and the branch is already gone, so
	// little is left to refuse the removal; a locked worktree still can. What that
	// leaves is a worktree work remove still reaches, and never a branch nothing can
	// clear.
	if err := git.RemoveWorktree(e.Repo, c.path, false); err != nil {
		if !branched {
			return err
		}
		return fmt.Errorf("deleted branch %s, but its worktree stayed: %w", c.branch, err)
	}
	return nil
}
