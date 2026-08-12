// Package work is the core beneath every front end: it asks the resolvers what
// an identifier is, makes the worktree if it is not there, and hands it to an
// action.
package work

import (
	"cmp"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"sync"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/worktree"
)

// Resolver is the near seam: it says which places are its own, offers the picker
// its candidates, and creates the worktree.
type Resolver interface {
	worktree.System

	// Identify names the place behind what the core is holding: an identifier a
	// person typed, or a worktree the repository has open, which a resolver tells
	// apart by [worktree.Open.None]. An identifier is read into a place rather than
	// checked against the system behind it; whether that place exists is
	// [Resolver.Prepare]'s question. Where both are in hand the identifier is one to
	// confirm, and confirming it must ask no more than naming the worktree from its
	// branch alone would.
	//
	// A resolver owns a worktree under the branch it named that worktree by, which
	// is what [Env.Branches] prints, or a listing prints what cannot be retyped.
	//
	// [worktree.ErrUnknown] means this resolver does not answer for what it was
	// shown, and the next one is asked. Every other error stops the run.
	Identify(id string, o worktree.Open) (worktree.Place, error)

	// Offer lists the places worth starting, whether or not they have a worktree.
	Offer() ([]worktree.Place, error)

	// Prepare completes a place a worktree is about to be made for, naming the
	// branch, or says why no worktree may be made for it. Both are free of
	// consequence, so the core runs it ahead of the creation.
	Prepare(p worktree.Place) (worktree.Place, error)

	// Create checks the place's branch out into a worktree at path.
	Create(p worktree.Place, path string) error
}

// Action is the far seam's running half. Every one of them runs, in order, and
// none of them runs on the way back into a worktree that was already there.
type Action interface {
	worktree.System
	Run(t worktree.Tree) error
}

// Opener is the far seam's other half: the one action a worktree opens on, of
// which exactly one per run does.
type Opener interface {
	worktree.System

	// Open renders the handoff from the values every source supplied. The values
	// are the core's account of the worktree, so an action with a name of its own
	// to place renders against a copy.
	Open(t worktree.Tree, vals worktree.Values) (worktree.Handoff, error)
}

// Systems are the implementations behind the seams, in the order they are asked.
type Systems struct {
	Resolvers []Resolver
	Actions   []Action
	Openers   []Opener

	// Sources are asked what they know about any worktree, whichever resolver
	// answered for it. The resolver that did is asked first and separately.
	Sources []worktree.Source

	// Named is the one resolver a verb reaches by name rather than through the
	// chain: add's, which is handed a name of the user's own rather than an
	// identifier to recognise.
	Named Resolver

	// Disabled are the systems the settings left out, in the order they would have
	// been asked in. They are asked one question and no more: whether an identifier
	// nothing here answered for is spelled as one of theirs.
	Disabled []worktree.System
}

// Wiring names the systems for a repository once its settings are read.
// checkout is where work was invoked, whose HEAD a new branch forks from; the
// worktree itself still lands under repo. A front end asks with neither and
// reads the names and flags alone off what comes back, so a system wired here
// reaches for nothing until it is asked a question.
type Wiring func(repo, checkout string, cfg config.Config) Systems

// Env is the repository work operates on, the settings it reads, and the systems
// behind its seams.
type Env struct {
	Repo    string
	Config  config.Config
	Systems Systems
}

// Open finds the repository containing dir, reads its settings and wires the
// systems that answer for it.
func Open(dir string, wire Wiring) (Env, error) {
	repo, err := git.Root(dir)
	if err != nil {
		return Env{}, err
	}
	cfg, err := config.Load(repo)
	if err != nil {
		return Env{}, err
	}
	return Env{Repo: repo, Config: cfg, Systems: wire(repo, dir, cfg)}, nil
}

// The name becomes a directory of its own and an argument to git, so it may not
// traverse, and may not open with a dash.
var worktreeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// checkName holds a name to the worktree-name rule.
func checkName(name string) error {
	if !worktreeName.MatchString(name) {
		return fmt.Errorf("%q is not a usable worktree name", name)
	}
	return nil
}

// Candidate is one place to work: the place a resolver made of it, the worktree
// it already has, and the resolver to go back to for the rest.
type Candidate struct {
	worktree.Place
	Open bool   // a worktree for it already exists
	Icon string // the mark that resolver draws its rows with, empty where it named none

	by     Resolver
	path   string // where that worktree sits
	branch string // what it has checked out, empty when it is detached
}

