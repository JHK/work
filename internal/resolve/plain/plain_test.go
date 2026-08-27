package plain

import (
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// The verb that names this resolver takes an identifier at its word: a number is
// a worktree of that name rather than a pull request.
//
// No command reaches it. A forge claims a bare number by its spelling whether or
// not the settings asked for one, so add never invents a name of it.
func TestIdentifyTakesANumberAtItsWord(t *testing.T) {
	got, err := Named(t.TempDir(), t.TempDir()).Identify("7", worktree.Open{})

	require.NoError(t, err)
	testenv.Equal(t, worktree.Place{ID: "7", Name: "7", Branch: "7"}, got, "Identify names the wrong place")
}
