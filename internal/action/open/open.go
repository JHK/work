// Package open hands a worktree to the command the [open] table names: the shell
// someone types in. [Values] supplies what that command needs beyond the
// worktree itself.
package open

import (
	"cmp"
	"os"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/worktree"
)

// name is what this action goes by, in the key and the flag alike.
const name = string(config.ActionShell)

// Action is the command the [open] table names.
type Action struct{ command config.Command }

// Shell is that action over the command the settings named.
func Shell(o config.Open) Action { return Action{o.Shell()} }

func (Action) Name() string { return name }

// Flag is what names this action on the command line.
func (Action) Flag() (string, string) {
	return name, "hand the worktree to open.shell, your login shell by default"
}

// Open renders the command work replaces itself with.
func (a Action) Open(t worktree.Tree, vals worktree.Values) (worktree.Handoff, error) {
	run, err := a.command.Render(vals)
	if err != nil {
		return worktree.Handoff{}, err
	}
	return worktree.Handoff{Dir: t.Path, Run: run}, nil
}

// Values supplies what the [open] table's key asks for beyond the worktree
// itself, to whichever command placed it.
type Values struct{}

// Supply reads the environment for the name the key places.
func (Values) Supply(worktree.Tree) (worktree.Values, error) {
	return worktree.Values{
		"Shell": cmp.Or(os.Getenv("SHELL"), "/bin/sh"),
	}, nil
}
