package work

import (
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/beads"
)

func TestVetBead(t *testing.T) {
	workable := beads.Bead{ID: "bd-1", Title: "A ticket", Status: "open", Type: "task", AcceptanceCriteria: "It works"}

	tests := []struct {
		name  string
		bead  beads.Bead
		ready map[string]bool
		want  string // a fragment of the reason, or "" for workable
	}{
		{"workable and ready", workable, map[string]bool{"bd-1": true}, ""},
		{"in progress needs no ready check", with(workable, func(b *beads.Bead) { b.Status = "in_progress" }), nil, ""},
		{"deferred", with(workable, func(b *beads.Bead) { b.Status = "deferred" }), nil, "unrefined"},
		{"closed", with(workable, func(b *beads.Bead) { b.Status = "closed" }), nil, "already closed"},
		{"epic", with(workable, func(b *beads.Bead) { b.Type = "epic" }), nil, "is an epic"},
		{"no acceptance criteria", with(workable, func(b *beads.Bead) { b.AcceptanceCriteria = "  \n" }), nil, "no acceptance criteria"},
		{"blocked status", with(workable, func(b *beads.Bead) { b.Status = "blocked" }), nil, "is blocked, not workable"},

		// /refine would move it out of the status that is the actual blocker.
		{"unworkable status outranks missing criteria", with(workable, func(b *beads.Bead) { b.Status, b.AcceptanceCriteria = "blocked", "" }), nil, "is blocked, not workable"},
		{"blocked by a dependency", workable, map[string]bool{"bd-2": true}, "open dependency"},

		// A deferred epic is unrefined first: the cheapest fix comes first.
		{"deferred outranks epic", with(workable, func(b *beads.Bead) { b.Status, b.Type = "deferred", "epic" }), nil, "unrefined"},
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

func with(b beads.Bead, f func(*beads.Bead)) beads.Bead {
	f(&b)
	return b
}
