// Package open hands a worktree to one of the commands the [open] table names:
// the shell, an editor, a diff. There is one action here rather than three,
// because a command is all any of them is. What tells them apart is which values
// its key may place, which the settings already say, so the shell that needs
// $SHELL, the editor that needs $EDITOR and the diff that needs a base to
// measure against differ in what [Values] supplies them and in nothing else.
package open

import (
	"cmp"
	"os"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/worktree"
)

// Action is one command the [open] table names.
type Action struct {
	name    string
	usage   string // the line --help shows for the flag that names it
	command config.Command
}

// Shell, Editor and Diff are the three the table holds, each named as the flag
// that reaches it and the key that spells it.
func Shell(o config.Open) Action {
	return Action{string(config.ActionShell), "hand the worktree to open.shell, your login shell by default", o.Shell()}
}

func Editor(o config.Open) Action {
	return Action{string(config.ActionEditor), "hand the worktree to open.editor, $VISUAL else $EDITOR by default", o.Editor()}
}

func Diff(o config.Open) Action {
	return Action{string(config.ActionDiff), "hand the worktree to open.diff, its diff against the fork point by default", o.Diff()}
}

func (a Action) Name() string { return a.name }

// Flag is what names this action on the command line, which is the key that
// spells it.
func (a Action) Flag() (string, string) { return a.name, a.usage }

// Applies refuses where the values in hand leave no command to run, which is what
// an editor the environment never named comes to. A shell has a fallback every
// machine has and a diff's base is a constant revision, so neither is refused
// here without the settings saying so.
func (a Action) Applies(vals worktree.Values) error {
	if err := a.command.Applies(vals); err != nil {
		return worktree.Absent(err)
	}
	return nil
}

// Open renders the command work replaces itself with.
func (a Action) Open(t worktree.Tree, vals worktree.Values) (worktree.Handoff, error) {
	run, err := a.command.Render(vals)
	if err != nil {
		return worktree.Handoff{}, err
	}
	return worktree.Handoff{Dir: t.Path, Run: run}, nil
}

// Values supplies what the [open] table's keys ask for beyond the worktree
// itself. It is a source rather than three actions' private business, so which key
// may place what it named stays the settings' allowlist to say.
type Values struct{}

// Supply reads the environment for the two keys that name it. The base is a
// revision handed over for git to resolve rather than one read here, so it is the
// same whatever the worktree and is supplied for one that is not there yet.
func (Values) Supply(t worktree.Tree) (worktree.Values, error) {
	return worktree.Values{
		"Shell":  cmp.Or(os.Getenv("SHELL"), "/bin/sh"),
		"Editor": cmp.Or(os.Getenv("VISUAL"), os.Getenv("EDITOR")),
		"Base":   base,
	}, nil
}

// base is a revision for git to resolve: the main checkout's head, which git
// names from any worktree of the repository. Taking the merge-base of it and the
// worktree's own head is the diff command's, so that what the diff is measured
// against is one commit and the working tree is the other side of it.
const base = "main-worktree/HEAD"
