// Package beads is the bd issue tracker: it turns the tickets bd knows into
// places to work and claims the ticket a worktree was made for.
package beads

import (
	"github.com/JHK/work-cli/internal/run"
	"github.com/JHK/work-cli/internal/worktree"
)

// binary is the CLI this integration goes through.
const binary = "bd"

// bead carries the fields work needs to name a worktree and judge whether the
// ticket can be worked. bd reports far more; the rest is the tracker's business.
type bead struct {
	ID                 worktree.ID    `json:"id"`
	Title              worktree.Label `json:"title"`
	Status             string         `json:"status"`
	Type               string         `json:"issue_type"`
	AcceptanceCriteria string         `json:"acceptance_criteria"`
}

// all lists every bead the tracker knows, closed ones included: a worktree
// outlives the status of the ticket it was opened for.
func all(repo worktree.Repo) ([]bead, error) {
	return run.JSON[[]bead](string(repo), binary, "list", "--all", "--limit", "0", "--json")
}

// ready lists every bead whose dependencies are satisfied.
func ready(repo worktree.Repo) ([]bead, error) {
	return run.JSON[[]bead](string(repo), binary, "ready", "--limit", "0", "--json")
}
