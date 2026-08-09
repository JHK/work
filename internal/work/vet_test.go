package work

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/beads"
)

var (
	workable = beads.Bead{ID: "bd-1", Title: "A ticket", Status: "open", Type: "task", AcceptanceCriteria: "It works"}
	epic     = with(workable, func(b *beads.Bead) { b.Type = "epic" })
)

var vettings = []struct {
	name  string
	bead  beads.Bead
	ready bool
	want  string // a fragment of the reason, or "" for workable
}{
	{"workable and ready", workable, true, ""},
	{"in progress needs no ready check", with(workable, func(b *beads.Bead) { b.Status = "in_progress" }), false, ""},
	{"deferred", with(workable, func(b *beads.Bead) { b.Status = "deferred" }), false, "unrefined"},
	{"closed", with(workable, func(b *beads.Bead) { b.Status = "closed" }), false, "already closed"},
	{"no acceptance criteria", with(workable, func(b *beads.Bead) { b.AcceptanceCriteria = "  \n" }), false, "no acceptance criteria"},
	{"blocked status", with(workable, func(b *beads.Bead) { b.Status = "blocked" }), false, "is blocked, not workable"},

	// /refine would move it out of the status that is the actual blocker.
	{"unworkable status outranks missing criteria", with(workable, func(b *beads.Bead) { b.Status, b.AcceptanceCriteria = "blocked", "" }), false, "is blocked, not workable"},
	{"blocked by a dependency", workable, false, "open dependency"},

	{"open epic, dependencies unasked", epic, false, ""},
	{"deferred epic", with(epic, func(b *beads.Bead) { b.Status = "deferred" }), false, ""},
	{"epic without acceptance criteria", with(epic, func(b *beads.Bead) { b.AcceptanceCriteria = "" }), false, ""},
	{"closed epic", with(epic, func(b *beads.Bead) { b.Status = "closed" }), false, "already closed"},
}

func TestVetBead(t *testing.T) {
	for _, tt := range vettings {
		t.Run(tt.name, func(t *testing.T) {
			got, err := vetBead(tt.bead, func() (bool, error) { return tt.ready, nil })
			switch {
			case err != nil:
				t.Fatalf("vetBead() errored: %v", err)
			case tt.want == "" && got != "":
				t.Errorf("vetBead() = %q, want workable", got)
			case tt.want != "" && !strings.Contains(got, tt.want):
				t.Errorf("vetBead() = %q, want it to mention %q", got, tt.want)
			}
		})
	}
}

// The economy of the query, stated as the property rather than as a list of the
// rules that hold it: a verdict both answers agree on cannot have turned on the
// question, so vetBead must reach it without asking. Any rule added ahead of the
// dependency check answers to this without being named here.
func TestVetBeadAsksOnlyWhereTheAnswerDecides(t *testing.T) {
	for _, tt := range vettings {
		t.Run(tt.name, func(t *testing.T) {
			var asked int
			vet := func(ready bool) string {
				got, err := vetBead(tt.bead, func() (bool, error) { asked++; return ready, nil })
				if err != nil {
					t.Fatalf("vetBead() errored: %v", err)
				}
				return got
			}
			if yes, no := vet(true), vet(false); yes == no && asked != 0 {
				t.Errorf("vetBead() asked whether the bead is ready %d times for a verdict the answer cannot change", asked)
			}
		})
	}
}

func TestVetBeadReportsAFailedQuery(t *testing.T) {
	boom := errors.New("bd is unreachable")
	if _, err := vetBead(workable, func() (bool, error) { return false, boom }); !errors.Is(err, boom) {
		t.Errorf("vetBead() error = %v, want it to report %v", err, boom)
	}
}

// A listing that already reported the bead ready spares the query. The
// repository here is not one, so a query still made fails rather than answers.
func TestReadinessTakesTheListingsWord(t *testing.T) {
	e := Env{Repo: filepath.Join(t.TempDir(), "not-a-repo")}
	s := State{Bead: workable, Ready: true}

	if ok, err := e.readiness(s)(); err != nil || !ok {
		t.Errorf("readiness(vouched) = %v, %v; want it ready without asking bd", ok, err)
	}
	s.Ready = false
	if _, err := e.readiness(s)(); err == nil {
		t.Error("readiness without a vouched bead did not ask bd")
	}
}

func with(b beads.Bead, f func(*beads.Bead)) beads.Bead {
	f(&b)
	return b
}
