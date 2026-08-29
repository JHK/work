// Package claude hands a worktree to Claude Code: a session opened on whatever
// that worktree was made for, and the worktree itself where nothing was.
package claude

import (
	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/worktree"
)

// Name is what this action goes by, and the table its settings sit in.
const Name = "claude"

// Opener renders the command a worktree opens on.
type Opener struct {
	command config.Command
}

func New(table config.Claude) Opener { return Opener{command: table.Command()} }

func (o Opener) Name() string { return Name }

// Open renders the command work replaces itself with, or the worktree itself
// where that command renders to nothing.
func (o Opener) Open(t worktree.Tree, vals worktree.Values) (worktree.Handoff, error) {
	run, err := o.command.Render(vals)
	if err != nil {
		return worktree.Handoff{}, err
	}
	return worktree.Handoff{Dir: t.Path, Run: run}, nil
}
