// Package beads claims the ticket a worktree was made for. It is the far half of
// the system internal/resolve/beads is the near half of, over the one bd client
// both sit on: resolving a ticket names its worktree, and the worktree coming
// into being is what claiming it means.
package beads

import (
	"github.com/JHK/work-cli/internal/beads"
	"github.com/JHK/work-cli/internal/worktree"
)

// Name is what this system goes by on both seams, and what --no-claim turns off.
const Name = "beads"

// Claim marks the ticket a fresh worktree was made for as being worked.
type Claim struct {
	repo string
}

func New(repo string) Claim { return Claim{repo: repo} }

func (c Claim) Name() string { return Name }

// Flag is what declines this action on the command line. The worktree is still
// made and the ticket still vetted; bd is not told the ticket is being worked.
func (c Claim) Flag() (string, string) {
	return "no-claim", "create the worktree without claiming the ticket; the vetting still applies"
}

// Run claims the ticket. The name is the whole question: this system's own
// resolver is what sources a place to it, so a place sourced anywhere else is
// another tracker's and bd is left out of it. Only a worktree this run created
// reaches an action, so there is no re-entry to keep out of bd's history.
func (c Claim) Run(t worktree.Tree) error {
	if t.Source != Name {
		return nil
	}
	return beads.Claim(c.repo, t.ID)
}
