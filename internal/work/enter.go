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
	// ActionAsk is the choice between the four above, put to the person entering.
	// It names no command of its own, so it never reaches the handoff.
	ActionAsk = config.ActionAsk
)

// Ask puts the choice between the actions offered and returns the one chosen.
// Drawing it is the front end's, so work names the offer and nothing more.
type Ask func(offer []Action) (Action, error)

// Options are the choices a front end makes on the way in, beyond the target
// itself.
type Options struct {
	Action  Action // what the worktree is handed over to
	Ask     Ask    // the screen ActionAsk reaches; nothing can ask without one
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
// else the one the moment's key holds. Either may be ask, which settles it no
// further: the screen does.
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

// offer are the actions the screen chooses between: never ask, which is the
// screen itself, and no editor where nothing can name one, so what is offered is
// what will run. The order is the one they are drawn in.
func (e Env) offer(s State) []Action {
	out := []Action{ActionAgent, ActionShell}
	if _, err := e.Editor(s); err == nil {
		out = append(out, ActionEditor)
	}
	return append(out, ActionDiff)
}

// enter takes an inspected target the rest of the way.
func (e Env) enter(s State, o Options) (Entry, error) {
	action := e.opensOn(s, o)

	// Ahead of the vetting, wherever the editor was named outright: one that cannot
	// be named leaves nothing created or claimed, and the screen leaves it off
	// instead. Every other command is rendered once there is a worktree to run it
	// in, a diff having no merge-base until the branch exists.
	if action == ActionEditor {
		if _, err := e.Editor(s); err != nil {
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
		reason, err := vetBead(s.Bead, e.readiness(s))
		if err != nil {
			return Entry{}, err
		}
		if reason != "" {
			return Entry{}, errors.New(reason)
		}
	}

	// Past the vetting, so a ticket that cannot be worked is refused rather than
	// asked about, and ahead of the provisioning, so a screen dismissed here
	// leaves nothing created and nothing claimed.
	if action == ActionAsk {
		if o.Ask == nil {
			return Entry{}, errors.New("nothing here can ask which action to open on; name one with a flag")
		}
		chosen, err := o.Ask(e.offer(s))
		if err != nil {
			return Entry{}, err
		}
		// A screen answering with ask or with nothing has settled nothing, and the
		// launcher below is no answer to fall through to.
		if chosen == ActionAsk || chosen == ActionUnnamed {
			return Entry{}, errors.New("the screen named no action to open on")
		}
		action = chosen
	}

	if err := e.Provision(s); err != nil {
		return Entry{}, err
	}

	if vetting && !o.NoClaim {
		if err := e.Claim(s.Target); err != nil {
			return Entry{}, err
		}
	}

	// The launcher is what the agent means for a worktree only just created: there
	// is nothing there to enter yet, and no conversation to return to.
	render := e.Launch
	switch action {
	case ActionShell:
		render = e.Shell
	case ActionEditor:
		render = e.Editor
	case ActionDiff:
		render = e.Diff
	case ActionAgent:
		if s.Exists {
			render = e.Agent
		}
	}
	// Past the provisioning and the claim, so a command that will not render leaves
	// the worktree made and the ticket claimed, as one that will not start does.
	// The editor alone was rendered ahead of both, and renders the same twice.
	run, err := render(s)
	if err != nil {
		return Entry{}, err
	}
	return Entry{State: s, Handoff: Handoff{Dir: s.Path, Run: run}}, nil
}
