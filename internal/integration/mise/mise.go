// Package mise grants a fresh worktree the config trust its sessions need. It is
// the one thing that speaks to mise.
package mise

import (
	"log/slog"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/run"
	"github.com/JHK/work-cli/internal/worktree"
)

// Trust marks a fresh worktree's mise configs as trusted.
type Trust struct{}

func (Trust) Name() worktree.IntegrationName { return config.MiseIntegration }

// OnCreated lets mise find the configs itself. Best effort: a grant that fails
// only means the session prompts, never the worktree just made.
func (Trust) OnCreated(t worktree.Tree) error {
	if _, err := run.Output(string(t.Path), "mise", "trust"); err != nil {
		slog.Warn(err.Error())
	}
	return nil
}

// Open has nothing to open: a worktree never opens on the tool trust.
func (Trust) Open(worktree.Tree) (worktree.Handoff, error) { return worktree.Handoff{}, nil }
