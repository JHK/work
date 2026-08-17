package work

import (
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// defaultDir is where worktrees go with nothing configured.
var defaultDir = config.Default().Worktree.Directory

// namesGit reports whether a refusal hands git's own words on: the command work
// put to it, and what git answered. Every refusal of work's own preconditions is
// free of both.
func namesGit(err error) bool {
	said := err.Error()
	return strings.HasPrefix(said, "git ") || strings.Contains(said, "fatal:")
}
