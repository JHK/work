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

	// Offer lists the places worth starting, whether or not they have a worktree. A
	// refusal costs the offered rows and nothing else.
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

	// Open renders the handoff from the values gathered for the worktree. The values
	// are the core's account of the worktree, so an action with a name of its own
	// to place renders against a copy.
	Open(t worktree.Tree, vals worktree.Values) (worktree.Handoff, error)
}

// Systems are the implementations behind the seams, in the order they are asked.
type Systems struct {
	Resolvers []Resolver
	Actions   []Action
	Openers   []Opener

	// Named is the chain's tail for an identifier nothing answered for: the resolver
	// add hands a name of the user's own.
	Named Resolver
}

// Wiring names the systems for a repository once its settings are read.
// checkout is where work was invoked, whose HEAD a new branch forks from; the
// worktree itself still lands under repo. A front end asks with neither and
// reads the names and flags alone off what comes back, so a system wired here
// reaches for nothing until it is asked a question.
type Wiring func(repo, checkout string, cfg config.Config) Systems

// Env is the repository work operates on, the directory it was invoked in, the
// settings it reads, and the systems behind its seams.
type Env struct {
	Repo string

	// Dir is where work was invoked, which is what standing in a worktree is
	// judged against. It is absolute, an empty one standing nowhere.
	Dir string

	Config  config.Config
	Systems Systems
}

// Open finds the repository containing dir and wires the systems cfg asks for.
func Open(dir string, cfg config.Config, wire Wiring) (Env, error) {
	repo, err := git.Root(dir)
	if err != nil {
		return Env{}, err
	}
	here, err := filepath.Abs(dir)
	if err != nil {
		return Env{}, err
	}
	return Env{Repo: repo, Dir: here, Config: cfg, Systems: wire(repo, dir, cfg)}, nil
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

// Dir is what the worktree's directory is called, empty where the candidate has
// none. A branch may be spelled with separators in it; a directory never is.
func (c Candidate) Dir() string {
	if c.path == "" {
		return ""
	}
	return filepath.Base(c.path)
}

// actOn is why a verb may not act on the worktree a candidate has: there is
// none, it is the repository itself, or the process stands inside the one git is
// about to move or take away, which would leave the shell in a directory that is
// gone.
func (e Env) actOn(c Candidate, verb string) error {
	if !c.Open {
		return fmt.Errorf("%s has no worktree to %s", c.Name, verb)
	}
	if e.mainCheckout(c) {
		return fmt.Errorf("%s is the main checkout; work %s acts on a worktree under it", c.Name, verb)
	}
	if e.holds(c) {
		return fmt.Errorf("%s is the worktree you are standing in; run work %s from outside it", c.Name, verb)
	}
	return nil
}

// mainCheckout reports whether a candidate is where the repository itself sits.
func (e Env) mainCheckout(c Candidate) bool { return git.SameDir(c.path, e.Repo) }

// holds reports whether [Env.Dir] sits within the worktree a candidate has, a
// worktree nested in that one counting as within it.
func (e Env) holds(c Candidate) bool { return e.Dir != "" && git.Inside(e.Dir, c.path) }

// stoodIn is where the worktree the shell is in sits: the innermost of the ones
// holding [Env.Dir], and empty where it stands outside them all.
func (e Env) stoodIn(open []Candidate) string {
	found := ""
	for _, c := range open {
		if e.holds(c) && (found == "" || git.Inside(c.path, found)) {
			found = c.path
		}
	}
	return found
}

// lessStoodIn drops the row for the worktree at here, an empty here naming none.
func lessStoodIn(rows []Candidate, here string) []Candidate {
	if here == "" {
		return rows
	}
	return slices.DeleteFunc(rows, func(c Candidate) bool { return c.path == here })
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
	c, err := e.place(arg, open)
	if errors.Is(err, errUnanswered) {
		return Candidate{}, fmt.Errorf("%w %q", errUnanswered, arg)
	}
	return c, err
}

// Unanswered reports whether a refusal is an identifier no system answered for.
func Unanswered(err error) bool { return errors.Is(err, errUnanswered) }

// Nameable reports whether a name is one a worktree could be made for.
func Nameable(name string) bool { return checkName(name) == nil }

// place is the candidate an identifier names against the worktrees open. A name
// nothing answered for comes back wrapping [errUnanswered].
func (e Env) place(id string, open []Candidate) (Candidate, error) {
	// A worktree listed under the name is that worktree, whatever a resolver would
	// make of it.
	if named := byName(open, id); len(named) > 0 {
		return shortest(named), nil
	}
	r, p, err := e.identify(id, worktree.Open{})
	if err != nil {
		return Candidate{}, err
	}
	// A place no worktree could be made for is one nothing really answered for.
	if err := checkName(p.Name); err != nil {
		return Candidate{}, err
	}
	return e.locate(r, p, open)
}

// byName is the worktrees listed under an identifier.
func byName(open []Candidate, name string) []Candidate {
	var found []Candidate
	for _, c := range open {
		if c.Name == name {
			found = append(found, c)
		}
	}
	return found
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

// errUnanswered means the chain ran out with the identifier unrecognised, which
// is a name to add rather than a place to reach.
var errUnanswered = errors.New("nothing answers for")

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
		return nil, worktree.Place{}, errUnanswered
	}
	return nil, worktree.Place{}, fmt.Errorf("nothing answers for the worktree at %s", o.Path)
}

