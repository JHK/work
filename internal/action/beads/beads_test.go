package beads

import (
	"slices"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// The name is the whole question: this system's own resolver is what sources a
// place to it, so a place sourced anywhere else is another tracker's and bd is
// left out of it.
func TestClaimGoesByTheSource(t *testing.T) {
	tests := []struct {
		source string
		want   []string // what bd was asked to run, if anything
	}{
		// Claiming names the ticket the worktree was made for, which is the id the
		// resolver put on the place rather than the worktree's directory.
		{Name, []string{"bd update one-two --claim"}},
		{"github", nil},
		{"", nil},
	}
	for _, tt := range tests {
		t.Run("sourced to "+tt.source, func(t *testing.T) {
			// A bd that records what it was asked and answers nothing.
			ran := testenv.Stubs(t, testenv.Stub{Name: "bd"})
			c := New(t.TempDir())

			tree := worktree.Tree{Place: worktree.Place{Source: tt.source, ID: "one-two", Name: "one-two"}, Path: "/wt"}
			if err := c.Run(tree); err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got := ran(); !slices.Equal(got, tt.want) {
				t.Errorf("a place sourced to %q asked bd %q; want %q", tt.source, got, tt.want)
			}
		})
	}
}
