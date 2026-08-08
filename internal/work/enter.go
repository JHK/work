package work

import (
	"errors"

	"github.com/JHK/work-cli/internal/config"
)

// Action is what a worktree is handed over to. One worktree opens on one
// command, so a front end names one of these and cannot name two. It is the
// settings' own type: a flag and an [action] key name the one set of actions,
// and the names they go by are the settings' to refuse.
type Action = config.ActionName

const (
	// ActionUnnamed is no action named, which leaves the moment's own key to
	// settle it: action.create for a worktree about to be made, action.enter for
	// one already there.
	ActionUnnamed Action = ""
	ActionAgent          = config.ActionAgent  // the agent, on whatever the worktree already carries
	ActionShell          = config.ActionShell  // open.shell, a worktree just created included
	ActionEditor         = config.ActionEditor // open.editor
	ActionDiff           = config.ActionDiff   // open.diff
)

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
	return e.enter(e.inspectAt(c), o)
}

// opensOn is the action to hand the worktree to: the one the front end named,
// else the one the moment's key holds.
func (e Env) opensOn(s State, o Options) Action {
	switch {
	case o.Action != ActionUnnamed:
		return o.Action
	case s.Exists:
		return e.Config.Action.Enter()
	default:
		return e.Config.Action.Create()
	}
}

// enter takes an inspected target the rest of the way.
func (e Env) enter(s State, o Options) (Entry, error) {
	action := e.opensOn(s, o)

	// Ahead of the vetting, whichever named the editor: one that cannot be named
	// leaves nothing created or claimed. Every other command is rendered once
	// there is a worktree to run it in, a diff having no merge-base until the
	// branch exists.
	var editor []string
	if action == ActionEditor {
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
		reason, err := e.vet(s.Bead, s.Ready)
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
	if action == ActionEditor {
		entry.Handoff.Run = editor
		return entry, nil
	}

	// The launcher is what the agent means for a worktree only just created: there
	// is nothing there to enter yet, and no conversation to return to.
	render := e.Launch
	switch action {
	case ActionShell:
		render = e.Shell
	case ActionDiff:
		render = e.Diff
	case ActionAgent:
		if s.Exists {
			render = e.Agent
		}
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
