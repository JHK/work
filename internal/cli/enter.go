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
	browsed := target == ""
	if browsed {
		t, err = pick(env)
	} else {
		t, err = work.Resolve(env.Repo, target)
	}
	if err != nil {
		return err
	}
	if o.start && t.Kind == work.KindPR {
		return errors.New("--start works on beads, not PRs")
	}

	h, err := decide(env, env.Inspect(t), o, browsed)
	if err != nil {
		return err
	}
	return h.Exec()
}

// decide turns the inspected state into the handoff, provisioning and claiming
// along the way. A browsed target is confirmed before anything is created; a
// named one was specified fully enough to act on.
func decide(env work.Env, s work.State, o options, browsed bool) (work.Handoff, error) {
	// Claiming, and only claiming, marks a bead as being worked, so the vetting
	// that guards it follows the claim rather than the flag that asked.
	claiming := s.Target.Kind == work.KindBead && !o.shell && (o.start || !s.Exists)
	if claiming {
		if s.TicketErr != nil {
			return work.Handoff{}, s.TicketErr
		}
		if s.Reason != "" {
			return work.Handoff{}, errors.New(s.Reason)
		}
	}

	if !s.Exists {
		branch, err := s.Branch()
		if err != nil {
			return work.Handoff{}, err
		}
		if browsed {
			if err := confirm(fmt.Sprintf("create worktree %s on branch %s", s.Target.Name, branch)); err != nil {
				return work.Handoff{}, err
			}
		}
		if err := env.Provision(s); err != nil {
			return work.Handoff{}, err
		}
	}

	if claiming {
		if err := env.Claim(s.Target); err != nil {
			return work.Handoff{}, err
		}
	}

	h := work.Handoff{Dir: s.Target.Path}
	if o.start {
		h.Run = s.StartLaunch(o.model, o.effort).Argv()
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
