package work

import (
	"errors"
	"fmt"
	"slices"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/worktree"
)

// ask is the name in the action namespace that names no action: it stands for the
// choice between the ones that do, which the core settles before any of them runs.
// It cannot be an action itself, because an action is handed a worktree that
// exists and a screen dismissed has to leave nothing created.
const ask = string(config.ActionAsk)

// Ask puts the choice between the actions offered and returns the name chosen.
// Drawing it is the front end's, so work names the offer and nothing more.
type Ask func(offer []string) (string, error)

// Options are the choices a front end makes on the way in, beyond the place
// itself.
type Options struct {
	Open string   // the action the worktree opens on, empty for the moment's own key
	Ask  Ask      // the screen ask reaches; nothing can ask without one
	Skip []string // actions not to run, under the names they go by
}

// Enter takes a place to work through to the handoff, preparing, creating and
// acting along the way, and spares the resolvers the questions finding it already
// answered.
func (e Env) Enter(c Candidate, o Options) (worktree.Handoff, error) {
	// Every candidate comes from a resolver, so one without is a front end that made
	// its own rather than a place to work.
	if c.by == nil {
		return worktree.Handoff{}, errors.New("no system answers for this place")
	}
	place := c.Place

	// Named ahead of everything, so that a name no action goes by is refused before
	// anything is asked of a tracker. Whether the action applies is judged below,
	// once there are values to judge it against.
	opener, err := e.named(c, o)
	if err != nil {
		return worktree.Handoff{}, err
	}

	// Only a worktree about to be made is prepared: the branch and the refusal are
	// what making one needs, and re-entering one needs neither. This is where a
	// ticket that cannot be worked is refused, and it holds whatever the worktree
	// would have opened on and whichever actions were declined.
	if !c.Open {
		if place, err = c.by.Prepare(place); err != nil {
			return worktree.Handoff{}, err
		}
		// Completing a place is the one moment a resolver may still change its name,
		// and the name is about to become a directory: the rule is the core's wherever
		// a name reaches a path.
		if err := checkName(place.Name); err != nil {
			return worktree.Handoff{}, err
		}
	}

	// The values a worktree that may not exist yet can already be described by,
	// gathered once for both ways an action is settled: an action a flag named and one
	// the screen chose are judged against the same account of the place. A source that
	// can only answer from inside a worktree supplies nothing here, which is what tells
	// a tool the machine does not have from a value still coming.
	t := worktree.Tree{Place: place, Path: c.path, By: c.by}
	vals := e.values(t)

	// Past the refusal, so a place that cannot be worked is never asked about, and
	// ahead of the creation, so an action that cannot run and a screen dismissed both
	// leave nothing created and nothing claimed.
	if opener == nil {
		if opener, err = e.chosen(vals, o); err != nil {
			return worktree.Handoff{}, err
		}
	} else if err := opener.Applies(vals); err != nil {
		return worktree.Handoff{}, err
	}

	if !c.Open {
		t.Path, t.Created = e.path(place.Name), true
		if err := c.by.Create(place, t.Path); err != nil {
			return worktree.Handoff{}, err
		}
		// Gathered again now that there is a worktree, so the sources that could only
		// answer from inside one are asked where they can answer. A worktree that was
		// already there was never described without one.
		vals = e.values(t)
	}

	// Every action the worktree coming into being means, in the order they were
	// wired. A worktree that was already there means none of them.
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

	// Past the creation and the actions, so a command that will not render leaves the
	// worktree made and the ticket claimed, as one that will not start does.
	return opener.Open(t, vals)
}

// Handoff is the command the named action makes of a tree, rendered against the
// values every source supplies for it. It is the sequence's last step alone, for
// a caller holding something to hand over that no resolver made and no action
// runs on: nothing is prepared, created or acted on here.
func (e Env) Handoff(t worktree.Tree, action string) (worktree.Handoff, error) {
	op, err := e.opener(action)
	if err != nil {
		return worktree.Handoff{}, err
	}
	return op.Open(t, e.values(t))
}

// named is the action a flag or the moment's own key named, or nil where that
// name is the screen: action.create for a worktree about to be made,
// action.enter for one already there.
func (e Env) named(c Candidate, o Options) (Opener, error) {
	name := o.Open
	if name == "" {
		if c.Open {
			name = string(e.Config.Action.Enter())
		} else {
			name = string(e.Config.Action.Create())
		}
	}
	if name == ask {
		return nil, nil
	}
	return e.opener(name)
}

// chosen puts the offer to the screen and returns the action it came back with.
func (e Env) chosen(vals worktree.Values, o Options) (Opener, error) {
	if o.Ask == nil {
		return nil, errors.New("nothing here can ask which action to open on; name one with a flag")
	}
	offer, err := e.offer(vals)
	if err != nil {
		return nil, err
	}
	if len(offer) == 0 {
		return nil, errors.New("no action applies to this worktree")
	}
	name, err := o.Ask(offer)
	if err != nil {
		return nil, err
	}
	// A screen answering with the screen, or with nothing, has settled nothing, and
	// there is no action to fall through to.
	if name == ask || name == "" {
		return nil, errors.New("the screen named no action to open on")
	}
	return e.opener(name)
}

// offer are the actions the screen chooses between: the ones the values in hand
// leave a command to run, so what is offered is what will run. The order is the one
// they were wired in, which is the one they are drawn in.
//
// Only [worktree.ErrAbsent] leaves an action off: an action refusing for any other
// reason has failed rather than found nothing to hand the worktree to, and stops the
// run where it would otherwise vanish from the screen.
func (e Env) offer(vals worktree.Values) ([]string, error) {
	var out []string
	for _, op := range e.Systems.Openers {
		err := op.Applies(vals)
		if errors.Is(err, worktree.ErrAbsent) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, op.Name())
	}
	return out, nil
}

// opener is the action that goes by a name, and is where a name no action goes by
// is refused. A name one of the switched-off systems goes by is refused as that
// rather than as a name work has never heard of.
func (e Env) opener(name string) (Opener, error) {
	for _, op := range e.Systems.Openers {
		if op.Name() == name {
			return op, nil
		}
	}
	for _, s := range e.Systems.Disabled {
		if s.Name() == name {
			return nil, errors.New(off(name))
		}
	}
	return nil, fmt.Errorf("nothing here goes by the action %q", name)
}
