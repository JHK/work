// Package mise grants a fresh worktree the config trust its sessions need. It is
// the one thing that speaks to mise, so the client it used to sit on is its own.
package mise

import (
	"github.com/JHK/work-cli/internal/run"
	"github.com/JHK/work-cli/internal/worktree"
)

// Name is what this action goes by.
const Name = "mise"

// Trust marks a fresh worktree's mise configs as trusted. It does its work on
// creation and has nothing to do on the way back into a worktree, which is what
// makes it an action rather than something a worktree opens on.
type Trust struct{}

func (Trust) Name() string { return Name }

// Run lets mise find the configs itself rather than reproducing its discovery
// order here. Best effort: a missing mise, or a grant that fails, only means the
// session prompts, so nothing here refuses the worktree that was just made.
func (Trust) Run(t worktree.Tree) error {
	_, _ = run.Output(t.Path, "mise", "trust")
	return nil
}
