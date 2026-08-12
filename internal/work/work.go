// Package work is the core beneath every front end. It knows worktrees and the
// sequence: ask the resolvers what an identifier is, make the worktree if it is
// not there, hand it to an action. A ticket, a pull request, an agent, an editor
// and mise all sit behind the two seams it declares here.
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
// its candidates, and creates the worktree. Everything a tracker or a forge knows is
// behind it.
type Resolver interface {
	worktree.System

	// Identify names the place behind what the core is holding: an identifier a person
	// typed, or a worktree the repository has open. A resolver reads which by
	// [worktree.Open.None]. Where the core holds a worktree it may hold an identifier
	// as well, which is one to confirm rather than to read: the question is then
	// whether that worktree is that place's, and answering it must not need what
	// naming the worktree from its branch alone would.
	//
	// It asks as little as it can, because every reading goes through here: a tracker
	// that cannot be reached must not cost a worktree that is already open, and an
	// identifier is read into a place rather than checked against the system behind
	// it, which is [Resolver.Prepare]'s and runs only where a worktree is about to be
	// made.
	//
	// [worktree.ErrUnknown] means this resolver does not answer for what it was shown,
	// and the next one is asked. Whether a system that cannot be reached counts as not
	// recognising something is the resolver's call; the core sees the one bit either
	// way. Every other error stops the run.
	Identify(id string, o worktree.Open) (worktree.Place, error)

	// Offer lists the places worth starting, whether or not they have a worktree.
	Offer() ([]worktree.Place, error)

	// Prepare completes a place a worktree is about to be made for, naming the
	// branch, or says why no worktree may be made for it. Both are free of
	// consequence, so the core runs it ahead of any screen: a place that cannot be
	// worked is refused rather than asked about.
	Prepare(p worktree.Place) (worktree.Place, error)

	// Create checks the place's branch out into a worktree at path.
	Create(p worktree.Place, path string) error
}

// Action is the far seam, in the half that runs because a worktree came into
// being: trusting its mise config, claiming the ticket it was made for. Every one
// of them runs, in order, and none of them runs on the way back into a worktree
// that was already there.
type Action interface {
	worktree.System
	Run(t worktree.Tree) error
}

// Opener is the far seam's other half: the one action a worktree opens on. It
// renders the command work replaces itself with, and exactly one per run does.
type Opener interface {
	worktree.System

	// Applies says whether the values gathered so far leave this a command to run,
	// returning [worktree.ErrAbsent] where they do not, which [worktree.Absent]
	// carries on a refusal of the action's own. It is asked before anything
	// is created, so what it refuses is left off the screen and refused outright
	// where a flag named it. Every other error stops the run, whether the screen or
	// a flag reached this action.
	Applies(vals worktree.Values) error

	// Open renders the handoff from the values every source supplied. Rendering is
	// free of consequence, so it runs after the worktree exists and nothing depends
	// on it having run.
	//
	// The values are the core's, gathered once and shown to every opener: an action
	// with a name of its own to place renders against a copy rather than writing into
	// what it was handed.
	Open(t worktree.Tree, vals worktree.Values) (worktree.Handoff, error)
}

// Systems are the implementations behind the seams, in the order they are asked.
// cmd/work is the only place they are named.
type Systems struct {
	Resolvers []Resolver
	Actions   []Action
	Openers   []Opener

	// Sources are asked what they know about a worktree, whichever resolver answered
	// for it. The resolver that did is asked first and separately, its answer
	// describing the one place it resolved; these describe any worktree, the
	// environment and git among them.
	Sources []worktree.Source

	// Named is the one resolver a verb reaches by name rather than through the
	// chain: add's, which is handed a name of the user's own rather than an
	// identifier to recognise. A verb that says which system it means is the one
	// thing a chain cannot express.
	Named Resolver

	// Disabled are the systems the settings left out, in the order they would have
	// been asked in. Nothing is wired to them and nothing is asked of them beyond one
	// question, and only of those that can answer it at all: whether an identifier
	// nothing here answered for is spelled as one of theirs, so that a system
	// switched off reads as switched off rather than as work not knowing what it was
	// given.
	Disabled []worktree.System
}

// Wiring names the systems for a repository once its settings are read.
// checkout is where work was invoked, whose HEAD a new branch forks from; the
// worktree itself still lands under repo.
//
// A front end settles what it offers before it knows either, so it asks with
// neither and reads the names and flags off what comes back. Wiring a system is
// therefore naming it: what a system reaches for, it reaches for when asked a
// question and not when it is built.
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
// traverse, and may not open with a dash. It is the core's rule because the core
// owns the paths, whatever a resolver would like to call a place.
var worktreeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// checkName holds a name to the worktree-name rule, and is how breaking it reads
// wherever it is broken.
func checkName(name string) error {
	if !worktreeName.MatchString(name) {
		return fmt.Errorf("%q is not a usable worktree name", name)
	}
	return nil
}

// Candidate is one place to work, whether it was offered or asked for by name:
// the place a resolver made of it, the worktree it already has, and the resolver
// to go back to for the rest of what only that resolver knows.
type Candidate struct {
	worktree.Place
	Open bool   // a worktree for it already exists
	Icon string // the mark that resolver draws its rows with, empty where it named none

	by     Resolver
	path   string // where that worktree sits
	branch string // what it has checked out, empty when it is detached
}

