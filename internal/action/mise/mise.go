// Package mise grants a fresh worktree the config trust its sessions need. It is
// the one thing that speaks to mise.
package mise

import (
	"log/slog"

	"github.com/JHK/work-cli/internal/run"
	"github.com/JHK/work-cli/internal/worktree"
)

// Name is what this action goes by.
const Name = "mise"

// Trust marks a fresh worktree's mise configs as trusted.
type Trust struct{}

func (Trust) Name() string { return Name }

// Run lets mise find the configs itself. Best effort: a grant that fails only
// means the session prompts, never the worktree just made.
func (Trust) Run(t worktree.Tree) error {
	if _, err := run.Output(t.Path, "mise", "trust"); err != nil {
		slog.Warn(err.Error())
	}
	return nil
}
