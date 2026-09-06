// Package beads is the adapter over the bd issue tracker.
package beads

import (
	"github.com/JHK/work-cli/internal/run"
	"github.com/JHK/work-cli/internal/worktree"
)

// Binary is the tracker itself.
const Binary = "bd"

// Bead carries the fields work needs to name a worktree and judge whether the
// ticket can be worked. bd reports far more; the rest is the tracker's business.
type Bead struct {
	ID                 worktree.ID    `json:"id"`
	Title              worktree.Label `json:"title"`
	Status             string         `json:"status"`
	Type               string         `json:"issue_type"`
	AcceptanceCriteria string         `json:"acceptance_criteria"`
}

// All lists every bead the tracker knows, closed ones included: a worktree
// outlives the status of the ticket it was opened for.
func All(repo worktree.Repo) ([]Bead, error) {
	return run.JSON[[]Bead](string(repo), Binary, "list", "--all", "--limit", "0", "--json")
}

// Ready lists every bead whose dependencies are satisfied.
func Ready(repo worktree.Repo) ([]Bead, error) {
	return run.JSON[[]Bead](string(repo), Binary, "ready", "--limit", "0", "--json")
}

// Claim assigns the bead to the current actor and moves it to in_progress.
func Claim(repo worktree.Repo, id worktree.ID) error {
	_, err := run.Output(string(repo), Binary, "update", string(id), "--claim")
	return err
}

// CreateWorktree adds a worktree wired to the repository's shared database. bd
// takes no fork point, so the branch forks from what the checkout at from has at HEAD.
func CreateWorktree(from, path worktree.Path, branch worktree.Branch) error {
	_, err := run.Output(string(from), Binary, "worktree", "create", string(path), "--branch", string(branch))
	return err
}
