package work

import "errors"

// Options are the choices a front end makes on the way in, beyond the target
// itself.
type Options struct {
	Shell   bool   // hand the worktree to open.shell instead of launching a session
	Editor  bool   // hand the worktree to open.editor instead of a session or a shell
	Diff    bool   // hand the worktree to open.diff instead of a session or a shell
	NoClaim bool   // create the worktree without claiming the bead
	Model   string // model for the launched session
	Effort  string // effort for the launched session
}

// Entry is what Enter arrived at: the handoff to run, and the target it was
// reached through, which a front end reads for what it wants to show.
type Entry struct {
	Handoff Handoff
	State   State
}

// Enter takes a target from inspection to the handoff, vetting, provisioning
// and claiming along the way. Creating a worktree is the moment work on that
// target begins.
func (e Env) Enter(t Target, o Options) (Entry, error) {
	return e.enter(e.Inspect(t), false, o)
}

// EnterCandidate enters one of the candidates Candidates offered, sparing git
// and bd the questions that listing already answered.
func (e Env) EnterCandidate(c Candidate, o Options) (Entry, error) {
	return e.enter(e.inspectAt(c.Target, c.path), c.ready, o)
}

// enter takes an inspected target the rest of the way. ready says whether bd
// has already been heard to call the bead workable.
func (e Env) enter(s State, ready bool, o Options) (Entry, error) {
	// Ahead of the vetting: an editor that cannot be named leaves nothing created
	// or claimed. Every other command is rendered once there is a worktree to run
	// it in, a diff having no merge-base until the branch exists.
	var editor []string
	if o.Editor {
		var err error
		if editor, err = e.Editor(s, o); err != nil {
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

	// A worktree only just created opens on the launcher, unless --shell asked
	// otherwise; everything else lands in the shell.
	launching := !s.Exists && !o.Shell

	entry := Entry{State: s, Handoff: Handoff{Dir: s.Path}}
	if o.Editor {
		entry.Handoff.Run = editor
		return entry, nil
	}

	render := e.Shell
	switch {
	case o.Diff:
		render = e.Diff
	case launching:
		render = e.Launch
	}
	// Past the provisioning and the claim, so a command that will not render leaves
	// the worktree made and the ticket claimed, as one that will not start does.
	run, err := render(s, o)
	if err != nil {
		return Entry{}, err
	}
	entry.Handoff.Run = run
	return entry, nil
}
