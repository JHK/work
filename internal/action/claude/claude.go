// Package claude hands a worktree to Claude Code: a session opened on whatever
// that worktree was made for.
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

// Open renders the command work replaces itself with: a session started on
// whatever the worktree was made for.
func (o Opener) Open(t worktree.Tree, vals worktree.Values) (worktree.Handoff, error) {
	run, err := first(vals, o.commands.StartTicket(), o.commands.StartPullRequest(), o.commands.StartSession())
	if err != nil {
		return worktree.Handoff{}, err
	}
	return worktree.Handoff{Dir: t.Path, Run: run}, nil
}

// first renders the earliest command every value of which was supplied. Only
// [config.ErrUnsupplied] moves to the next key; any other refusal is a
// misconfigured key, and stops.
func first(vals worktree.Values, commands ...config.Command) ([]string, error) {
	var err error
	for _, c := range commands {
		var run []string
		if run, err = c.Render(vals); err == nil {
			return run, nil
		}
		if !errors.Is(err, config.ErrUnsupplied) {
			return nil, err
		}
	}
	return nil, err
}
