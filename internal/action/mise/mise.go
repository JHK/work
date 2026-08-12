// Package mise grants a fresh worktree the config trust its sessions need. It is
// the one thing that speaks to mise.
package mise

import (
	"github.com/JHK/work-cli/internal/run"
	"github.com/JHK/work-cli/internal/worktree"
)

// Name is what this action goes by.
const Name = "mise"

// Trust marks a fresh worktree's mise configs as trusted.
type Trust struct{}

func (Trust) Name() string { return Name }

// Run lets mise find the configs itself. Best effort: a grant that fails only
// means the session prompts, so it never refuses the worktree just made.
func (Trust) Run(t worktree.Tree) error {
	_, _ = run.Output(t.Path, "mise", "trust")
	return nil
}