// Add is the place a worktree is about to be made for: what a system answers for
// the identifier with, else a name of the user's own. A place that already has a
// worktree is refused.
func (e Env) Add(id string) (Candidate, error) {
	open, err := e.Worktrees()
	if err != nil {
		return Candidate{}, err
	}
	c, err := e.place(id, open)
	if errors.Is(err, errUnanswered) {
		return e.invent(id)
	}
	if err != nil {
		return Candidate{}, err
	}
	if c.Open {
		return Candidate{}, fmt.Errorf("%s already has a worktree; enter it with work switch %s", c.Name, c.Name)
	}
	return c, nil
}

// invent is the place a name of the user's own makes: no ticket and no pull
// request behind it, on a branch spelled exactly as the name is.
func (e Env) invent(name string) (Candidate, error) {
	if err := checkName(name); err != nil {
		return Candidate{}, err
	}
	p, err := e.Systems.Named.Identify(name, worktree.Open{})
	if err != nil {
		return Candidate{}, err
	}
	return answered(e.Systems.Named, Candidate{Place: p}), nil
}

// Switchable is why a candidate cannot be entered: it has no worktree open.
func (e Env) Switchable(c Candidate) error {
	if c.Open {
		return nil
	}
	return fmt.Errorf("%s has no worktree open; work add %s makes one", c.Name, c.Name)
}

