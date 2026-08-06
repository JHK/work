package work

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/beads"
)

func TestVetBead(t *testing.T) {
	workable := beads.Bead{ID: "bd-1", Title: "A ticket", Status: "open", Type: "task", AcceptanceCriteria: "It works"}

	tests := []struct {
		name  string
		bead  beads.Bead
		ready bool
		want  string // a fragment of the reason, or "" for workable
	}{
		{"workable and ready", workable, true, ""},
		{"in progress needs no ready check", with(workable, func(b *beads.Bead) { b.Status = "in_progress" }), false, ""},
		{"deferred", with(workable, func(b *beads.Bead) { b.Status = "deferred" }), false, "unrefined"},
		{"closed", with(workable, func(b *beads.Bead) { b.Status = "closed" }), false, "already closed"},
		{"epic", with(workable, func(b *beads.Bead) { b.Type = "epic" }), false, "is an epic"},
		{"no acceptance criteria", with(workable, func(b *beads.Bead) { b.AcceptanceCriteria = "  \n" }), false, "no acceptance criteria"},
		{"blocked status", with(workable, func(b *beads.Bead) { b.Status = "blocked" }), false, "is blocked, not workable"},

		// /refine would move it out of the status that is the actual blocker.
		{"unworkable status outranks missing criteria", with(workable, func(b *beads.Bead) { b.Status, b.AcceptanceCriteria = "blocked", "" }), false, "is blocked, not workable"},
		{"blocked by a dependency", workable, false, "open dependency"},

		// A deferred epic is unrefined first: the cheapest fix comes first.
		{"deferred outranks epic", with(workable, func(b *beads.Bead) { b.Status, b.Type = "deferred", "epic" }), false, "unrefined"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := vetBead(tt.bead, tt.ready)
			switch {
			case tt.want == "" && got != "":
				t.Errorf("vetBead() = %q, want workable", got)
			case tt.want != "" && !strings.Contains(got, tt.want):
				t.Errorf("vetBead() = %q, want it to mention %q", got, tt.want)
			}
		})
	}
}

// A listing that already reported the bead ready spares vet the query. The
// repository here is not one, so a query still made fails rather than answers.
func TestVetTakesTheListingsWord(t *testing.T) {
	e := Env{Repo: filepath.Join(t.TempDir(), "not-a-repo")}
	open := beads.Bead{ID: "bd-1", Status: "open", Type: "task", AcceptanceCriteria: "It works"}

	if reason, err := e.vet(open, true); err != nil || reason != "" {
		t.Errorf("vet(vouched) = %q, %v; want it workable without asking bd", reason, err)
	}
	if _, err := e.vet(open, false); err == nil {
		t.Error("vet without a vouched bead did not ask bd")
	}
}

func with(b beads.Bead, f func(*beads.Bead)) beads.Bead {
	f(&b)
	return b
}
