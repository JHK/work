package work

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JHK/work-cli/internal/git"
)

// Move is what moved: where the worktree was and where it sits now, and what its
// branch was called and is called now.
type Move struct {
	From, To string
	Was, Now string // both empty where the worktree is detached
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
func (e Env) Move(c Candidate, dest string) (Move, error) {
	m, err := e.plan(c, dest)
	if err != nil {
		return Move{}, err
	}
	if err := git.MoveWorktree(e.Repo, c.path, m.To); err != nil {
		return Move{}, err
	}
	if !m.Renamed() {
		return m, nil
	}
	if err := git.RenameBranch(e.Repo, m.Was, m.Now); err != nil {
		if back := git.MoveWorktree(e.Repo, m.To, c.path); back != nil {
			return Move{}, fmt.Errorf("moved worktree to %s, but %w, and it would not move back: %w", m.To, err, back)
		}
		return Move{}, err
	}
	return m, nil
}

// plan is what the move would come to, and every refusal that costs nothing.
func (e Env) plan(c Candidate, dest string) (Move, error) {
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
	m := Move{From: c.path, To: to, Was: c.branch}
	if c.branch != "" {
		m.Now = filepath.Base(to)
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
func (e Env) destination(from, dest string) (string, error) {
	if err := checkName(filepath.Base(dest)); err != nil {
		return "", err
	}
	switch {
	case !strings.ContainsRune(dest, filepath.Separator):
		return filepath.Join(filepath.Dir(from), dest), nil
	case filepath.IsAbs(dest):
		return filepath.Clean(dest), nil
	}
	return filepath.Join(e.Dir, dest), nil
}
