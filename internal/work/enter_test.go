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
func TestBothOfAnActionsMomentsReadOneSetOfValues(t *testing.T) {
	by := &supplier{}
	a := &keeper{}
	e := Env{
		Repo:    worktree.Repo(testenv.InitRepo(t)),
		Config:  config.Default(),
		Systems: Systems{Actions: []Action{a}, Handback: a},
	}
	place := worktree.Place{ID: "bd-1", Name: "bd-1", Label: "a title"}

	_, err := e.Enter(answered(by, Candidate{Place: place}), Options{Verb: "add"})

	require.NoError(t, err, "the worktree was refused")
	testenv.Equal(t, a.opened, a.created, "the two moments read values of their own")
	require.Equal(t, 1, by.asks, "the values were assembled more than once")
}

// supplier stands at the near seam: it makes the worktree a directory, and
// supplies the one value the core does not hold.
type supplier struct{ asks int }

func (*supplier) Name() worktree.SystemName { return "supplier" }

func (*supplier) Icon() string { return "s" }

func (*supplier) Identify(worktree.ID, worktree.Open) (worktree.Place, error) {
	return worktree.Place{}, worktree.ErrUnknown
}

func (*supplier) Offer() ([]worktree.Place, error) { return nil, nil }

func (*supplier) Prepare(p worktree.Place) (worktree.Place, error) {
	p.Branch = worktree.Branch(p.Name)
	return p, nil
}

func (*supplier) Create(_ worktree.Place, path worktree.Path) error {
	return os.MkdirAll(string(path), 0o755)
}

func (s *supplier) Supply(t worktree.Tree) (worktree.Values, error) {
	s.asks++
	return worktree.Values{worktree.SubjectValue: string(t.ID) + ": " + string(t.Label)}, nil
}

// keeper stands at the far seam and keeps what each of its two moments read.
type keeper struct{ created, opened worktree.Values }

func (*keeper) Name() worktree.SystemName { return "keeper" }

func (k *keeper) OnCreated(t worktree.Tree) error {
	k.created = t.Values
	return nil
}

func (k *keeper) Open(t worktree.Tree) (worktree.Handoff, error) {
	k.opened = t.Values
	return worktree.Handoff{Dir: t.Path}, nil
}
