package work

import (
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
)

// steps is the order the core put the seams in, which is most of what the
// sequence is.
type steps struct{ seen []string }

func (s *steps) at(step string) { s.seen = append(s.seen, step) }

// resolver is one system on the near seam, saying as little as a resolver can.
// It leaves Place.Source alone, as an implementation does: the core stamps it.
type resolver struct {
	steps   *steps
	name    string
	unknown bool   // recognises no identifier, so the chain moves on
	fails   error  // recognises the identifier but cannot answer for it
	names   string // what it makes of an identifier, the identifier itself where unset
	refuses error  // will not have a worktree made for this place
	branch  string // what Prepare names
	renames string // what Prepare calls the place, where it renames it at all
	adopts  string // the branch it answers for
	mute    bool   // cannot name a worktree from its branch alone, as an unreachable tracker cannot
	bare    bool   // answers for any worktree, as the last resolver in a chain does

	created string // where the core asked for the worktree to be made
}

func (r *resolver) Name() string { return r.name }

// adopted is the one place this resolver answers for a worktree by.
const adopted = "adopted"

func (r *resolver) Identify(id string, o worktree.Open) (worktree.Place, error) {
	if o.None() {
		switch {
		case r.unknown:
			return worktree.Place{}, fmt.Errorf("%w: %s wants none of %q", worktree.ErrUnknown, r.name, id)
		case r.fails != nil:
			return worktree.Place{}, r.fails
		}
		// A resolver that reads an identifier into a name of its own, as one turning a
		// pull request URL into the branch it checks out does.
		if r.names != "" {
			return worktree.Place{ID: id, Name: r.names}, nil
		}
		return worktree.Place{ID: id, Name: id}, nil
	}
	if r.bare {
		// Nothing behind it to be named by, so an identifier to confirm is settled by
		// the directory, as it is for a resolver with no system behind it.
		if id != "" && id != o.Path {
			return worktree.Place{}, worktree.ErrUnknown
		}
		return worktree.Place{ID: o.Path, Name: o.Branch, Branch: o.Branch}, nil
	}
	// The one branch it answers for, and only under the name it answers for it by.
	if o.Branch != r.adopts || (id != "" && id != adopted) {
		return worktree.Place{}, worktree.ErrUnknown
	}
	// From a branch alone a system that cannot be reached names nothing; asked with the
	// identifier as well it recognises its own worktree without asking anything.
	if id == "" && r.mute {
		return worktree.Place{}, worktree.ErrUnknown
	}
	return worktree.Place{ID: adopted, Name: adopted, Branch: o.Branch, Label: "a title"}, nil
}

func (r *resolver) Offer() ([]worktree.Place, error) { return nil, nil }

func (r *resolver) Prepare(p worktree.Place) (worktree.Place, error) {
	r.steps.at("prepare")
	if r.refuses != nil {
		return p, r.refuses
	}
	if r.renames != "" {
		p.Name = r.renames
	}
	p.Branch, p.Label = r.branch, "a title"
	return p, nil
}

// Supply is this resolver's account of the place it answered for, which is what an
// action learns about the system behind a worktree.
func (r *resolver) Supply(t worktree.Tree) (worktree.Values, error) {
	return worktree.Values{"ID": t.ID, "Label": "the resolver's"}, nil
}

func (r *resolver) Create(p worktree.Place, path string) error {
	r.created = path
	r.steps.at("create " + filepath.Base(path) + " on " + p.Branch)
	return nil
}

// forge is a resolver that knows its own identifiers by the way they are spelled,
// as a forge knows a pull request URL. A resolver on its own knows no such thing,
// which is what a tracker is: any name at all is a possible ticket id.
type forge struct{ *resolver }

func (f forge) Claims(id string) (worktree.Place, bool) {
	p, err := f.Identify(id, worktree.Open{})
	return p, err == nil
}

// action is one system on the far seam's on-create half.
type action struct {
	steps *steps
	name  string
}

func (a *action) Name() string { return a.name }

func (a *action) Run(t worktree.Tree) error {
	a.steps.at("run " + a.name)
	return nil
}

