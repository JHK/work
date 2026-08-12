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

// opening is the wiring of [env] with two openers a key or a flag can name and a
// third the machine has nothing to hand a worktree to, and with each moment's own
// key named rather than left to the first opener.
func opening(t *testing.T, s *steps, create, enter string) (Env, *resolver) {
	t.Helper()
	e, r, _ := env(t, s,
		&opener{steps: s, name: "first"},
		&opener{steps: s, name: "second"},
		&opener{steps: s, name: "missing", absent: true},
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
// them, and neither is offered an Ask, so a default naming the screen would be a
// refusal here rather than a handover.
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

// The screen is reached by ask, whether the flag or the key spells it, and what
// came back wins over the key that would otherwise have settled it.
func TestAskReachesTheScreenWhicheverNamedIt(t *testing.T) {
	for _, tt := range []struct {
		name, enter string
		opts        Options
	}{
		{"the key", ask, Options{}},
		{"the flag over a key naming an action", "first", Options{Open: ask}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var s steps
			e, r := opening(t, &s, "first", tt.enter)
			c := adopt(t, e, r)

			var offered []string
			opts := tt.opts
			opts.Ask = func(offer []string) (string, error) {
				offered = offer
				return "second", nil
			}
			if _, err := e.Enter(c, opts); err != nil {
				t.Fatalf("Enter: %v", err)
			}
			// The offer is what will run, in the order the openers were wired, and the
			// action the machine has nothing for is not on it.
			if want := []string{"first", "second"}; !slices.Equal(offered, want) {
				t.Errorf("the screen was offered %q; want %q", offered, want)
			}
			if !slices.Equal(s.seen, []string{"open second"}) {
				t.Errorf("the run ran %q; want the action the screen chose", s.seen)
			}
		})
	}
}

// The screen is what settles ask, so a front end with none, and one whose screen
// settled nothing, are both refused rather than handed an action work chose for
// them: there is no action to fall through to.
func TestAnUnsettledScreenIsRefused(t *testing.T) {
	answering := func(name string) Ask {
		return func([]string) (string, error) { return name, nil }
	}
	tests := []struct {
		name string
		ask  Ask
		want string
	}{
		{"no screen at all", nil, "name one with a flag"},
		{"a screen answering ask", answering(ask), "named no action"},
		{"a screen answering nothing", answering(""), "named no action"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s steps
			e, r := opening(t, &s, "first", ask)
			c := adopt(t, e, r)

			_, err := e.Enter(c, Options{Ask: tt.ask})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Enter = %v; want it refused, saying %q", err, tt.want)
			}
		})
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

// A key naming an action the machine has nothing to hand a worktree to is
// refused where the flag naming it is: before anything is created, so nothing is
// left made or claimed.
func TestAKeyNamingAnAbsentActionIsRefusedBeforeCreating(t *testing.T) {
	var s steps
	e, _ := opening(t, &s, "missing", "first")

	c, _ := e.Resolve("x")
	if _, err := e.Enter(c, Options{}); err == nil {
		t.Fatal("Enter: want the absent action refused")
	}
	// The preparation ran, being what the values are judged against and free of
	// consequence either way; nothing past it did.
	if !slices.Equal(s.seen, []string{"prepare"}) {
		t.Errorf("a key naming an absent action left %q behind; want the preparation alone", s.seen)
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
		{"a screen", Options{Open: ask}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s steps
			e, r := opening(t, &s, "first", "first")
			r.refuses = errors.New("x is blocked by an open dependency")

			opts := tt.opts
			opts.Ask = func([]string) (string, error) {
				t.Error("asked which action to open on despite the place being refused")
				return "first", nil
			}
			c, _ := e.Resolve("x")
			if _, err := e.Enter(c, opts); err == nil || !strings.Contains(err.Error(), "open dependency") {
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
