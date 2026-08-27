// Package beads claims the ticket a worktree was made for.
package beads

import (
	"github.com/JHK/work-cli/internal/beads"
	"github.com/JHK/work-cli/internal/worktree"
)

// Name is what this system goes by on both seams.
const Name = "beads"

// Claim marks the ticket a fresh worktree was made for as being worked.
type Claim struct {
	repo string
}

func New(repo string) Claim { return Claim{repo: repo} }

func (c Claim) Name() string { return Name }

// Run claims the ticket, and only where this system's own resolver sourced the
// place: one sourced anywhere else is another tracker's.
func (c Claim) Run(t worktree.Tree) error {
	if t.Source != Name {
		return nil
	}
	return beads.Claim(c.repo, t.ID)
}