// opener is the far seam's other half, and records the worktree it was handed.
type opener struct {
	steps   *steps
	name    string
	absent  bool  // has nothing to hand the worktree to
	fails   error // cannot say whether it has, which is a different refusal
	got     worktree.Tree
	values  worktree.Values
	applied worktree.Values // what Applies was judged against
}

func (o *opener) Name() string { return o.name }

func (o *opener) Applies(vals worktree.Values) error {
	o.applied = vals
	switch {
	case o.absent:
		return worktree.Absent(fmt.Errorf("no %s here", o.name))
	case o.fails != nil:
		return o.fails
	}
	return nil
}

func (o *opener) Open(t worktree.Tree, vals worktree.Values) (worktree.Handoff, error) {
	o.steps.at("open " + o.name)
	o.got, o.values = t, vals
	return worktree.Handoff{Dir: t.Path, Run: []string{o.name}}, nil
}

// env wires one system of each sort over a repository that need not exist: the
// sequence is what is under test, not what git and bd make of it.
func env(t *testing.T, s *steps, openers ...*opener) (Env, *resolver, []*action) {
	t.Helper()
	r := &resolver{steps: s, name: "near", branch: "x-slug", adopts: "adopted-branch"}
	actions := []*action{{steps: s, name: "first"}, {steps: s, name: "second"}}
	cfg := config.Default()
	cfg.Action = config.Action{CreateName: config.ActionName(openers[0].name), EnterName: config.ActionName(openers[0].name)}
	e := Env{Repo: testenv.InitRepo(t), Config: cfg, Systems: Systems{
		Resolvers: []Resolver{r},
		Actions:   []Action{actions[0], actions[1]},
		Named:     r,
	}}
	for _, o := range openers {
		e.Systems.Openers = append(e.Systems.Openers, o)
	}
	return e, r, actions
}

// The sequence is the core's whole job: prepare the place, create the worktree,
// run what its coming into being means, hand it over.
func TestEnterRunsTheSeamsInOrder(t *testing.T) {
	var s steps
	op := &opener{steps: &s, name: "far"}
	e, r, _ := env(t, &s, op)

	c, err := e.Resolve("x")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	h, err := e.Enter(c, Options{})
	if err != nil {
		t.Fatalf("Enter: %v", err)
	}

	want := []string{"prepare", "create x on x-slug", "run first", "run second", "open far"}
	if !slices.Equal(s.seen, want) {
		t.Errorf("the sequence ran %q; want %q", s.seen, want)
	}
	if h.Run[0] != "far" {
		t.Errorf("Enter handed over to %q; want the opener's own command", h.Run)
	}

	// What the action is told: the worktree is fresh, the branch is the one Prepare
	// named, and the resolver is there to be asked for the rest.
	if !op.got.Created {
		t.Error("the opener was not told the worktree is fresh")
	}
	if op.got.Branch != "x-slug" || op.got.Label != "a title" {
		t.Errorf("the opener was handed %+v; want the place Prepare completed", op.got.Place)
	}
	if op.got.By != Resolver(r) {
		t.Error("the opener was not told which resolver answered")
	}
}

// A place that cannot be worked is refused where the refusal costs nothing: no
// worktree is made, no action runs, and no screen is drawn.
func TestAPlaceThatCannotBeWorkedLeavesNothingBehind(t *testing.T) {
	var s steps
	e, r, _ := env(t, &s, &opener{steps: &s, name: "far"})
	r.refuses = errors.New("x is blocked by an open dependency")

	c, _ := e.Resolve("x")
	if _, err := e.Enter(c, Options{}); err == nil || err.Error() != r.refuses.Error() {
		t.Fatalf("Enter = %v; want the resolver's refusal", err)
	}
	if !slices.Equal(s.seen, []string{"prepare"}) {
		t.Errorf("the sequence ran %q; want the preparation alone", s.seen)
	}
}

// Completing a place is the one moment a resolver may still change its name, and
// the name is about to become a directory of its own. The rule is the core's
// wherever a name reaches a path, so a place completed into one no worktree could
// carry is refused with nothing created.
func TestANameThePreparationChangedIsHeldToTheSameRule(t *testing.T) {
	var s steps
	e, r, _ := env(t, &s, &opener{steps: &s, name: "far"})
	r.renames = "../elsewhere"

	c, err := e.Resolve("x")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if _, err := e.Enter(c, Options{}); err == nil || !strings.Contains(err.Error(), "usable worktree name") {
		t.Fatalf("Enter = %v; want the name refused", err)
	}
	if !slices.Equal(s.seen, []string{"prepare"}) {
		t.Errorf("a name no worktree could carry left %q behind; want the preparation alone", s.seen)
	}
}

