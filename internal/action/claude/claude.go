// Package claude hands a worktree to Claude Code: a fresh one to a session opened
// on whatever it was made for, one already there to the conversation it already
// carries.
package claude

import (
	"errors"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/worktree"
)

// Name is what this action goes by: what a flag, an [action] key and its own
// table of commands spell.
const Name = "claude"

// Opener renders the command a worktree opens on.
type Opener struct {
	commands config.Claude
	carried  transcripts
}

// New reads the commands from the settings and the conversations a worktree
// carries from Claude Code's own transcripts.
func New(commands config.Claude) Opener {
	return Opener{commands: commands, carried: recorded{}}
}

func (o Opener) Name() string { return Name }

// Flag is what names this action on the command line.
func (o Opener) Flag() (string, string) {
	return Name, "hand the worktree to claude, a Claude Code session by default"
}

// Open renders the command work replaces itself with. A worktree this run created
// has nothing inside it to return to, so it opens on a session started for whatever
// it was made for; one already there opens on what it already carries.
func (o Opener) Open(t worktree.Tree, vals worktree.Values) (worktree.Handoff, error) {
	run, err := o.command(t, vals)
	if err != nil {
		return worktree.Handoff{}, err
	}
	return worktree.Handoff{Dir: t.Path, Run: run}, nil
}

// command is the first of the keys the values in hand can render, from the most
// that can be said about a worktree to the least: a ticket's id and title, a pull
// request's number, then a bare session. Which key applies follows from what the
// resolver supplied, and nothing here names a tracker or a forge.
func (o Opener) command(t worktree.Tree, vals worktree.Values) ([]string, error) {
	if !t.Created {
		return o.resume(t, vals)
	}
	return first(vals, o.commands.StartTicket(), o.commands.StartPullRequest(), o.commands.StartSession())
}

// resume returns to what the worktree already carries: no conversation starts one,
// a single one is returned to, and several reach claude's own list.
func (o Opener) resume(t worktree.Tree, vals worktree.Values) ([]string, error) {
	list, err := o.carried.list(t.Path)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return o.commands.StartSession().Render(vals)
	}
	// An empty name drops the element that placed it, reaching claude's own list.
	session := ""
	if len(list) == 1 {
		session = list[0]
	}
	// On top of the values rather than into them: they are the core's account of
	// the worktree, and Session is this key's alone.
	resuming := worktree.Values{"Session": session}
	resuming.Merge(vals)
	return o.commands.ResumeSession().Render(resuming)
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
