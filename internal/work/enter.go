package work

import (
	"errors"
	"fmt"
)

// Action is what a worktree is handed over to. One worktree opens on one
// command, so a front end names one of these and cannot name two.
type Action int

const (
	// ActionUnnamed is the moment's own: the launcher for a worktree just
	// created, open.shell for one already there.
	ActionUnnamed Action = iota
	ActionAgent          // the agent, on whatever the worktree already carries
	ActionShell          // open.shell, a worktree just created included
	ActionEditor         // open.editor
	ActionDiff           // open.diff
)

// actionNames are the names the actions go by, in the enum's own order.
var actionNames = [...]string{"unnamed", "agent", "shell", "editor", "diff"}

func (a Action) String() string {
	if a < 0 || int(a) >= len(actionNames) {
		return fmt.Sprintf("Action(%d)", int(a))
	}
	return actionNames[a]
}

// Options are the choices a front end makes on the way in, beyond the target
// itself.
type Options struct {
	Action  Action // what the worktree is handed over to
	NoClaim bool   // create the worktree without claiming the bead
}

// Entry is what Enter arrived at: the handoff to run, and the target it was
// reached through, which a front end reads for what it wants to show.
type Entry struct {
	Handoff Handoff
	State   State
}

// Enter takes a place to work through to the handoff, vetting, provisioning and
// claiming along the way, and spares git and bd the questions finding it already
// answered. Creating a worktree is the moment work on that target begins.
func (e Env) Enter(c Candidate, o Options) (Entry, error) {
	return e.enter(e.inspectAt(c.Target, c.path), c.ready, o)
}

// enter takes an inspected target the rest of the way. ready says whether bd
// has already been heard to call the bead workable.
func (e Env) enter(s State, ready bool, o Options) (Entry, error) {
	// Ahead of the vetting: an editor that cannot be named leaves nothing created
	// or claimed. Every other command is rendered once there is a worktree to run
	// it in, a diff having no merge-base until the branch exists.
	var editor []string
	if o.Action == ActionEditor {
		var err error
		if editor, err = e.Editor(s); err != nil {
			return Entry{}, err
		}
	}

	// Work on a ticket begins with its worktree, whatever that worktree opens on,
	// so the vetting runs wherever one is about to be created and no flag reaches
	// past it. Only the claim it guards is declinable.
	vetting := !s.Exists && s.Target.Kind == KindBead
	if vetting {
		if s.TicketErr != nil {
			return Entry{}, s.TicketErr
		}
		reason, err := e.vet(s.Bead, ready)
		if err != nil {
			return Entry{}, err
		}
		if reason != "" {
			return Entry{}, errors.New(reason)
		}
	}

	if err := e.Provision(s); err != nil {
		return Entry{}, err
	}

	if vetting && !o.NoClaim {
		if err := e.Claim(s.Target); err != nil {
			return Entry{}, err
		}
	}

	entry := Entry{State: s, Handoff: Handoff{Dir: s.Path}}
	if o.Action == ActionEditor {
		entry.Handoff.Run = editor
		return entry, nil
	}

	// The launcher is what a worktree only just created opens on, an unnamed
	// action and an agent alike: there is nothing there to enter yet, and no
	// conversation to return to.
	render := e.Launch
	switch {
	case o.Action == ActionShell, o.Action == ActionUnnamed && s.Exists:
		render = e.Shell
	case o.Action == ActionDiff:
		render = e.Diff
	case o.Action == ActionAgent && s.Exists:
		render = e.Agent
	}
	// Past the provisioning and the claim, so a command that will not render leaves
	// the worktree made and the ticket claimed, as one that will not start does.
	run, err := render(s)
	if err != nil {
		return Entry{}, err
	}
	entry.Handoff.Run = run
	return entry, nil
}
