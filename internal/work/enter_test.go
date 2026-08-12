package work

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
)

// opening is the wiring of [env] with two openers a key or a flag can name, and
// with each moment's own key named rather than left to the first opener.
func opening(t *testing.T, s *steps, create, enter string) (Env, *resolver) {
	t.Helper()
	e, r := env(t, s,
		&opener{steps: s, name: "first"},
		&opener{steps: s, name: "second"},
	)
	e.Config.Action = config.Action{CreateName: config.ActionName(create), EnterName: config.ActionName(enter)}
	return e, r
}

// adopt puts a worktree on the branch the resolver answers for, so that entering
// it is a re-entry rather than a creation.
func adopt(t *testing.T, e Env, r *resolver) Candidate {
	t.Helper()
	testenv.Git(t, e.Repo, "worktree", "add", "-b", r.adopts, filepath.Join(e.Repo, defaultDir, "a"))
	c, err := e.Resolve(adopted)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !c.Open {
		t.Fatalf("Resolve = %+v; want the worktree already there", c.Place)
	}
	return c
}

// Which key names the action is the moment's own: action.create for a worktree
// about to be made, action.enter for one already there.
func TestTheMomentDecidesWhichKeyNamesTheAction(t *testing.T) {
	var s steps
	e, r := opening(t, &s, "second", "first")

	fresh, err := e.Resolve("x")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if _, err := e.Enter(fresh, Options{}); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if !slices.Contains(s.seen, "open second") {
		t.Errorf("a fresh worktree ran %q; want the action action.create names", s.seen)
	}

	s.seen = nil
	if _, err := e.Enter(adopt(t, e, r), Options{}); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if !slices.Equal(s.seen, []string{"open first"}) {
		t.Errorf("a worktree already there ran %q; want the action action.enter names, alone", s.seen)
	}
}

// What a repository that configured nothing opens on, with no key naming an
// action and no flag given: a shell either way, work being a cd first. The
// systems are what a repository asks for, so neither moment may fall to one of
// them.
func TestTheDefaultActionsAreReachedWithNoSystemOn(t *testing.T) {
	var s steps
	shell := string(config.ActionShell)
	e, r := opening(t, &s, "", "")
	e.Systems.Openers = []Opener{&opener{steps: &s, name: shell}}

	fresh, err := e.Resolve("x")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if _, err := e.Enter(fresh, Options{}); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if !slices.Contains(s.seen, "open "+shell) {
		t.Errorf("a worktree just created ran %q; want the action.create a repository configures nothing for", s.seen)
	}

	s.seen = nil
	if _, err := e.Enter(adopt(t, e, r), Options{}); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if !slices.Equal(s.seen, []string{"open " + shell}) {
		t.Errorf("a worktree already there ran %q; want that same action.enter, alone", s.seen)
	}
}

// A flag naming the action wins over whichever key the moment would have read.
func TestAFlagWinsOverTheKey(t *testing.T) {
	var s steps
	e, r := opening(t, &s, "first", "first")

	fresh, _ := e.Resolve("x")
	if _, err := e.Enter(fresh, Options{Open: "second"}); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if !slices.Contains(s.seen, "open second") {
		t.Errorf("a fresh worktree ran %q; want the action the flag named", s.seen)
	}

	s.seen = nil
	if _, err := e.Enter(adopt(t, e, r), Options{Open: "second"}); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if !slices.Equal(s.seen, []string{"open second"}) {
		t.Errorf("a worktree already there ran %q; want the action the flag named", s.seen)
	}
}

// A name no action goes by is refused ahead of everything, so a typo costs
// nothing and no tracker is asked about a run that cannot finish.
func TestANameNoActionGoesByIsRefusedFirst(t *testing.T) {
	var s steps
	e, _ := opening(t, &s, "first", "first")

	c, _ := e.Resolve("x")
	_, err := e.Enter(c, Options{Open: "nobody"})
	if err == nil || !strings.Contains(err.Error(), `goes by the action "nobody"`) {
		t.Fatalf("Enter = %v; want the name refused", err)
	}
	if len(s.seen) != 0 {
		t.Errorf("naming no action left %q behind; want nothing run at all", s.seen)
	}
}

// The refusal guards the creation, so it holds whatever the invocation opens on
// and whether or not it declines an action.
func TestAPlaceThatCannotBeWorkedIsRefusedEveryWayIn(t *testing.T) {
	tests := []struct {
		name string
		opts Options
	}{
		{"the key's own action", Options{}},
		{"an action named by a flag", Options{Open: "second"}},
		{"an action declined", Options{Skip: []string{"first"}}},
		{"an action named and declined", Options{Open: "second", Skip: []string{"first"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s steps
			e, r := opening(t, &s, "first", "first")
			r.refuses = errors.New("x is blocked by an open dependency")

			c, _ := e.Resolve("x")
			if _, err := e.Enter(c, tt.opts); err == nil || !strings.Contains(err.Error(), "open dependency") {
				t.Errorf("Enter = %v; want the refusal, saying which rule it broke", err)
			}
			if !slices.Equal(s.seen, []string{"prepare"}) {
				t.Errorf("the run left %q behind; want the preparation alone", s.seen)
			}
		})
	}
}

// A front end that made its own place rather than resolving one has no resolver
// to prepare or create with, which is the one thing the sequence cannot do
// without.
func TestAPlaceNoResolverAnsweredForIsRefused(t *testing.T) {
	var s steps
	e, _ := opening(t, &s, "first", "first")

	if _, err := e.Enter(Candidate{}, Options{}); err == nil {
		t.Error("Enter with a place no resolver answered for: want it refused")
	}
}