// An action is declined by the name it goes by, which is what --no-claim spells.
func TestAnActionDeclinedByNameDoesNotRun(t *testing.T) {
	var s steps
	e, _, _ := env(t, &s, &opener{steps: &s, name: "far"})

	c, _ := e.Resolve("x")
	if _, err := e.Enter(c, Options{Skip: []string{"first"}}); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if slices.Contains(s.seen, "run first") {
		t.Errorf("the sequence ran %q; want the declined action left out", s.seen)
	}
	if !slices.Contains(s.seen, "run second") {
		t.Errorf("the sequence ran %q; want the action that was not declined", s.seen)
	}
}

// The screen sits between the refusal and the creation: a place that cannot be
// worked is never asked about, and a screen dismissed leaves nothing made.
func TestTheScreenSitsBetweenTheRefusalAndTheCreation(t *testing.T) {
	var s steps
	op := &opener{steps: &s, name: "far"}
	e, _, _ := env(t, &s, op)
	asked := func(offer []string) (string, error) {
		s.at("ask " + offer[0])
		return "far", nil
	}

	c, _ := e.Resolve("x")
	if _, err := e.Enter(c, Options{Open: ask, Ask: asked}); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	want := []string{"prepare", "ask far", "create x on x-slug", "run first", "run second", "open far"}
	if !slices.Equal(s.seen, want) {
		t.Errorf("the sequence ran %q; want %q", s.seen, want)
	}

	s.seen = nil
	dismissed := func([]string) (string, error) { return "", errors.New("cancelled") }
	if _, err := e.Enter(c, Options{Open: ask, Ask: dismissed}); err == nil {
		t.Fatal("Enter: want the dismissal")
	}
	if !slices.Equal(s.seen, []string{"prepare"}) {
		t.Errorf("a dismissed screen left %q behind; want the preparation alone", s.seen)
	}
}

// An action whose tool is not there is left off the screen, and refused ahead of
// everything where a flag named it.
func TestAnAbsentActionIsOffTheOfferAndRefusedWhenNamed(t *testing.T) {
	var s steps
	missing := &opener{steps: &s, name: "missing", absent: true}
	there := &opener{steps: &s, name: "there"}
	e, _, _ := env(t, &s, missing, there)

	got, err := e.offer(nil)
	if err != nil {
		t.Fatalf("offer: %v", err)
	}
	if !slices.Equal(got, []string{"there"}) {
		t.Errorf("the offer is %q; want the action that applies alone", got)
	}

	c, _ := e.Resolve("x")
	if _, err := e.Enter(c, Options{Open: "missing"}); !errors.Is(err, worktree.ErrAbsent) {
		t.Fatalf("Enter = %v; want an absent action", err)
	}
	// The preparation ran, being what the values are judged against and free of
	// consequence either way; nothing past it did.
	if !slices.Equal(s.seen, []string{"prepare"}) {
		t.Errorf("naming an absent action left %q behind; want the preparation alone", s.seen)
	}
}

// An action refusing for anything but an absent tool has failed rather than found
// nothing to hand the worktree to: the run stops and the refusal reaches the user,
// whichever way the action was reached. Off the screen is where the other one goes,
// and a failure vanishing there would be a failure nobody is told about.
func TestAnOpenerFailingForAnotherReasonStopsTheRun(t *testing.T) {
	for _, tt := range []struct {
		name string
		opts Options
	}{
		{"flag", Options{Open: "broken"}},
		// A screen drawn at all is the regression: the failure would have gone off the
		// offer and something else been chosen in its place.
		{"screen", Options{Open: ask, Ask: func([]string) (string, error) {
			return "", errors.New("the screen was drawn")
		}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var s steps
			broken := &opener{steps: &s, name: "broken", fails: errors.New("the editor's settings will not parse")}
			e, _, _ := env(t, &s, broken, &opener{steps: &s, name: "there"})

			c, _ := e.Resolve("x")
			_, err := e.Enter(c, tt.opts)
			if err == nil || !strings.Contains(err.Error(), broken.fails.Error()) {
				t.Fatalf("Enter = %v; want the action's own refusal", err)
			}
			if !slices.Equal(s.seen, []string{"prepare"}) {
				t.Errorf("a failing action left %q behind; want the preparation alone", s.seen)
			}
		})
	}
}

