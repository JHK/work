package work

import (
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
	"github.com/stretchr/testify/require"
)

// No command reaches this: no action that ships renders a value, so the seam is
// what the case can ask.
func TestBothOfAnActionsMomentsReadOneSetOfValues(t *testing.T) {
	repo := worktree.Repo(testenv.InitRepo(t))
	by := &supplier{repo: repo}
	a := &keeper{}
	e := Env{
		Repo:         repo,
		Config:       config.Default(),
		Integrations: Integrations{Actions: []Action{a}},
	}
	place := worktree.Place{ID: "bd-1", Name: "bd-1", Label: "a title"}

	_, err := e.Enter(answered(by, Candidate{Place: place}), Options{})

	require.NoError(t, err, "the worktree was refused")
	testenv.Equal(t, a.opened, a.created, "the two moments read values of their own")
	require.Equal(t, 1, by.asks, "the values were assembled more than once")
}

// supplier stands at the near seam: it makes the worktree, and supplies the one
// value the core does not hold.
type supplier struct {
	repo worktree.Repo
	asks int
}

func (*supplier) Name() worktree.IntegrationName { return "supplier" }

func (*supplier) Icon() string { return "s" }

func (*supplier) Identify(worktree.ID, worktree.Open) (worktree.Place, error) {
	return worktree.Place{}, worktree.ErrUnknown
}

func (*supplier) Offer() ([]worktree.Place, error) { return nil, nil }

func (*supplier) Prepare(p worktree.Place) (worktree.Place, error) {
	p.Branch = worktree.Branch(p.Name)
	return p, nil
}

func (s *supplier) Create(p worktree.Place, path worktree.Path) error {
	return git.NewWorktree(worktree.Path(s.repo), path, p.Branch)
}

func (s *supplier) Supply(t worktree.Tree) (worktree.Values, error) {
	s.asks++
	return worktree.Values{worktree.SubjectValue: string(t.ID) + ": " + string(t.Label)}, nil
}

// keeper stands at the far seam as the agent, which is what a creation opens on,
// and keeps what each of its two moments read.
type keeper struct{ created, opened worktree.Values }

func (*keeper) Name() worktree.IntegrationName { return config.ClaudeIntegration }

func (k *keeper) OnCreated(t worktree.Tree) error {
	k.created = t.Values
	return nil
}

func (k *keeper) Open(t worktree.Tree) (worktree.Handoff, error) {
	k.opened = t.Values
	return worktree.Handoff{Dir: t.Path}, nil
}