// Resolve maps an identifier to the place it names and to the worktree that
// place already has.
func (e Env) Resolve(arg string) (Candidate, error) {
	if arg == "" {
		return Candidate{}, errors.New("no target given")
	}
	open, err := e.Worktrees()
	if err != nil {
		return Candidate{}, err
	}
	// A worktree listed under the name is that worktree, whatever a resolver would
	// make of the name, so what the picker shows is entered by retyping it.
	var named []Candidate
	for _, c := range open {
		if c.Name == arg {
			named = append(named, c)
		}
	}
	if len(named) > 0 {
		return shortest(named), nil
	}

	r, p, err := e.identify(arg, worktree.Open{})
	if err != nil {
		return Candidate{}, err
	}
	// A place no worktree could be made for is one nothing really answered for.
	if err := checkName(p.Name); err != nil {
		return Candidate{}, e.switchedOff(arg, err)
	}
	return e.locate(r, p, open)
}

// switchedOff is the refusal an identifier gets where a system the settings left
// out would have answered for it. The question is [worktree.Claimant] and never
// [Resolver.Identify], whose last resolver takes whatever is left.
func (e Env) switchedOff(id string, err error) error {
	for _, s := range e.Systems.Disabled {
		c, ok := s.(worktree.Claimant)
		if !ok {
			continue
		}
		p, its := c.Claims(id)
		if !its || checkName(p.Name) != nil {
			continue
		}
		return fmt.Errorf("nothing answers for %q; %s", id, off(s.Name()))
	}
	return err
}

// off is how a system the settings left out reads: the name it goes by and the
// key that puts it back.
func off(name string) string {
	return fmt.Sprintf("%s is off, put it back with %s = true", name, config.SystemKey(name))
}

// answered stamps a candidate with the resolver that answered for it: the name,
// the mark a screen draws it with, and the resolver itself.
func answered(r Resolver, c Candidate) Candidate {
	c.Source, c.by = r.Name(), r
	if d, ok := r.(worktree.Drawn); ok {
		c.Icon = d.Icon()
	}
	return c
}

// identify is the chain: the first resolver to answer for what the core is
// holding owns it, and one that does not recognise it passes it on.
func (e Env) identify(id string, o worktree.Open) (Resolver, worktree.Place, error) {
	for _, r := range e.Systems.Resolvers {
		p, err := r.Identify(id, o)
		if errors.Is(err, worktree.ErrUnknown) {
			continue
		}
		if err != nil {
			return nil, worktree.Place{}, err
		}
		return r, p, nil
	}
	// The last resolver answers for whatever is left, so a worktree never runs the
	// chain out: nothing recognising one is the listing failing.
	if o.None() {
		return nil, worktree.Place{}, e.switchedOff(id, fmt.Errorf("nothing answers for %q", id))
	}
	return nil, worktree.Place{}, fmt.Errorf("nothing answers for the worktree at %s", o.Path)
}

// Add is the place a name of the user's own makes: a worktree with no ticket and
// no pull request behind it, on a branch spelled exactly as the name is. The
// chain would read the name as a ticket id, so it is never asked.
func (e Env) Add(name string) (Candidate, error) {
	if err := checkName(name); err != nil {
		return Candidate{}, err
	}
	p, err := e.Systems.Named.Identify(name, worktree.Open{})
	if err != nil {
		return Candidate{}, err
	}
	return answered(e.Systems.Named, Candidate{Place: p}), nil
}

