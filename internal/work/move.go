package work

import (
	"errors"
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

// Move moves a worktree's directory and renames its branch to the destination's
// last element. No ticket is touched, no tracker asked and no action run. Both
// halves land or neither does.
func (e Env) Move(c Candidate, dest string) (Move, error) {
	if err := e.actOn(c, "move"); err != nil {
		return Move{}, err
	}
	to, err := destination(c.path, dest)
	if err != nil {
		return Move{}, err
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

	if err := git.MoveWorktree(e.Repo, c.path, to); err != nil {
		return Move{}, err
	}
	if !m.Renamed() {
		return m, nil
	}
	if err := git.RenameBranch(e.Repo, m.Was, m.Now); err != nil {
		// Neither half lands: the worktree goes back where it stood.
		if back := git.MoveWorktree(e.Repo, to, c.path); back != nil {
			return Move{}, fmt.Errorf("moved worktree to %s, but %w, and it would not move back: %w", to, err, back)
		}
		return Move{}, err
	}
	return m, nil
}

// destination is where a worktree lands: a bare name beside where it sits, and
// one carrying a separator a path of its own, read from the directory work was
// invoked in where it is relative. Its last element is the name either way.
func destination(from, dest string) (string, error) {
	if dest == "" {
		return "", errors.New("no destination given")
	}
	if err := checkName(filepath.Base(dest)); err != nil {
		return "", err
	}
	if !strings.ContainsRune(dest, filepath.Separator) {
		return filepath.Join(filepath.Dir(from), dest), nil
	}
	return filepath.Abs(dest)
}