// Resolve maps an identifier to the place it names and to the worktree that place
// already has. One listing serves both readings: the name is read off the open
// worktrees where one answers to it, and the resolvers are asked where none does.
func (e Env) Resolve(arg string) (Candidate, error) {
	if arg == "" {
		return Candidate{}, errors.New("no target given")
	}
	open, err := e.Worktrees()
	if err != nil {
		return Candidate{}, err
	}
	// A worktree already listed under the name is that worktree, whatever a
	// resolver would make of the name, so that what the picker and the completion
	// show is entered by retyping it. Several listed under the one name go the way
	// locate settles them, so naming a place and locating it land in the same one.
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
	// A place no worktree could be made for is one nothing really answered for, so it
	// is handed over as such.
	if err := checkName(p.Name); err != nil {
		return Candidate{}, e.switchedOff(arg, err)
	}
	return e.locate(r, p, open)
}

// switchedOff is the refusal an identifier gets where a system the settings left
// out is what would have answered for it: not that work cannot read what it was
// given, but that nothing is wired to read it.
//
// It is reached only where nothing usable came back, which is two things. A chain
// that ran out is one, and a resolver that recognised the identifier and then
// failed is not: what that one said is what the run stops on. The other is a
// resolver answering for whatever is left masking one that is off, so the answer
// the chain did give is judged too.
//
// The question is [worktree.Claimant] and never [Resolver.Identify], which is
// contracted for recognition inside the chain, where the last resolver takes
// whatever is left: a system asked that one off the chain answers for identifiers
// that are no more its than anyone's, and a typo would be attributed to it. What a
// system does claim is put through the name rule as well, so a key is named only
// where turning that system back on would have made a place.
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

// off is how a system the settings left out reads wherever one is reached: the
// name it goes by and the key that puts it back, worded once for both seams.
func off(name string) string {
	return fmt.Sprintf("%s is off, put it back with %s = true", name, config.SystemKey(name))
}

// answered stamps a candidate with the resolver that answered for it: the name
// it is sourced to, the mark a screen draws it with, and the resolver itself for
// the rest of what only that resolver knows. The name is the core's to put on,
// being the name it asked under: a resolver restating it would be a second
// spelling of one thing, and one nothing here checks.
func answered(r Resolver, c Candidate) Candidate {
	c.Source, c.by = r.Name(), r
	if d, ok := r.(worktree.Drawn); ok {
		c.Icon = d.Icon()
	}
	return c
}

// identify is the chain: the first resolver to answer for what the core is holding
// owns it, and one that does not recognise it passes it on. Every reading goes
// through here, so unrecognition means the one thing at every call site.
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
	// The chain ran out, which a worktree never does: the last resolver answers for
	// whatever is left, so nothing recognising one is the listing failing rather than
	// a system being off.
	if o.None() {
		return nil, worktree.Place{}, e.switchedOff(id, fmt.Errorf("nothing answers for %q", id))
	}
	return nil, worktree.Place{}, fmt.Errorf("nothing answers for the worktree at %s", o.Path)
}

// Add is the place a name of the user's own makes: a worktree with no ticket and
// no pull request behind it, on a branch spelled exactly as the name is. It names
// its resolver rather than putting the name through the chain, which would read
// it as a ticket id.
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

// Worktrees is every worktree the repository has and nothing besides, the main
// checkout excepted, each under the place the first resolver to answer for it
// says it stands for: what is already there to reach rather than what could be
// started. It is the one listing both readings start from.
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

// adopt is the place an open worktree stands for, from the first resolver that
// answers for it. The last resolver in the chain answers for whatever is left, so
// a worktree nothing recognises is still a worktree to reach.
func (e Env) adopt(o worktree.Open) (Candidate, error) {
	r, p, err := e.identify("", o)
	if err != nil {
		return Candidate{}, err
	}
	return answered(r, Candidate{Place: p, Open: true, path: o.Path, branch: o.Branch}), nil
}

// locate is the worktree a resolved place already has: the resolver that named the
// place, asked of each worktree the repository has open, whether it is that place's.
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
		// The place as the resolver describes it on that worktree, which knows the branch
		// it found the place by rather than the one the place would render today.
		found = append(found, answered(r, Candidate{Place: its, Open: true, path: c.path, branch: c.branch}))
	}
	if len(found) == 0 {
		return answered(r, Candidate{Place: p}), nil
	}
	return shortest(found), nil
}

// shortest settles several worktrees answering for one place, which is a repository
// arranged by hand: the shortest branch takes it, its spelling breaking a tie, so it
// is settled on the branch and never on git's listing order.
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
	// No resolver reads another's answer, so the offers are issued at once and the
	// wait is the slowest of them rather than their sum.
	var wg sync.WaitGroup
	wg.Go(func() { open, err = e.Worktrees() })
	for i, r := range e.Systems.Resolvers {
		wg.Go(func() { offers[i], _ = r.Offer() })
	}
	wg.Wait()
	if err != nil {
		return nil, err
	}

	// Keyed on the name, which is what a place of any source is retyped as, so one
	// place to work is offered once however it was found.
	out := slices.Clone(open)
	seen := make(map[string]int, len(out))
	for i, c := range out {
		seen[c.Name] = i
	}
	for i, places := range offers {
		r := e.Systems.Resolvers[i]
		for _, p := range places {
			if at, already := seen[p.Name]; already {
				// Naming a worktree asks as little as it can, so a resolver reading a branch
				// alone may have no title for the row; the offer beside it is what completes
				// it, and the dedup would otherwise drop the title along with the row that
				// carried it. Only the resolver that answered for the worktree completes it:
				// another's offer of the same name is another place spelled alike.
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
// then whatever the resolver that answered for it knows, then whatever the ambient
// sources know. The first to supply a name owns it, so the resolver's account of the
// place it resolved is never overwritten by a source that answers for any worktree.
//
// A source that will not answer costs the values it would have supplied and nothing
// else: what that leaves unrunnable is the command that needed them, which is where
// the refusal belongs.
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
