// Package beads resolves the tickets bd knows into places to work: it names the
// branch a ticket's worktree checks out, says which open worktree is whose,
// offers the tickets worth starting, refuses one that cannot be worked, and has
// bd make the worktree. Claiming the ticket is internal/action/beads.
package beads

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/JHK/work-cli/internal/beads"
	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/worktree"
)

// Name is what this system goes by on both seams.
const Name = "beads"

// Resolver answers for the repository's beads. It holds the listings it made, so
// one run asks bd for each of them at most once and the vetting reads the very
// snapshot that called a bead ready.
type Resolver struct {
	repo    string
	from    string // the checkout work was invoked in, whose HEAD a new branch forks from
	pattern config.Branch

	// allBeads is every bead bd knows, closed ones included; readyBeads is every
	// bead bd calls unblocked. Each is listed at most once per run.
	allBeads, readyBeads func() ([]beads.Bead, error)

	held sync.Map // id -> beads.Bead, the records already in hand
}

// New answers for the repository at repo, naming branches by the ticket pattern.
func New(repo, from string, branch config.Branch) *Resolver {
	return &Resolver{
		repo:       repo,
		from:       from,
		pattern:    branch,
		allBeads:   sync.OnceValues(func() ([]beads.Bead, error) { return beads.All(repo) }),
		readyBeads: sync.OnceValues(func() ([]beads.Bead, error) { return beads.Ready(repo) }),
	}
}

func (r *Resolver) Name() string { return Name }

// Icon marks a row that stands for a ticket.
func (r *Resolver) Icon() string { return "◆" }

// Identify names the bead behind what the core is holding.
//
// An identifier alone is taken at its word and bd is not asked, so this resolver
// answers last and takes whatever is left; whether bd knows the id is
// [Resolver.Prepare]'s question.
//
// A worktree is named by the longest id bd knows that owns its branch. A bd that
// will not list names none of them, and the worktree is a plain one until it does.
func (r *Resolver) Identify(id string, o worktree.Open) (worktree.Place, error) {
	if o.None() {
		return worktree.Place{ID: id, Name: id}, nil
	}
	// Ahead of the listing: confirming an identifier against a branch must not need bd.
	if id != "" && !r.pattern.Owns(id, o.Branch) {
		return worktree.Place{}, notMine(o)
	}

	found := r.longest(o.Branch)
	if id != "" {
		// A longer id bd knows owns this branch, so the branch is that ticket's.
		if len(found) > len(id) {
			return worktree.Place{}, notMine(o)
		}
		found = id
	}
	if found == "" {
		return worktree.Place{}, notMine(o)
	}
	b, _ := r.record(found)
	return worktree.Place{ID: found, Name: found, Branch: o.Branch, Label: b.Title}, nil
}

func notMine(o worktree.Open) error {
	return fmt.Errorf("%w: no bead is named by branch %q", worktree.ErrUnknown, o.Branch)
}

// Offer is what bd calls workable, which is what is worth starting.
func (r *Resolver) Offer() ([]worktree.Place, error) {
	list, err := r.readyBeads()
	if err != nil {
		return nil, err
	}
	out := make([]worktree.Place, 0, len(list))
	for _, b := range list {
		// Kept whole, so entering one of these vets it against the snapshot that called
		// it ready rather than asking bd again.
		r.held.Store(b.ID, b)
		out = append(out, worktree.Place{ID: b.ID, Name: b.ID, Label: b.Title})
	}
	return out, nil
}

// Prepare names the branch the ticket's worktree will check out, a slug of the
// title following the id, or says why the ticket may not be worked at all. It
// asks bd and nothing more.
func (r *Resolver) Prepare(p worktree.Place) (worktree.Place, error) {
	b, err := r.bead(p.ID)
	if err != nil {
		return p, err
	}
	if err := r.vet(b); err != nil {
		return p, err
	}
	p.Branch = r.pattern.Ticket(b.ID, slug(b.Title))
	p.Label = b.Title
	return p, nil
}

// Create has bd add the worktree, wired to the repository's shared database and
// forked from what the checkout work was invoked in has at HEAD.
func (r *Resolver) Create(p worktree.Place, path string) error {
	return beads.CreateWorktree(r.from, path, p.Branch)
}

// Supply is what this resolver tells a command about the worktree it resolved:
// the ticket's id and title.
func (r *Resolver) Supply(t worktree.Tree) (worktree.Values, error) {
	return worktree.Values{"ID": t.ID, "Title": t.Label}, nil
}

// vet reports why the bead cannot be worked. Only the last rule asks bd anything.
func (r *Resolver) vet(b beads.Bead) error {
	switch {
	case b.Status == "closed":
		return fmt.Errorf("%s is already closed", b.ID)
	// Refining an epic is what its worktree is for, and refining is what would
	// supply the rest.
	case b.Type == "epic":
		return nil
	case b.Status == "deferred":
		return fmt.Errorf("%s is unrefined; refine it first with /refine %s", b.ID, b.ID)
	// Ahead of the criteria check: /refine would move the bead out of the status
	// that is the actual blocker.
	case b.Status != "open" && b.Status != "in_progress":
		return fmt.Errorf("%s is %s, not workable", b.ID, b.Status)
	case strings.TrimSpace(b.AcceptanceCriteria) == "":
		return fmt.Errorf("%s has no acceptance criteria; refine it first with /refine %s", b.ID, b.ID)
	case b.Status == "open":
		ok, err := r.workable(b.ID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%s is blocked by an open dependency", b.ID)
		}
	}
	return nil
}

// workable asks whether bd lists the bead as unblocked.
func (r *Resolver) workable(id string) (bool, error) {
	list, err := r.readyBeads()
	if err != nil {
		return false, err
	}
	_, ok := find(list, id)
	return ok, nil
}

// bead is the record for an id: one already in hand, else the full listing's own,
// else bd asked for it outright, which is where a name bd does not know is refused.
func (r *Resolver) bead(id string) (beads.Bead, error) {
	if b, ok := r.held.Load(id); ok {
		return b.(beads.Bead), nil
	}
	if b, ok := r.record(id); ok {
		return b, nil
	}
	b, err := beads.Show(r.repo, id)
	if err != nil {
		return beads.Bead{}, err
	}
	r.held.Store(id, b)
	return b, nil
}

// record is the bead the full listing named, if it named one. A miss falls
// through to a query: a bd that will not list may still answer for one id.
func (r *Resolver) record(id string) (beads.Bead, bool) {
	list, err := r.allBeads()
	if err != nil {
		return beads.Bead{}, false
	}
	return find(list, id)
}

// longest names the bead a branch belongs to: the longest known id owning it, so
// that a branch of one-two's does not fall to one.
func (r *Resolver) longest(branch string) string {
	// Ahead of the listing: a detached worktree has no branch for any id to own.
	if branch == "" {
		return ""
	}
	list, err := r.allBeads()
	if err != nil {
		return ""
	}
	best := ""
	for _, b := range list {
		if len(b.ID) > len(best) && r.pattern.Owns(b.ID, branch) {
			best = b.ID
		}
	}
	return best
}

func find(list []beads.Bead, id string) (beads.Bead, bool) {
	for _, b := range list {
		if b.ID == id {
			return b, true
		}
	}
	return beads.Bead{}, false
}

const slugLen = 40

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slug(title string) string {
	s := nonSlug.ReplaceAllString(strings.ToLower(title), "-")
	s = strings.Trim(s, "-")
	if len(s) > slugLen {
		s = s[:slugLen]
	}
	return strings.Trim(s, "-")
}
