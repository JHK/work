package mise

import (
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// Granting the trust is best effort: a missing mise, or a grant that fails, only
// means the session prompts, so nothing here refuses the worktree that was just
// made.
func TestRunNeverRefusesTheWorktree(t *testing.T) {
	tests := []struct {
		name string
		mise []testenv.Stub // what the action finds on PATH, if anything
	}{
		{"no mise at all", nil},
		{"a mise that fails", []testenv.Stub{{Name: "mise", Exits: 3}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir()) // a machine with nothing installed
			testenv.Stubs(t, tt.mise...)
			tree := worktree.Tree{Place: worktree.Place{Name: "scratch"}, Path: t.TempDir(), Created: true}
			if err := (Trust{}).Run(tree); err != nil {
				t.Errorf("Run = %v; want the worktree left standing", err)
			}
		})
	}
}

// The action goes by the name the wiring and an [action] key spell it as.
func TestName(t *testing.T) {
	if got := (Trust{}).Name(); got != "mise" {
		t.Errorf("Name() = %q; want %q", got, "mise")
	}
}
