package cli

import (
	"errors"

	"github.com/JHK/work-cli/internal/work"
)

// enter resolves the target, asks work to bring its worktree into being, and
// hands the terminal over to what came back.
func enter(o options, target string) error {
	env, err := work.Open(".")
	if err != nil {
		return err
	}
	c, err := candidate(env, o, target)
	if err != nil {
		return err
	}

	e, err := env.Enter(c, work.Options{Action: o.action(), Ask: ask, NoClaim: o.noClaim})
	if err != nil {
		return err
	}
	return e.Handoff.Exec()
}

// candidate is the place to work: --create takes the name as its own, an
// identifier names one outright, and without either the picker hands one over.
func candidate(env work.Env, o options, target string) (work.Candidate, error) {
	switch {
	// The picker offers what exists; a bare worktree is created by name only.
	case o.create && target == "":
		return work.Candidate{}, errors.New("--create needs a name")
	case o.create:
		return env.Create(target)
	case target == "":
		return pick(env)
	}
	return env.Resolve(target)
}