// An action a flag named and one the screen chose are judged against the same values,
// so an action gating on what a resolver supplied cannot apply one way and not the
// other. The preparation is what puts a ticket's title on the place, so it runs ahead
// of both.
func TestAFlagAndTheScreenJudgeTheSameValues(t *testing.T) {
	var byFlag, byScreen worktree.Values
	for _, tt := range []struct {
		name string
		opts func(*opener) Options
		got  *worktree.Values
	}{
		{"flag", func(op *opener) Options { return Options{Open: op.name} }, &byFlag},
		{"screen", func(op *opener) Options {
			return Options{Open: ask, Ask: func(offer []string) (string, error) { return offer[0], nil }}
		}, &byScreen},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var s steps
			op := &opener{steps: &s, name: "far"}
			e, _, _ := env(t, &s, op)
			c, err := e.Resolve("x")
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if _, err := e.Enter(c, tt.opts(op)); err != nil {
				t.Fatalf("Enter: %v", err)
			}
			*tt.got = op.applied
		})
	}
	if !maps.Equal(byFlag, byScreen) {
		t.Errorf("a flag judged the action against %v and the screen against %v; want the same", byFlag, byScreen)
	}
	if byFlag["Label"] == "" {
		t.Errorf("the action was judged against %v; want what the preparation put on the place", byFlag)
	}
}

// The third form of the question is what a system that cannot be reached needs: it
// names no worktree from a branch alone, so the worktree falls to the resolver
// answering for whatever is left, and the identifier is what recognises it again.
func TestAPlaceIsFoundOnAWorktreeNothingCouldName(t *testing.T) {
	var s steps
	op := &opener{steps: &s, name: "far"}
	e, r, _ := env(t, &s, op)
	r.mute = true
	e.Systems.Resolvers = []Resolver{r, &resolver{steps: &s, name: "bare", bare: true}}
	testenv.Git(t, e.Repo, "worktree", "add", "-b", r.adopts, filepath.Join(e.Repo, defaultDir, "a"))

	// The listing has it under its branch, nothing having named it.
	open, err := e.Worktrees()
	if err != nil {
		t.Fatalf("Worktrees: %v", err)
	}
	if len(open) != 1 || open[0].Source != "bare" {
		t.Fatalf("the listing holds %+v; want the worktree under the resolver that took what was left", open)
	}

	c, err := e.Resolve(adopted)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !c.Open || c.Source != r.name {
		t.Fatalf("Resolve = %+v; want the worktree found by its identifier, under %s", c.Place, r.name)
	}
	if _, err := e.Enter(c, Options{}); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if !slices.Equal(s.seen, []string{"open far"}) {
		t.Errorf("the sequence ran %q; want the handover alone", s.seen)
	}
}

// An identifier a resolver does not recognise moves to the next; one it
// recognised but could not answer for stops the run.
func TestUnrecognitionChainsAndFailureStops(t *testing.T) {
	var s steps
	e, r, _ := env(t, &s, &opener{steps: &s, name: "far"})
	silent := &resolver{steps: &s, name: "silent", unknown: true}
	e.Systems.Resolvers = []Resolver{silent, r}

	c, err := e.Resolve("x")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if c.Source != "near" {
		t.Errorf("Resolve reached %q; want the resolver after the one that passed", c.Source)
	}

	broken := &resolver{steps: &s, name: "broken", fails: errors.New("bd is not answering")}
	e.Systems.Resolvers = []Resolver{broken, r}
	if _, err := e.Resolve("x"); err == nil || err.Error() != broken.fails.Error() {
		t.Errorf("Resolve = %v; want the failure to stop the run", err)
	}
}

// source supplies a fixed set, and records the worktree it was asked about.
type source struct {
	supply worktree.Values
	asked  []string // the path of each worktree it was asked about
}