// Worktrees is every worktree the repository has, the main checkout excepted,
// each under the place the first resolver to answer for it says it stands for.
func (e Env) Worktrees() ([]Candidate, error) {
	// Without a repository git would answer for whatever directory the process
	// happens to be in.
	if e.Repo == "" {
		return nil, nil
	}
	list, err := git.Linked(e.Repo)
	if err != nil {
		return nil, err
	}
	out := make([]Candidate, 0, len(list))
	for _, w := range list {
		o := worktree.Open{Path: w.Path, Branch: w.Branch}
		c, err := e.adopt(o)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// Branches is what the repository's worktrees have checked out, the main
// checkout excepted, in git's order: a detached one under its directory. It asks
// git and nothing else, so a listing costs no tracker and no forge.
func (e Env) Branches() ([]string, error) {
	if e.Repo == "" {
		return nil, nil
	}
	list, err := git.Linked(e.Repo)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(list))
	for _, w := range list {
		out = append(out, worktree.Open{Path: w.Path, Branch: w.Branch}.Name())
	}
	return out, nil
}

// adopt is the place an open worktree stands for, from the first resolver that
// answers for it.
func (e Env) adopt(o worktree.Open) (Candidate, error) {
	r, p, err := e.identify("", o)
	if err != nil {
		return Candidate{}, err
	}
	return answered(r, Candidate{Place: p, Open: true, path: o.Path, branch: o.Branch}), nil
}

// locate is the worktree a resolved place already has: the resolver that named
// the place, asked of each open worktree whether it is that place's.
func (e Env) locate(r Resolver, p worktree.Place, open []Candidate) (Candidate, error) {
	var found []Candidate
	for _, c := range open {
		o := worktree.Open{Path: c.path, Branch: c.branch}
		its, err := r.Identify(p.ID, o)
		if errors.Is(err, worktree.ErrUnknown) {
			continue
		}
		if err != nil {
			return Candidate{}, err
		}
		// The place as the resolver describes it on that worktree, which knows the
		// branch it found the place by rather than the one it would render today.
		found = append(found, answered(r, Candidate{Place: its, Open: true, path: c.path, branch: c.branch}))
	}
	if len(found) == 0 {
		return answered(r, Candidate{Place: p}), nil
	}
	return shortest(found), nil
}

// shortest settles several worktrees answering for one place: the shortest
// branch takes it, its spelling breaking a tie, never git's listing order.
func shortest(found []Candidate) Candidate {
	return slices.MinFunc(found, func(a, b Candidate) int {
		return cmp.Or(cmp.Compare(len(a.branch), len(b.branch)), cmp.Compare(a.branch, b.branch))
	})
}

// Candidates lists what the repository offers to work on: every worktree git
// knows, then what each resolver offers that has none. A resolver that will not
// answer costs its own rows, never the worktrees.
func (e Env) Candidates() ([]Candidate, error) {
	var (
		open   []Candidate
		err    error
		offers = make([][]worktree.Place, len(e.Systems.Resolvers))
	)
	// No resolver reads another's answer.
	var wg sync.WaitGroup
	wg.Go(func() { open, err = e.Worktrees() })
	for i, r := range e.Systems.Resolvers {
		wg.Go(func() { offers[i], _ = r.Offer() })
	}
	wg.Wait()
	if err != nil {
		return nil, err
	}

	// Keyed on the name, which is what a place of any source is retyped as.
	out := slices.Clone(open)
	seen := make(map[string]int, len(out))
	for i, c := range out {
		seen[c.Name] = i
	}
	for i, places := range offers {
		r := e.Systems.Resolvers[i]
		for _, p := range places {
			if at, already := seen[p.Name]; already {
				// A resolver that read a branch alone may have no title; its own offer
				// completes one. Another's is another place spelled alike.
				if out[at].Source == r.Name() && out[at].Label == "" {
					out[at].Label = p.Label
				}
				continue
			}
			// An offer the core could not make a directory for could be shown but never
			// retyped.
			if checkName(p.Name) != nil {
				continue
			}
			seen[p.Name] = len(out)
			out = append(out, answered(r, Candidate{Place: p}))
		}
	}
	return out, nil
}

// path is where a worktree for a place would be created.
func (e Env) path(name string) string {
	return filepath.Join(e.Repo, e.Config.Worktree.Dir(), name)
}

// values are what a command for this worktree renders with: the worktree itself,
// then the resolver that answered for it, then the ambient sources. The first to
// supply a name owns it, and a source that will not answer costs only the values
// it would have supplied.
func (e Env) values(t worktree.Tree) worktree.Values {
	sources := e.Systems.Sources
	if s, ok := t.By.(worktree.Source); ok {
		sources = slices.Concat([]worktree.Source{s}, sources)
	}
	vals := worktree.Values{"Name": t.Name, "Dir": t.Path}
	for _, s := range sources {
		if supplied, err := s.Supply(t); err == nil {
			vals.Merge(supplied)
		}
	}
	return vals
}
