// Package plain resolves a worktree that stands for nothing but itself: one asked
// for by name outright, or one discovered under a branch no other resolver answers
// for. It speaks to git and to no system beyond it.
package plain

import (
	"fmt"

	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/worktree"
)

// Name is what this system goes by.
const Name = "plain"

// Resolver answers for the worktrees nothing else claims.
type Resolver struct {
	repo string
	from string // the checkout work was invoked in, whose HEAD a new branch forks from
	own  bool   // the verb's own resolver, handed a name rather than an identifier
}

// New answers for the worktrees the resolvers ahead of it left, and for no
// identifier at all. [Named] is the same resolver where the name is the user's own.
func New(repo, from string) Resolver { return Resolver{repo: repo, from: from} }

// Named is the resolver a verb reaches by name: it takes the name at its word,
// there being nothing to recognise it against.
func Named(repo, from string) Resolver {
	r := New(repo, from)
	r.own = true
	return r
}

func (Resolver) Name() string { return Name }

// Icon marks a row that stands for nothing but itself.
func (Resolver) Icon() string { return "◇" }

// Identify takes an identifier at its word, which only [Named] is handed one; a
// worktree is whatever is left, the directory settling it where both are in hand.
func (r Resolver) Identify(id string, o worktree.Open) (worktree.Place, error) {
	if o.None() {
		if !r.own {
			return worktree.Place{}, fmt.Errorf("%w: %q is a name to add, not one to recognise", worktree.ErrUnknown, id)
		}
		return worktree.Place{ID: id, Name: id, Branch: id}, nil
	}
	if id != "" && !git.SameDir(o.Path, id) {
		return worktree.Place{}, fmt.Errorf("%w: the worktree at %s is not %s", worktree.ErrUnknown, o.Path, id)
	}
	return worktree.Place{ID: o.Path, Name: o.Name(), Branch: o.Branch}, nil
}

// Offer has nothing to offer: a name of the user's own is typed, never listed.
func (r Resolver) Offer() ([]worktree.Place, error) { return nil, nil }

// Prepare asserts the name is free, a branch already holding it being a worktree
// to re-enter rather than create.
func (r Resolver) Prepare(p worktree.Place) (worktree.Place, error) {
	if git.HasBranch(r.repo, p.Branch) {
		return p, fmt.Errorf("branch %s already exists; enter its worktree with work %s", p.Branch, p.Name)
	}
	return p, nil
}

// Create forks a branch of its own from what the checkout work was invoked in
// has at HEAD.
func (r Resolver) Create(p worktree.Place, path string) error {
	return git.NewWorktree(r.from, path, p.Branch)
}