func (s *source) Supply(t worktree.Tree) (worktree.Values, error) {
	s.asked = append(s.asked, t.Path)
	return s.supply, nil
}

// A command is rendered from what the systems wired together happen to know
// between them: the resolver that answered for the place, then the ambient sources.
// A name two of them supply is the resolver's, which is the only one describing the
// place rather than any worktree.
func TestValuesComeFromTheSourcesAndTheResolverFirst(t *testing.T) {
	var s steps
	op := &opener{steps: &s, name: "far"}
	e, _, _ := env(t, &s, op)
	ambient := &source{supply: worktree.Values{"Editor": "gvim", "Label": "the source's"}}
	e.Systems.Sources = []worktree.Source{ambient}

	c, err := e.Resolve("x")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if _, err := e.Enter(c, Options{}); err != nil {
		t.Fatalf("Enter: %v", err)
	}

	// The worktree itself, the resolver's account of it, and the ambient source's,
	// with the resolver winning the name they both supply.
	want := worktree.Values{
		"Name": "x", "Dir": op.got.Path,
		"ID": "x", "Label": "the resolver's",
		"Editor": "gvim",
	}
	if !maps.Equal(op.values, want) {
		t.Errorf("the command was rendered with %v; want %v", op.values, want)
	}

	// Asked once before there was a worktree and once after, so a source that can only
	// answer from inside one gets the chance to.
	if len(ambient.asked) != 2 || ambient.asked[0] != "" || ambient.asked[1] == "" {
		t.Errorf("the source was asked about %q; want one worktree that is not there yet and one that is", ambient.asked)
	}
}

// The handoff on its own is the last step without the sequence in front of it,
// which is what a caller holding something to hand over that is not a worktree
// reaches: the named action renders against the core's values, and nothing is
// prepared, created or run.
func TestHandoffRendersTheNamedActionAlone(t *testing.T) {
	var s steps
	op := &opener{steps: &s, name: "far"}
	e, _, _ := env(t, &s, op)
	e.Systems.Sources = []worktree.Source{&source{supply: worktree.Values{"Editor": "gvim"}}}
	tree := worktree.Tree{Place: worktree.Place{Name: "a-file"}, Path: "/somewhere/a-file"}

	h, err := e.Handoff(tree, "far")
	if err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	if !slices.Equal(h.Run, []string{"far"}) {
		t.Errorf("Handoff() runs %q; want the opener's own command", h.Run)
	}
	if !slices.Equal(s.seen, []string{"open far"}) {
		t.Errorf("Handoff() ran %q; want the opener and nothing else", s.seen)
	}
	want := worktree.Values{"Name": "a-file", "Dir": tree.Path, "Editor": "gvim"}
	if !maps.Equal(op.values, want) {
		t.Errorf("the command was rendered with %v; want %v", op.values, want)
	}

	if _, err := e.Handoff(tree, "nothing"); err == nil {
		t.Error("Handoff() took an action nothing goes by")
	}
}

// A worktree already there is neither prepared nor acted on: it is read off the
// listing under the name the resolver that adopted it gave, and opened.
func TestAWorktreeAlreadyThereIsOnlyOpened(t *testing.T) {
	var s steps
	op := &opener{steps: &s, name: "far"}
	e, r, _ := env(t, &s, op)

	c := adopt(t, e, r)
	if c.Label != "a title" {
		t.Fatalf("Resolve = %+v; want the worktree the resolver adopted", c.Place)
	}
	if _, err := e.Enter(c, Options{}); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if !slices.Equal(s.seen, []string{"open far"}) {
		t.Errorf("the sequence ran %q; want the handover alone", s.seen)
	}
	if op.got.Created {
		t.Error("the opener was told a worktree that was already there is fresh")
	}
}

