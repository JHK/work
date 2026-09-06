// Package claude hands a worktree to Claude Code: a session opened on whatever
// that worktree was made for, and the worktree itself where nothing was.
package claude

import (
	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/worktree"
)

// Name is what this action goes by, and the table its settings sit in.
const Name = "claude"

// Session is the agent a worktree opens on.
type Session struct {
	command config.Command
}

func New(table config.Claude) Session { return Session{command: table.Command()} }

func (s Session) Name() worktree.IntegrationName { return Name }

// OnCreated has nothing to do: the agent is handed a worktree when it opens.
func (Session) OnCreated(worktree.Tree) error { return nil }

// Open renders the command work replaces itself with, or the worktree itself
// where that command renders to nothing.
func (s Session) Open(t worktree.Tree) (worktree.Handoff, error) {
	run, err := s.command.Render(t.Values)
	if err != nil {
		return worktree.Handoff{}, err
	}
	return worktree.Handoff{Dir: t.Path, Run: run}, nil
}
