package work

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/worktree"
)

// Move is what moved: where the worktree was and where it sits now, and what its
// branch was called and is called now.
type Move struct {
	From, To worktree.Path
	Was, Now worktree.Branch // both empty where the worktree is detached
}

// Renamed reports whether the branch took a name it did not already have.
func (m Move) Renamed() bool { return m.Was != m.Now }

// Movable is why a worktree cannot be moved at all, which is known from the
// candidate alone: a front end asks it ahead of the destination rather than
// after.
func (e Env) Movable(c Candidate) error { return e.actionable(c, "move") }

// Move moves a worktree's directory and renames its branch to the destination's
// last element. No ticket is touched, no tracker asked and no action run. Both
// halves land or neither does. A candidate [Env.Movable] refuses is refused here
// too, a front end asking ahead of the destination being what that is for.
func (e Env) Move(c Candidate, dest worktree.Path) (Move, error) {
	m, err := e.plan(c, dest)
	if err != nil {
		return Move{}, err
	}
	// The branch goes first: a name git will not take costs nothing here, where a
	// directory already moved would have to go back.
	if m.Renamed() {
		if err := git.RenameBranch(e.Repo, m.Was, m.Now); err != nil {
			return Move{}, err
		}
	}
	if err := git.MoveWorktree(e.Repo, c.path, m.To); err != nil {
		return Move{}, e.putTheBranchBack(m, err)
	}
	return m, nil
}

// putTheBranchBack undoes the rename the directory would not follow, and says so
// where the branch will not go back either.
func (e Env) putTheBranchBack(m Move, why error) error {
	if !m.Renamed() {
		return why
	}
	if back := git.RenameBranch(e.Repo, m.Now, m.Was); back != nil {
		return fmt.Errorf("renamed branch %s to %s, but %w, and it would not go back: %w", m.Was, m.Now, why, back)
	}
	return why
}

// plan is what the move would come to, and every refusal that costs nothing.
func (e Env) plan(c Candidate, dest worktree.Path) (Move, error) {
	if err := e.Movable(c); err != nil {
		return Move{}, err
	}
	to, err := e.destination(c.path, dest)
	if err != nil {
		return Move{}, err
	}
	// Ahead of the vacancy check, whose way out is to take the directory away.
	if git.SameDir(to, c.path) {
		return Move{}, fmt.Errorf("%s is where %s already sits", to, c.Name)
	}
	// Ahead of the rename, so a destination already there costs nothing rather than
	// leaving the branch renamed and the worktree where it was.
	if err := git.Vacant(to); err != nil {
		return Move{}, err
	}
	m := Move{From: c.path, To: to, Was: c.branch}
	if c.branch != "" {
		m.Now = worktree.Branch(filepath.Base(string(to)))
	}
	// Ahead of the move, so a name already taken costs nothing rather than leaving
	// the worktree moved and its branch behind.
	if m.Renamed() && git.HasBranch(e.Repo, m.Now) {
		return Move{}, fmt.Errorf("branch %s already exists", m.Now)
	}
	return m, nil
}

// destination is where a worktree lands: a bare name beside where it sits, and
// one carrying a separator a path of its own, read from [Env.Dir] where it is
// relative. Its last element is the name either way.
func (e Env) destination(from, dest worktree.Path) (worktree.Path, error) {
	to := string(dest)
	if err := checkName(worktree.Name(filepath.Base(to))); err != nil {
		return "", err
	}
	switch {
	case !strings.ContainsRune(to, filepath.Separator):
		return worktree.Path(filepath.Join(filepath.Dir(string(from)), to)), nil
	case filepath.IsAbs(to):
		return worktree.Path(filepath.Clean(to)), nil
	}
	return worktree.Path(filepath.Join(string(e.Dir), to)), nil
}