// Worktrees is every worktree the repository has with a working tree to reach,
// in git's order, each under the place the first resolver to answer for it says
// it stands for.
func (e Env) Worktrees() ([]Candidate, error) {
	list, err := e.listing()
	if err != nil {
		return nil, err
	}
	out := make([]Candidate, 0, len(list))
	for _, w := range list {
		c, err := e.adopt(worktree.Open{Path: w.Path, Branch: w.Branch})
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// Enterable is what entering offers: every worktree open but the one the shell
// already stands in.
func (e Env) Enterable() ([]Candidate, error) {
	open, err := e.Worktrees()
	if err != nil {
		return nil, err
	}
	return lessStoodIn(open, e.stoodIn(open)), nil
}

// Removable is what removing and moving offer: every worktree open that
// [Env.actOn] does not refuse, so the listing and the refusal are one rule.
func (e Env) Removable() ([]Candidate, error) {
	open, err := e.Worktrees()
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(open, func(c Candidate) bool { return e.actOn(c, "remove") != nil }), nil
}

// listing is the worktrees git reports, less the row a bare repository lists for
// itself, and none at all where there is no repository for git to report on.
func (e Env) listing() ([]git.Worktree, error) {
	if e.Repo == "" {
		return nil, nil
	}
	list, err := git.Worktrees(e.Repo)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(list, func(w git.Worktree) bool { return w.Bare }), nil
}

// Branches is what the repository's worktrees have checked out, in git's order:
// a detached one under its directory. It asks git and nothing else, so a listing
// costs no tracker and no forge.
func (e Env) Branches() ([]string, error) {
	list, err := e.listing()
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

// shortest settles several worktrees answering for one place: a branch takes it
// from a detached worktree, which goes by its directory rather than by an
// address of its own, then the shortest branch, its spelling breaking a tie,
// never git's listing order.
func shortest(found []Candidate) Candidate {
	return slices.MinFunc(found, func(a, b Candidate) int {
		return cmp.Or(
			cmp.Compare(detached(a), detached(b)),
			cmp.Compare(len(a.branch), len(b.branch)),
			cmp.Compare(a.branch, b.branch),
		)
	})
}

// detached orders a worktree with no branch behind every worktree that has one.
func detached(c Candidate) int {
	if c.branch == "" {
		return 1
	}
	return 0
}

// Candidates lists what the repository offers to work on: every worktree
// [Env.Enterable] offers, then what each resolver offers that has none. A
// resolver that will not answer costs its own rows, never the worktrees, and its
// refusal comes back beside them for the front end to say.
func (e Env) Candidates() ([]Candidate, []error, error) {
	var (
		open    []Candidate
		err     error
		offers  = make([][]worktree.Place, len(e.Systems.Resolvers))
		refused = make([]error, len(e.Systems.Resolvers))
	)
	// No resolver reads another's answer.
	var wg sync.WaitGroup
	wg.Go(func() { open, err = e.Worktrees() })
	for i, r := range e.Systems.Resolvers {
		wg.Go(func() { offers[i], refused[i] = r.Offer() })
	}
	wg.Wait()
	if err != nil {
		return nil, nil, err
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
	// After the merge, or a resolver offering that place puts it back as a row with
	// no worktree.
	out = lessStoodIn(out, e.stoodIn(open))
	return out, slices.DeleteFunc(refused, func(err error) bool { return err == nil }), nil
}

// Addable is what a worktree can be made for: the places the resolvers offer
// that have none yet.
func (e Env) Addable() ([]Candidate, []error, error) {
	list, refused, err := e.Candidates()
	if err != nil {
		return nil, nil, err
	}
	return slices.DeleteFunc(list, func(c Candidate) bool { return c.Open }), refused, nil
}

// path is where a worktree for a place would be created.
func (e Env) path(name string) string {
	return filepath.Join(e.Repo, e.Config.Worktree.Dir(), name)
}

// inside refuses a worktree directory leading out of the repository. The setting
// is read as a path; a symlink standing where it names is not.
func (e Env) inside() error {
	dir := e.Config.Worktree.Dir()
	if contains(e.Repo, filepath.Join(e.Repo, dir)) {
		return nil
	}
	return fmt.Errorf("%q resolves outside the repository", dir)
}

// contains reports whether path sits below root once symlinks are resolved. A
// path not on disk yet cannot lead anywhere, and passes.
func contains(root, path string) bool {
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		return true
	}
	if root, err = filepath.EvalSymlinks(root); err != nil {
		return true
	}
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != "." && filepath.IsLocal(rel)
}

// values are what a command for this worktree renders with: the worktree itself,
// then the resolver that answered for it.
func (e Env) values(t worktree.Tree) worktree.Values {
	vals := worktree.Values{"Name": t.Name, "Dir": t.Path}
	if s, ok := t.By.(worktree.Source); ok {
		if supplied, err := s.Supply(t); err == nil {
			vals.Merge(supplied)
		}
	}
	return vals
}
