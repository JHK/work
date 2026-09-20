package beads

import (
	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/run"
	"github.com/JHK/work-cli/internal/worktree"
)

// Claim marks the ticket a fresh worktree was made for as being worked.
type Claim struct {
	repo worktree.Repo
}

func NewClaim(repo worktree.Repo) Claim { return Claim{repo: repo} }

func (c Claim) Name() worktree.IntegrationName { return config.BeadsIntegration }

// OnCreated claims the ticket, and only where this integration's own resolver
// sourced the place: one sourced anywhere else is another tracker's.
func (c Claim) OnCreated(t worktree.Tree) error {
	if t.Source != config.BeadsIntegration {
		return nil
	}
	// bd's --claim assigns the bead to the current actor and moves it to in_progress.
	_, err := run.Output(string(c.repo), binary, "update", string(t.ID), "--claim")
	return err
}

// Open has nothing to open: a worktree never opens on the tracker.
func (Claim) Open(worktree.Tree) (worktree.Handoff, error) { return worktree.Handoff{}, nil }
