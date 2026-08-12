// Package plain resolves a worktree that stands for nothing but itself: one asked
// for by name outright, or one discovered under a branch no other resolver answers
// for. It speaks to git and to no system beyond it, so it has no ticket to vet, no
// title to name a branch from and no values to supply.
package plain

import (
	"fmt"
	"path/filepath"

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
// identifier at all: a name typed at a verb is something to recognise, and a
// chain that made a branch of whatever it did not recognise would make one of
// every typo. [Named] is the same resolver where the name is the user's own.
func New(repo, from string) Resolver { return Resolver{repo: repo, from: from} }

// Named is the resolver a verb reaches by name: it takes the name at its word,
// there being nothing to recognise it against.
func Named(repo, from string) Resolver {
	r := New(repo, from)
	r.own = true
	return r
}

func (Resolver) Name() string { return Name }

// Icon marks a row that stands for nothing but itself: the ticket's mark, hollow.
func (Resolver) Icon() string { return "◇" }

// Identify names the place behind what the core is holding.
//
// An identifier alone is the verb's own resolver being handed a name: it is taken at
// its word, the branch spelled exactly as the name is, nothing guessed and nothing
// asked. In the chain there is no such name — an identifier nothing recognised is one
// nothing answers for, and what a system would have answered for is the settings
// leaving that system out.
//
// A worktree is whatever is left, so that one nothing else recognises is still one to
// reach; a detached worktree is named by its directory, having no branch to be named
// by. With an identifier as well it is the directory that settles it, a plain
// worktree having nothing behind it to be named by and so being only itself.
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
	name := o.Branch
	if name == "" {
		name = filepath.Base(o.Path)
	}
	return worktree.Place{ID: o.Path, Name: name, Branch: o.Branch}, nil
}

// Offer has nothing to offer: a name of the user's own is typed, never listed.
func (r Resolver) Offer() ([]worktree.Place, error) { return nil, nil }

// Prepare asserts the name is free, a branch already holding it being a worktree
// to re-enter rather than create.
func (r Resolver) Prepare(place worktree.Place) (worktree.Place, error) {
	if git.HasBranch(r.repo, place.Branch) {
		return place, fmt.Errorf("branch %s already exists; enter its worktree with work %s", place.Branch, place.Name)
	}
	return place, nil
}

// Create forks a branch of its own from what the checkout work was invoked in
// has at HEAD.
func (r Resolver) Create(place worktree.Place, path string) error {
	return git.NewWorktree(r.from, path, place.Branch)
}