// A system the settings left out is wired to nothing and asked one question, once
// the wired resolvers have already refused: whether the identifier is spelled as
// one of its own. What it claims is refused as a system switched off, naming the
// key that puts it back, rather than as an identifier work cannot read.
//
// A resolver that answers for whatever is left masks one that is off, so the
// answer the chain did give is judged the same way: a place no worktree could be
// made for is one nothing really answered for.
func TestAnIdentifierASwitchedOffSystemAnswersForNamesItsKey(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		refused string // what the refusal reads without the system to name
	}{
		{"nothing answered for it", "x", "nothing answers for"},
		{"what answered made no place a worktree could be", "a/b", "usable worktree name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s steps
			e, r, _ := env(t, &s, &opener{steps: &s, name: "far"})
			r.unknown = tt.id == "x"

			if _, err := e.Resolve(tt.id); err == nil || !strings.Contains(err.Error(), tt.refused) {
				t.Fatalf("Resolve(%q) = %v; want %q where nothing is switched off", tt.id, err, tt.refused)
			}

			// It reads the identifier into a name of its own, which is what makes turning
			// it back on an answer: one that would land where the wired chain just did has
			// nothing to offer and is not named.
			e.Systems.Disabled = []worktree.System{forge{&resolver{steps: &s, name: "forge", names: "a-place"}}}
			_, err := e.Resolve(tt.id)
			if err == nil || !strings.Contains(err.Error(), "forge is off, put it back with forge.enabled = true") {
				t.Errorf("Resolve(%q) = %v; want the key that would have answered for it", tt.id, err)
			}
		})
	}
}

// A system that is off is named only where putting it back would have answered.
// One that would land exactly where the wired chain landed, on a name no worktree
// could carry, is not an answer, and a wired resolver that recognised the
// identifier and then failed has already said what the run stops on.
func TestASwitchedOffSystemIsNotNamedWhereItWouldNotHaveHelped(t *testing.T) {
	var s steps
	e, r, _ := env(t, &s, &opener{steps: &s, name: "far"})
	e.Systems.Disabled = []worktree.System{forge{&resolver{steps: &s, name: "forge"}}}

	// The off system makes the same unusable name of it that the wired one did.
	if _, err := e.Resolve("a/b"); err == nil || !strings.Contains(err.Error(), "usable worktree name") {
		t.Errorf(`Resolve("a/b") = %v; want the name rule, which turning the forge back on would not have got past`, err)
	}

	// A resolver that recognised the identifier and could not answer for it.
	r.fails = errors.New("the forge is not answering")
	if _, err := e.Resolve("x"); err == nil || err.Error() != r.fails.Error() {
		t.Errorf(`Resolve("x") = %v; want the failure the wired resolver stopped on`, err)
	}
}

// A name no switched-off system claims as its own is refused as a name nothing
// answers for, and attributed to none of them. A tracker cannot tell a ticket id
// from a typo without asking, and not asking is what switching it off was for, so
// what being off costs it is its rows and never someone else's typo.
func TestANameNoSwitchedOffSystemClaimsIsAttributedToNone(t *testing.T) {
	var s steps
	e, r, _ := env(t, &s, &opener{steps: &s, name: "far"})
	r.unknown = true
	e.Systems.Disabled = []worktree.System{
		// One that could only answer by asking, and so answers no such question at all.
		&resolver{steps: &s, name: "tracker"},
		// One that answers it, and for whom this is not one of its own.
		forge{&resolver{steps: &s, name: "forge", unknown: true}},
	}

	_, err := e.Resolve("typo")
	if err == nil || !strings.Contains(err.Error(), `nothing answers for "typo"`) {
		t.Fatalf(`Resolve("typo") = %v; want it refused as a name nothing answers for`, err)
	}
	for _, name := range []string{"tracker", "forge"} {
		if strings.Contains(err.Error(), name) {
			t.Errorf(`Resolve("typo") = %v; want no key named for a spelling %s never claimed`, err, name)
		}
	}
}

// An action of a system the settings left out is refused as that, and where a
// key naming it is read: before anything is created.
func TestAnActionOfASwitchedOffSystemNamesItsKey(t *testing.T) {
	var s steps
	e, _, _ := env(t, &s, &opener{steps: &s, name: "far"})
	e.Systems.Disabled = []worktree.System{&opener{steps: &s, name: "agent"}}

	c, _ := e.Resolve("x")
	_, err := e.Enter(c, Options{Open: "agent"})
	if err == nil || !strings.Contains(err.Error(), "agent is off, put it back with agent.enabled = true") {
		t.Fatalf("Enter = %v; want the key that puts the action back", err)
	}
	if len(s.seen) != 0 {
		t.Errorf("naming a switched-off action left %q behind; want nothing run at all", s.seen)
	}
}
