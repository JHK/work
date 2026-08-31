// Package beads is the adapter over the bd issue tracker.
package beads

import "github.com/JHK/work-cli/internal/run"

// Binary is the tracker itself.
const Binary = "bd"

// Bead carries the fields work needs to name a worktree and judge whether the
// ticket can be worked. bd reports far more; the rest is the tracker's business.
type Bead struct {
	ID                 string `json:"id"`
	Title              string `json:"title"`
	Status             string `json:"status"`
	Type               string `json:"issue_type"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
}

// All lists every bead the tracker knows, closed ones included: a worktree
// outlives the status of the ticket it was opened for.
func All(repo string) ([]Bead, error) {
	return run.JSON[[]Bead](repo, Binary, "list", "--all", "--limit", "0", "--json")
}

// Ready lists every bead whose dependencies are satisfied.
func Ready(repo string) ([]Bead, error) {
	return run.JSON[[]Bead](repo, Binary, "ready", "--limit", "0", "--json")
}

// Claim assigns the bead to the current actor and moves it to in_progress.
func Claim(repo, id string) error {
	_, err := run.Output(repo, Binary, "update", id, "--claim")
	return err
}

// CreateWorktree adds a worktree wired to the repository's shared database. bd
// takes no fork point, so the branch forks from what the checkout at from has at HEAD.
func CreateWorktree(from, path, branch string) error {
	_, err := run.Output(from, Binary, "worktree", "create", path, "--branch", branch)
	return err
}
