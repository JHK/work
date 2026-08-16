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

// Nothing else carries a refusal the action threw away, so it is said where the
// worktree went untrusted, naming what work reached for and what came back.
func TestRunSaysWhatItThrewAway(t *testing.T) {
	tests := []struct {
		name string
		mise []testenv.Stub // what the action finds on PATH, none for a machine without mise
		want string
	}{
		{"no mise at all", nil, "work: mise trust: mise is not on PATH\n"},
		{"a mise that refuses", []testenv.Stub{{Name: "mise", Grumbles: "config file is untrusted", Exits: 1}}, "work: mise trust: config file is untrusted\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mise == nil {
				t.Setenv("PATH", t.TempDir()) // a machine with nothing installed
			}
			testenv.Stubs(t, tt.mise...)
			said := testenv.Warnings(t)
			tree := worktree.Tree{Place: worktree.Place{Name: "scratch"}, Path: t.TempDir(), Created: true}

			if err := (Trust{}).Run(tree); err != nil {
				t.Fatalf("Run = %v; want the worktree left standing", err)
			}
			if said() != tt.want {
				t.Errorf("work said %q; want %q", said(), tt.want)
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
