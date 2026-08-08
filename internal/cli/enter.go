package cli

import (
	"github.com/JHK/work-cli/internal/work"
)

// enter resolves the target, asks work to bring its worktree into being, and
// hands the terminal over to what came back.
func enter(env work.Env, o options, target string) error {
	c, err := candidate(env, target)
	if err != nil {
		return err
	}
	return open(env, o, c)
}

// candidate is the place to work: an identifier names one outright, and without
// one the picker hands one over.
func candidate(env work.Env, target string) (work.Candidate, error) {
	if target == "" {
		return pickFrom(env.Candidates, "nothing to work on")
	}
	return env.Resolve(target)
}

// open is where every verb that opens something ends: work brings the worktree
// into being if it has to, and the terminal goes to what came back.
func open(env work.Env, o options, c work.Candidate) error {
	e, err := env.Enter(c, work.Options{Action: o.action(), Ask: ask, NoClaim: o.noClaim})
	if err != nil {
		return err
	}
	return e.Handoff.Exec()
}
