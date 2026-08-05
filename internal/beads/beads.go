// Package beads is the adapter over the bd issue tracker.
package beads

import (
	"encoding/json"
	"fmt"
	"strings"

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

// Titles names each of the given beads in one query. Ids bd does not know are
// absent from the result rather than an error.
func Titles(repo string, ids []string) (map[string]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	beads, err := decode(bd(repo, "list", "--id", strings.Join(ids, ","), "--limit", "0", "--json"))
	if err != nil {
		return nil, err
	}
	titles := make(map[string]string, len(beads))
	for _, b := range beads {
		titles[b.ID] = b.Title
	}
	return titles, nil
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

// CreateWorktree adds a worktree wired to the repository's shared database.
func CreateWorktree(repo, path, branch string) error {
	_, err := bd(repo, "worktree", "create", path, "--branch", branch)
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

func bd(repo string, args ...string) (string, error) {
	return run.Output(repo, "bd", args...)
}
