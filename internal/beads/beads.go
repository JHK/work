// Package beads is the adapter over the bd issue tracker.
package beads

import (
	"encoding/json"
	"fmt"

	"github.com/JHK/work-cli/internal/run"
)

// Bead carries the fields work needs to name a worktree and judge whether the
// ticket can be worked. bd reports far more; the rest is the tracker's business.
type Bead struct {
	ID                 string `json:"id"`
	Title              string `json:"title"`
	Status             string `json:"status"`
	Type               string `json:"issue_type"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
}

// Show looks up one bead. A missing bead and an unreachable database are both
// errors here; callers decide which paths may proceed without one.
func Show(repo, id string) (Bead, error) {
	beads, err := decode(bd(repo, "show", id, "--json"))
	if err != nil {
		return Bead{}, err
	}
	if len(beads) == 0 {
		return Bead{}, fmt.Errorf("no bead %q", id)
	}
	return beads[0], nil
}

// All lists every bead the tracker knows, closed ones included: a worktree
// outlives the status of the ticket it was opened for.
func All(repo string) ([]Bead, error) {
	return decode(bd(repo, "list", "--all", "--limit", "0", "--json"))
}

// Ready lists every bead whose dependencies are satisfied.
func Ready(repo string) ([]Bead, error) {
	return decode(bd(repo, "ready", "--limit", "0", "--json"))
}

// Claim assigns the bead to the current actor and moves it to in_progress.
func Claim(repo, id string) error {
	_, err := bd(repo, "update", id, "--claim")
	return err
}

// CreateWorktree adds a worktree wired to the repository's shared database,
// forked from what the checkout at from has at HEAD: bd takes no fork point of
// its own, so the git worktree add underneath reads HEAD where bd stands.
func CreateWorktree(from, path, branch string) error {
	_, err := bd(from, "worktree", "create", path, "--branch", branch)
	return err
}

func decode(out string, err error) ([]Bead, error) {
	if err != nil {
		return nil, err
	}
	var beads []Bead
	if err := json.Unmarshal([]byte(out), &beads); err != nil {
		return nil, fmt.Errorf("bd: %w", err)
	}
	return beads, nil
}

func bd(dir string, args ...string) (string, error) {
	return run.Output(dir, "bd", args...)
}
