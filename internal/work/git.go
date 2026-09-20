package work

import (
	"fmt"
	"os"

	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/worktree"
)

// gitAlone is the core's own answer, behind every integration the settings
// named: a worktree no integration claims is described out of git alone.
type gitAlone struct {
	repo worktree.Repo
	from worktree.Path // the checkout work was invoked in, whose HEAD a new branch forks from
}

func (e Env) gitAlone() gitAlone { return gitAlone{repo: e.Repo, from: e.Dir} }

func (gitAlone) Name() worktree.IntegrationName { return worktree.GitSource }

// Icon marks a row that stands for nothing but itself.
func (gitAlone) Icon() string { return "◇" }

// Identify never answers for an identifier alone; a worktree is whatever is
// left, the directory settling it where both are in hand.
func (gitAlone) Identify(id worktree.ID, o worktree.Open) (worktree.Place, error) {
	if o.None() {
		return worktree.Place{}, fmt.Errorf("%w: %q is a name to add, not one to recognise", worktree.ErrUnknown, id)
	}
	if id != "" && !git.SameDir(o.Path, worktree.Path(id)) {
		return worktree.Place{}, fmt.Errorf("%w: the worktree at %s is not %s", worktree.ErrUnknown, o.Path, id)
	}
	return worktree.Place{ID: worktree.ID(o.Path), Name: o.Name(), Branch: o.Branch}, nil
}

// place is a name of the user's own taken at its word, there being nothing to
// recognise it against.
func (gitAlone) place(name worktree.Name) worktree.Place {
	return worktree.Place{ID: worktree.ID(name), Name: name, Branch: worktree.Branch(name)}
}

// Offer has nothing to offer: a name of the user's own is typed, never listed.
func (gitAlone) Offer() ([]worktree.Place, error) { return nil, nil }

// Prepare asserts the name is free, a branch already holding it being a worktree
// to re-enter rather than create.
func (r gitAlone) Prepare(p worktree.Place) (worktree.Place, error) {
	if git.HasBranch(r.repo, p.Branch) {
		return p, fmt.Errorf("branch %s already exists; enter its worktree with work %s", p.Branch, p.Name)
	}
	return p, nil
}

func (r gitAlone) Create(p worktree.Place, path worktree.Path) error {
	return git.NewWorktree(r.from, path, p.Branch)
}

// handback is the far-seam action for a worktree that opens on nothing else: it
// hands the worktree itself back.
type handback struct{}

func (handback) Name() worktree.IntegrationName { return worktree.GitSource }

func (handback) OnCreated(worktree.Tree) error { return nil }

func (handback) Open(t worktree.Tree) (worktree.Handoff, error) {
	// Nothing runs in the directory to fail on it, so a worktree git lists but
	// nobody can enter is refused here or nowhere.
	if _, err := os.Stat(string(t.Path)); err != nil {
		return worktree.Handoff{}, err
	}
	return worktree.Handoff{Dir: t.Path}, nil
}
