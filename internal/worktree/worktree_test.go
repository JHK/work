package worktree

import (
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// No command reaches Merge. The first to set a name owns it, which keeps the
// core's account of the worktree ahead of the resolver's.
func TestValuesMergeKeepsTheFirstAnswer(t *testing.T) {
	// A value supplied empty is a value all the same.
	vals := Values{"Name": "bd-1", "Editor": ""}

	vals.Merge(Values{"Name": "the resolver's", "Editor": "vi", "Shell": "fish"})

	want := Values{"Name": "bd-1", "Editor": "", "Shell": "fish"}
	testenv.Equal(t, want, vals, "Merge overwrote a name already set")
}
