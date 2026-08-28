// Package claude hands a worktree to Claude Code: a session opened on whatever
// that worktree was made for, and the worktree itself where nothing was.
package claude

import (
	"errors"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/worktree"
)

// Name is what this action goes by, and what its own table of commands spells.
const Name = "claude"

// Opener renders the command a worktree opens on.
type Opener struct {
	commands config.Claude
}

// New reads the commands from the settings.
func New(commands config.Claude) Opener { return Opener{commands: commands} }

func (o Opener) Name() string { return Name }

// Open renders the command work replaces itself with: a session on whatever the
// worktree was made for, or the worktree itself where nothing answered for it.
func (o Opener) Open(t worktree.Tree, vals worktree.Values) (worktree.Handoff, error) {
	for _, c := range []config.Command{o.commands.StartTicket(), o.commands.StartPullRequest()} {
		run, err := c.Render(vals)
		if err == nil {
			return worktree.Handoff{Dir: t.Path, Run: run}, nil
		}
		// Only a value nothing supplied moves to the next key; any other refusal is a
		// misconfigured key.
		if !errors.Is(err, config.ErrUnsupplied) {
			return worktree.Handoff{}, err
		}
	}
	return worktree.Handoff{Dir: t.Path}, nil
}
