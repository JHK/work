package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/JHK/work-cli/internal/work"
)

// enter resolves the target, brings its worktree into being, and hands the
// terminal to a session inside it.
func enter(o options, target string) error {
	env, err := work.Open(".")
	if err != nil {
		return err
	}

	var t work.Target
	if target == "" {
		t, err = pick(env)
	} else {
		t, err = work.Resolve(target)
	}
	if err != nil {
		return err
	}

	h, err := decide(env, env.Inspect(t), o)
	if err != nil {
		return err
	}
	return h.Exec()
}

// decide turns the inspected state into the handoff, provisioning and claiming
// along the way. Creating a worktree is the moment work on that target begins.
func decide(env work.Env, s work.State, o options) (work.Handoff, error) {
	launching := !s.Exists && !o.shell
	// Claiming marks a bead as being worked, so the vetting that guards it runs
	// before anything is created.
	claiming := launching && s.Target.Kind == work.KindBead
	if claiming {
		if s.TicketErr != nil {
			return work.Handoff{}, s.TicketErr
		}
		if s.Reason != "" {
			return work.Handoff{}, errors.New(s.Reason)
		}
	}

	if err := env.Provision(s); err != nil {
		return work.Handoff{}, err
	}

	if claiming {
		if err := env.Claim(s.Target); err != nil {
			return work.Handoff{}, err
		}
	}

	h := work.Handoff{Dir: s.Path}
	if launching {
		h.Run = s.SessionLaunch(o.model, o.effort).Argv()
		return h, nil
	}
	report(s)
	h.Run = work.Shell()
	return h, nil
}

// report surfaces what the worktree already carries, so the shell you land in
// starts with its sessions one paste away.
func report(s work.State) {
	if s.SessionsErr != nil {
		fmt.Fprintln(os.Stderr, "work: could not read session history:", s.SessionsErr)
		return
	}
	if len(s.Sessions) == 0 {
		fmt.Println("No prior Claude sessions here. Start one with:", commandLine(work.Launch{}))
		return
	}
	for _, sess := range s.Sessions {
		fmt.Printf("  %s\n    %s\n", sess.Title, commandLine(work.Launch{Resume: sess.ID}))
	}
}

// commandLine renders a launch as the line the user would type.
func commandLine(l work.Launch) string {
	return strings.Join(l.Argv(), " ")
}
