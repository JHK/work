package work

import (
	"errors"
	"fmt"
	"maps"
	"os"
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

	created string   // where the core asked for the worktree to be made
	asked   []string // the path of each worktree it was asked to supply values for
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
		return worktree.Place{ID: o.Path, Name: o.Name(), Branch: o.Branch}, nil
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
	r.asked = append(r.asked, t.Path)
	return worktree.Values{"ID": t.ID, "Label": "the resolver's", "Name": "the resolver's"}, nil
}

func (r *resolver) Create(p worktree.Place, path string) error {
	r.created = path
	r.steps.at("create " + filepath.Base(path) + " on " + p.Branch)
	return nil
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
	steps  *steps
	name   string
	got    worktree.Tree
	values worktree.Values
}

func (o *opener) Name() string { return o.name }

func (o *opener) Open(t worktree.Tree, vals worktree.Values) (worktree.Handoff, error) {
	o.steps.at("open " + o.name)
	o.got, o.values = t, vals
	return worktree.Handoff{Dir: t.Path, Run: []string{o.name}}, nil
}

// env wires one system of each sort over a repository that need not exist: the
// sequence is what is under test, not what git and bd make of it.
func env(t *testing.T, s *steps, openers ...*opener) (Env, *resolver) {
	t.Helper()
	r := &resolver{steps: s, name: "near", branch: "x-slug", adopts: "adopted-branch"}
	actions := []*action{{steps: s, name: "first"}, {steps: s, name: "second"}}
	cfg := config.Default()
	cfg.Action = config.Action{CreateName: config.ActionName(openers[0].name), EnterName: config.ActionName(openers[0].name)}
	// The chain ends in one answering for whatever worktree git reports, the main
	// checkout included, and for no identifier at all.
	tail := &resolver{steps: s, name: "tail", bare: true, unknown: true}
	e := Env{Repo: testenv.InitRepo(t), Config: cfg, Systems: Systems{
		Resolvers: []Resolver{r, tail},
		Actions:   []Action{actions[0], actions[1]},
		Named:     r,
	}}
	for _, o := range openers {
		e.Systems.Openers = append(e.Systems.Openers, o)
	}
	return e, r
}

// The sequence is the core's whole job: prepare the place, create the worktree,
// run what its coming into being means, hand it over.
func TestEnterRunsTheSeamsInOrder(t *testing.T) {
	var s steps
	op := &opener{steps: &s, name: "far"}
	e, r := env(t, &s, op)

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
// worktree is made and no action runs.
func TestAPlaceThatCannotBeWorkedLeavesNothingBehind(t *testing.T) {
	var s steps
	e, r := env(t, &s, &opener{steps: &s, name: "far"})
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
	e, r := env(t, &s, &opener{steps: &s, name: "far"})
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

// A directory already sitting where the worktree would go is the core's to
// refuse: it names that directory in one line, and no resolver is asked to
// create anything over it.
func TestADirectoryAlreadyWhereTheWorktreeGoesIsRefused(t *testing.T) {
	var s steps
	e, _ := env(t, &s, &opener{steps: &s, name: "far"})
	path := filepath.Join(e.Repo, defaultDir, "x")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	c, err := e.Resolve("x")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	_, err = e.Enter(c, Options{})
	if err == nil || !strings.Contains(err.Error(), path) || namesGit(err) {
		t.Fatalf("Enter = %v; want %q named and no git command", err, path)
	}
	if !slices.Equal(s.seen, []string{"prepare"}) {
		t.Errorf("the sequence ran %q; want the preparation alone", s.seen)
	}
}

// An action is declined by the name it goes by, which is what --no-claim spells.
func TestAnActionDeclinedByNameDoesNotRun(t *testing.T) {
	var s steps
	e, _ := env(t, &s, &opener{steps: &s, name: "far"})

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

// The third form of the question is what a system that cannot be reached needs: it
// names no worktree from a branch alone, so the worktree falls to the resolver
// answering for whatever is left, and the identifier is what recognises it again.
func TestAPlaceIsFoundOnAWorktreeNothingCouldName(t *testing.T) {
	var s steps
	op := &opener{steps: &s, name: "far"}
	e, r := env(t, &s, op)
	r.mute = true
	path := filepath.Join(e.Repo, defaultDir, "a")
	testenv.Git(t, e.Repo, "worktree", "add", "-b", r.adopts, path)

	// The listing has it under its branch, nothing having named it.
	open, err := e.Worktrees()
	if err != nil {
		t.Fatalf("Worktrees: %v", err)
	}
	// Beside the main checkout, which nothing named either.
	if len(open) != 2 || open[1].Source != "tail" || open[1].Name != r.adopts {
		t.Fatalf("the listing holds %+v; want the worktree at %q under the resolver that took what was left", open, path)
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
	e, _ := env(t, &s, &opener{steps: &s, name: "far"})
	chain := e.Systems.Resolvers
	silent := &resolver{steps: &s, name: "silent", unknown: true}
	e.Systems.Resolvers = append([]Resolver{silent}, chain...)

	c, err := e.Resolve("x")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if c.Source != "near" {
		t.Errorf("Resolve reached %q; want the resolver after the one that passed", c.Source)
	}

	broken := &resolver{steps: &s, name: "broken", fails: errors.New("bd is not answering")}
	e.Systems.Resolvers = append([]Resolver{broken}, chain...)
	if _, err := e.Resolve("x"); err == nil || err.Error() != broken.fails.Error() {
		t.Errorf("Resolve = %v; want the failure to stop the run", err)
	}
}

// A command is rendered from the core's account of the worktree and from the
// resolver that answered for the place, which is the only system describing the
// place rather than any worktree. A name both supply is the core's.
func TestValuesComeFromTheResolverThatAnswered(t *testing.T) {
	var s steps
	op := &opener{steps: &s, name: "far"}
	e, r := env(t, &s, op)

	c, err := e.Resolve("x")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if _, err := e.Enter(c, Options{}); err != nil {
		t.Fatalf("Enter: %v", err)
	}

	want := worktree.Values{
		"Name": "x", "Dir": op.got.Path,
		"ID": "x", "Label": "the resolver's",
	}
	if !maps.Equal(op.values, want) {
		t.Errorf("the command was rendered with %v; want %v", op.values, want)
	}

	// Asked once, with the worktree there, so a resolver that can only answer from
	// inside one has one to answer from.
	if len(r.asked) != 1 || r.asked[0] == "" {
		t.Errorf("the resolver was asked about %q; want the worktree that now exists, once", r.asked)
	}
}

// A worktree already there is neither prepared nor acted on: it is read off the
// listing under the name the resolver that adopted it gave, and opened.
func TestAWorktreeAlreadyThereIsOnlyOpened(t *testing.T) {
	var s steps
	op := &opener{steps: &s, name: "far"}
	e, r := env(t, &s, op)

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

// A place the chain answered for that no worktree could be made for is refused
// as the name rule, and a resolver that recognised the identifier and then could
// not answer stops the run on what it said.
func TestWhatTheChainAnsweredWithIsWhatTheRefusalReads(t *testing.T) {
	var s steps
	e, r := env(t, &s, &opener{steps: &s, name: "far"})

	if _, err := e.Resolve("a/b"); err == nil || !strings.Contains(err.Error(), "usable worktree name") {
		t.Errorf(`Resolve("a/b") = %v; want the name rule`, err)
	}

	r.fails = errors.New("the forge is not answering")
	if _, err := e.Resolve("x"); err == nil || err.Error() != r.fails.Error() {
		t.Errorf(`Resolve("x") = %v; want the failure the resolver stopped on`, err)
	}
}
