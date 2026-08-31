package work

import (
	"errors"
	"fmt"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/worktree"
)

// Options are the choices a front end makes on the way in, beyond the place
// itself.
type Options struct {
	Verb  string // the verb typed, which is what settles a creation's opener
	Carry bool   // the invoking checkout's working state moves into a worktree just made
}

// Enter takes a place to work through to the handoff, preparing, creating and
// acting along the way.
func (e Env) Enter(c Candidate, o Options) (worktree.Handoff, error) {
	// A candidate without a resolver is one a front end made itself.
	if c.by == nil {
		return worktree.Handoff{}, errors.New("no system answers for this place")
	}

	t, carrying, err := e.create(c, o.Carry)
	if err != nil {
		return worktree.Handoff{}, err
	}

	t.Values = e.values(t)

	if t.Created {
		for _, a := range e.Systems.Actions {
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

	opener, err := e.openerFor(c, o)
	if err != nil {
		return worktree.Handoff{}, err
	}
	return opener.Open(t)
}

// create is the worktree a candidate has, made where it has none, and whether
// the checkout's working state is still to move into it.
func (e Env) create(c Candidate, carry bool) (worktree.Tree, bool, error) {
	t := worktree.Tree{Place: c.Place, Path: c.path, By: c.by}
	if c.Open {
		return t, false, nil
	}
	var err error
	if t.Place, err = c.by.Prepare(c.Place); err != nil {
		return worktree.Tree{}, false, err
	}
	// Preparing is the last moment a resolver may change the name, which is about
	// to become a directory.
	if err := checkName(t.Name); err != nil {
		return worktree.Tree{}, false, err
	}
	if err := e.inside(); err != nil {
		return worktree.Tree{}, false, err
	}
	t.Path, t.Created = e.path(t.Name), true
	if err := git.Vacant(t.Path); err != nil {
		return worktree.Tree{}, false, err
	}
	// Asked before the worktree is made: the directory about to appear under the
	// checkout would itself read as work in hand.
	carrying := carry && e.carries()
	if err := c.by.Create(t.Place, t.Path); err != nil {
		return worktree.Tree{}, false, err
	}
	return t, carrying, nil
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

// openerFor is what the moment opens on: a worktree this run created goes to the
// agent where the settings send that verb's creations, and every other worktree
// is handed back.
func (e Env) openerFor(c Candidate, o Options) (Opener, error) {
	if !c.Open && e.Config.OpensOnCreation(o.Verb) {
		return e.opener(config.ClaudeOpener)
	}
	return e.opener(config.ShellOpener)
}

func (e Env) opener(name string) (Opener, error) {
	for _, op := range e.Systems.Openers {
		if op.Name() == name {
			return op, nil
		}
	}
	return nil, fmt.Errorf("nothing here goes by the action %q", name)
}
