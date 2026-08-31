package work

import (
	"os"
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
	"github.com/stretchr/testify/require"
)

// No command reaches this: no action that ships renders a value, so the seam is
// what the case can ask.
func TestTheActionsAndTheOpenerReadOneSetOfValues(t *testing.T) {
	by := &supplier{}
	action, opener := &keeper{}, &keeper{}
	e := Env{
		Repo:    testenv.InitRepo(t),
		Config:  config.Default(),
		Systems: Systems{Actions: []Action{action}, Openers: []Opener{opener}},
	}
	place := worktree.Place{ID: "bd-1", Name: "bd-1", Label: "a title"}

	_, err := e.Enter(answered(by, Candidate{Place: place}), Options{Verb: "add"})

	require.NoError(t, err, "the worktree was refused")
	testenv.Equal(t, opener.values, action.values, "the action read values of its own")
	require.Equal(t, 1, by.asks, "the values were assembled more than once")
}

// supplier stands at the near seam: it makes the worktree a directory, and
// supplies the one value the core does not hold.
type supplier struct{ asks int }

func (*supplier) Name() string { return "supplier" }

func (*supplier) Icon() string { return "s" }

func (*supplier) Identify(string, worktree.Open) (worktree.Place, error) {
	return worktree.Place{}, worktree.ErrUnknown
}

func (*supplier) Offer() ([]worktree.Place, error) { return nil, nil }

func (*supplier) Prepare(p worktree.Place) (worktree.Place, error) {
	p.Branch = p.Name
	return p, nil
}

func (*supplier) Create(_ worktree.Place, path string) error { return os.MkdirAll(path, 0o755) }

func (s *supplier) Supply(t worktree.Tree) (worktree.Values, error) {
	s.asks++
	return worktree.Values{worktree.SubjectValue: t.ID + ": " + t.Label}, nil
}

// keeper stands at both halves of the far seam: it is an action and an opener.
type keeper struct{ values worktree.Values }

func (*keeper) Name() string { return config.ShellOpener }

func (k *keeper) Run(t worktree.Tree) error {
	k.values = t.Values
	return nil
}

func (k *keeper) Open(t worktree.Tree) (worktree.Handoff, error) {
	k.values = t.Values
	return worktree.Handoff{Dir: t.Path}, nil
}
