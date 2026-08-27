package work

import (
	"errors"
	"fmt"
	"slices"

	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/worktree"
)

// Options are the choices a front end makes on the way in, beyond the place
// itself.
type Options struct {
	Open  string   // the action the worktree opens on, empty for the moment's own key
	Skip  []string // actions not to run, under the names they go by
	Carry bool     // the invoking checkout's working state moves into a worktree just made
}

// Enter takes a place to work through to the handoff, preparing, creating and
// acting along the way.
func (e Env) Enter(c Candidate, o Options) (worktree.Handoff, error) {
	// A candidate without a resolver is one a front end made itself.
	if c.by == nil {
		return worktree.Handoff{}, errors.New("no system answers for this place")
	}

	// Named ahead of everything, so a name no action goes by is refused before
	// anything is asked of a tracker.
	opener, err := e.named(c, o)
	if err != nil {
		return worktree.Handoff{}, err
	}

	// Only a worktree about to be made is prepared, which is where a ticket that
	// cannot be worked is refused.
	t := worktree.Tree{Place: c.Place, Path: c.path, By: c.by}
	carrying := false
	if !c.Open {
		if t.Place, err = c.by.Prepare(c.Place); err != nil {
			return worktree.Handoff{}, err
		}
		// Preparing is the last moment a resolver may change the name, which is about
		// to become a directory.
		if err := checkName(t.Name); err != nil {
			return worktree.Handoff{}, err
		}
		if err := e.inside(); err != nil {
			return worktree.Handoff{}, err
		}
		t.Path, t.Created = e.path(t.Name), true
		if err := git.Vacant(t.Path); err != nil {
			return worktree.Handoff{}, err
		}
		// Asked before the worktree is made: the directory about to appear under the
		// checkout would itself read as work in hand.
		carrying = o.Carry && e.carries()
		if err := c.by.Create(t.Place, t.Path); err != nil {
			return worktree.Handoff{}, err
		}
	}

	// A worktree that was already there runs none of them.
	if t.Created {
		for _, a := range e.Systems.Actions {
			if slices.Contains(o.Skip, a.Name()) {
				continue
			}
			if err := a.Run(t); err != nil {
				return worktree.Handoff{}, err
			}
		}
	}

	// Last of all that can refuse, so nothing is moved out of the checkout for a
	// run that does not go through with the worktree.
	if carrying {
		if err := e.carry(t.Path); err != nil {
			return worktree.Handoff{}, err
		}
	}

	// Past the creation and the actions, so the resolver is asked of a worktree that
	// exists.
	return opener.Open(t, e.values(t))
}

// Carryable is why the checkout work was invoked in has nothing to hand over: it
// carries no changes.
func (e Env) Carryable() error {
	if e.carries() {
		return nil
	}
	return errors.New("this checkout carries no changes")
}

// carries reports whether the checkout work was invoked in holds a working state
// to hand over.
func (e Env) carries() bool {
	return e.Dir != "" && git.Dirty(e.Dir)
}

// carry moves the working state of the checkout work was invoked in into the
// worktree at to.
func (e Env) carry(to string) error {
	saved, err := git.Stash(e.Dir)
	if err != nil {
		return fmt.Errorf("%w; the worktree at %s is made, and the changes are where they were", err, to)
	}
	if !saved {
		return nil
	}
	if err := git.Unstash(to); err != nil {
		return fmt.Errorf("%w; the changes are stashed and %s carries part of them already: put that right, then git stash drop", err, to)
	}
	return nil
}

// named is the action a flag or the moment's own key named: action.create for a
// worktree about to be made, action.enter for one already there.
func (e Env) named(c Candidate, o Options) (Opener, error) {
	name := o.Open
	if name == "" {
		if c.Open {
			name = string(e.Config.Action.Enter())
		} else {
			name = string(e.Config.Action.Create())
		}
	}
	return e.opener(name)
}

// opener is the action that goes by a name.
func (e Env) opener(name string) (Opener, error) {
	for _, op := range e.Systems.Openers {
		if op.Name() == name {
			return op, nil
		}
	}
	return nil, fmt.Errorf("nothing here goes by the action %q", name)
}
