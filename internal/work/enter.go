package work

import "errors"

// Options are the choices a front end makes on the way in, beyond the target
// itself.
type Options struct {
	Shell  bool   // create the worktree without claiming the bead or launching a session
	Model  string // model for the launched session
	Effort string // effort for the launched session
}

// Entry is what Enter arrived at: the handoff to run, and how it got there, so
// a front end can report the parts it wants to show.
type Entry struct {
	Handoff  Handoff
	State    State
	Launched bool // the handoff starts a session, rather than handing over the shell
}

// Enter takes a target from inspection to the handoff, vetting, provisioning
// and claiming along the way. Creating a worktree is the moment work on that
// target begins.
func (e Env) Enter(t Target, o Options) (Entry, error) {
	s := e.Inspect(t)

	launching := !s.Exists && !o.Shell
	// Claiming marks a bead as being worked, so the vetting that guards it runs
	// before anything is created.
	claiming := launching && s.Target.Kind == KindBead
	if claiming {
		if s.TicketErr != nil {
			return Entry{}, s.TicketErr
		}
		if s.Reason != "" {
			return Entry{}, errors.New(s.Reason)
		}
	}

	if err := e.Provision(s); err != nil {
		return Entry{}, err
	}

	if claiming {
		if err := e.Claim(s.Target); err != nil {
			return Entry{}, err
		}
	}

	entry := Entry{State: s, Launched: launching, Handoff: Handoff{Dir: s.Path}}
	if launching {
		entry.Handoff.Run = s.SessionLaunch(o.Model, o.Effort).Argv()
	} else {
		entry.Handoff.Run = Shell()
	}
	return entry, nil
}
