package cli

import (
	"errors"

	"github.com/JHK/work-cli/internal/work"
)

// enter resolves the target, asks work to bring its worktree into being, and
// hands the terminal over to what came back.
func enter(env work.Env, o options, target string) error {
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
	// Deleting reaches only what is already open, so the list narrows to it.
	case o.delete && target == "":
		return pickFrom(env.Worktrees, "no worktrees to delete")
	case target == "":
		return pickFrom(env.Candidates, "nothing to work on")
	}
	return env.Resolve(target)
}

// pickFrom puts one listing in front of the picker.
func pickFrom(list func() ([]work.Candidate, error), none string) (work.Candidate, error) {
	candidates, err := list()
	if err != nil {
		return work.Candidate{}, err
	}
	if len(candidates) == 0 {
		return work.Candidate{}, errors.New(none)
	}
	return pick(candidates)
}
