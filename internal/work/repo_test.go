package work

import (
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// defaultDir is where worktrees go with nothing configured.
var defaultDir = config.Default().Worktree.Directory
