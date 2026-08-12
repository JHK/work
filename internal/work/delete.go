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
// touched, no tracker asked and no action run. The two are meant to go together
// or not at all; where only one half lands, the error names what stayed.
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

// take is the forced pair: git refuses neither half, and this order is the only
// one open to a worktree git will not detach.
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

// weigh is the unforced pair: either half can be refused, so both of git's gates
// are weighed before anything is removed.
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
	// Little is left to refuse the removal now, though a locked worktree still can.
	if err := git.RemoveWorktree(e.Repo, c.path, false); err != nil {
		if !branched {
			return err
		}
		return fmt.Errorf("deleted branch %s, but its worktree stayed: %w", c.branch, err)
	}
	return nil
}
