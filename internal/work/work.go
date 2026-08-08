// Package work is the core beneath every front end: it resolves an identifier
// to a target, reports whether that target already has a worktree, and takes
// entering one from vetting through provisioning and claiming to the handoff.
// A front end supplies the options and presents the result; the policy is here.
package work

import (
	"cmp"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JHK/work-cli/internal/beads"
	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/forge"
	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/sessions"
)

// Kind distinguishes the namespaces an identifier can land in.
type Kind int

const (
	KindBead Kind = iota
	KindPR
	// KindPlain is a worktree whose branch names neither: it is offered as it is
	// and entered with a shell. It is only ever discovered, never created.
	KindPlain
)

// Target is one resolved place to work.
type Target struct {
	Kind Kind
	ID   string // bead id, pull request number, or, for a plain worktree, its path
	Name string // what a row shows and the user retypes: the bead id, pr-<n>, or the branch
}

// Env holds the repository work operates on, the settings it reads, and the
// agent it asks what a worktree already carries.
type Env struct {
	Repo          string
	Config        config.Config
	Conversations sessions.Conversations
}

// Open finds the repository containing dir, reads its settings, and names the
// agent behind them.
func Open(dir string) (Env, error) {
	repo, err := git.Root(dir)
	if err != nil {
		return Env{}, err
	}
	cfg, err := config.Load(repo)
	if err != nil {
		return Env{}, err
	}
	return Env{Repo: repo, Config: cfg, Conversations: sessions.Claude{}}, nil
}

// State is everything knowable about a target on entry, less what an existing
// worktree makes moot.
type State struct {
	Target    Target
	Path      string // where the worktree is, or where it would be created
	Exists    bool
	Bead      beads.Bead
	Ready     bool  // bd has already been heard to call this bead workable
	TicketErr error // bd could not answer for this target
}

var (
	prURL = regexp.MustCompile(`^(?:[a-z]+://[^/]+/)?[^/\s]+/[^/\s]+/pull/([0-9]+)(?:[/?#].*)?$`)
	// The name becomes a directory of its own and an argument to bd and git, so it
	// may not traverse, and may not open with a dash.
	worktreeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
)

// Resolve maps an identifier to the place it names and to the worktree that
// place already has. One listing serves both readings: the name is read off the
// listing where a worktree answers to it and guessed where none does, and either
// way the worktree is found by its branch.
func (e Env) Resolve(arg string) (Candidate, error) {
	if arg == "" {
		return Candidate{}, errors.New("no target given")
	}
	worktrees, known := e.listing()
	t, err := e.named(arg, worktrees, known)
	if err != nil {
		return Candidate{}, err
	}
	path := e.located(t, worktrees, known)
	return Candidate{Target: t, Open: path != "", path: path, bead: record(known, t)}, nil
}

// named is the target an identifier stands for: a worktree already listed under
// the name is that worktree, whatever guess would make of the name, so that what
// the picker and the completion show is entered by retyping it.
func (e Env) named(arg string, worktrees []git.Worktree, known []beads.Bead) (Target, error) {
	for _, w := range worktrees {
		if t := e.targetAt(w, known); t.Name == arg {
			return t, nil
		}
	}
	return e.guess(arg)
}

// listing is the one answer a whole invocation works from: every worktree git
// reports, and the beads that name them, asked for only where there is a
// worktree to name. Nothing is known where bd would not answer: a worktree of a
// bead nothing can name is merely a plain one.
func (e Env) listing() ([]git.Worktree, []beads.Bead) {
	// Without a repository git would answer for whatever directory the process
	// happens to be in.
	if e.Repo == "" {
		return nil, nil
	}
	worktrees, err := git.Linked(e.Repo)
	if err != nil || len(worktrees) == 0 {
		return nil, nil
	}
	known, _ := beads.All(e.Repo)
	return worktrees, known
}

// guess reads an identifier no worktree answers to. A bare number, an
// .../<owner>/<repo>/pull/<n> URL, or a name a pull request's branch pattern
// could have produced is a pull request; anything else is a bead id.
func (e Env) guess(arg string) (Target, error) {
	n := ""
	if _, err := strconv.ParseUint(arg, 10, 32); err == nil {
		n = arg
	} else if m := prURL.FindStringSubmatch(arg); m != nil {
		n = m[1]
	} else if number, ok := e.Config.Branch.NumberIn(arg); ok {
		// The worktree's own name, so that what the picker shows can be retyped.
		n = number
	}
	if n != "" {
		// Canonicalise, so that 7 and 007 name one worktree and one refspec.
		i, err := strconv.ParseUint(n, 10, 32)
		if err != nil || i == 0 {
			return Target{}, fmt.Errorf("%q is not a pull request number", arg)
		}
		return e.prTarget(strconv.FormatUint(i, 10)), nil
	}

	if !worktreeName.MatchString(arg) {
		return Target{}, fmt.Errorf("%q is not a usable worktree name", arg)
	}
	return Target{Kind: KindBead, ID: arg, Name: arg}, nil
}

// targetAt reads the target a worktree stands for off the branch it has checked
// out. known are the beads bd listed; without them a bead's worktree is merely a
// plain one, which still lists.
func (e Env) targetAt(w git.Worktree, known []beads.Bead) Target {
	// Only a branch the guess would name itself is a pull request, so that pr-007,
	// which it canonicalises to pr-7, does not stand for a worktree elsewhere.
	if t, err := e.guess(w.Branch); err == nil && t.Kind == KindPR && t.Name == w.Branch {
		return t
	}
	if id := e.beadID(w.Branch, known); id != "" {
		return Target{Kind: KindBead, ID: id, Name: id}
	}
	return Target{Kind: KindPlain, ID: w.Path, Name: cmp.Or(w.Branch, filepath.Base(w.Path))}
}

// beadID names the bead a branch belongs to: the longest known id owning it, so
// that a branch of one-two's does not fall to one.
func (e Env) beadID(branch string, known []beads.Bead) string {
	best := ""
	for _, b := range known {
		if len(b.ID) > len(best) && e.Config.Branch.Owns(b.ID, branch) {
			best = b.ID
		}
	}
	return best
}

// record is the listing's own bead for a target, or the zero bead where the
// listing did not name it and entering has to ask bd for it.
func record(known []beads.Bead, t Target) beads.Bead {
	if t.Kind != KindBead {
		return beads.Bead{}
	}
	for _, b := range known {
		if b.ID == t.ID {
			return b
		}
	}
	return beads.Bead{}
}

// matches reports whether a worktree is the one this target stands for,
// wherever it happens to sit. A ticket is keyed on the branch, less the branches
// a longer known id owns, which are that other ticket's; where bd named no ids
// nothing is longer and the branch alone settles it, as it does for the listing.
// A plain worktree, having no ticket to be named by, is only itself.
func (e Env) matches(t Target, w git.Worktree, known []beads.Bead) bool {
	switch t.Kind {
	case KindPR:
		return w.Branch == t.Name
	case KindPlain:
		return git.SameDir(w.Path, t.ID)
	default:
		return e.Config.Branch.Owns(t.ID, w.Branch) && len(e.beadID(w.Branch, known)) <= len(t.ID)
	}
}

// Candidate is one place to work, whether it was offered or asked for by name:
// a target, how it reads, and what the listing already settled, which entering
// it takes rather than ask again.
type Candidate struct {
	Target Target
	Label  string // bead or PR title, empty where no adapter named it
	Open   bool   // a worktree for it already exists

	path  string     // where that worktree sits
	ready bool       // bd listed this bead as ready to work
	bead  beads.Bead // the record the listing named, zero where none did
}

// Candidates lists what the repository offers to work on: every worktree git
// knows, then the open pull requests and ready beads without one. An adapter
// that will not answer costs its own rows and its own labels, never the
// worktrees.
func (e Env) Candidates() ([]Candidate, error) {
	worktrees, err := git.Linked(e.Repo)
	if err != nil {
		return nil, err
	}
	// The one listing serves both the branch matching and the labels.
	known, _ := beads.All(e.Repo)
	ready, _ := beads.Ready(e.Repo)
	pulls, _ := forge.Open(e.Repo, git.OriginURL(e.Repo))
	return e.candidates(worktrees, known, ready, pulls), nil
}

// candidates assembles the list from what the adapters answered, each of which
// may have answered nothing.
func (e Env) candidates(worktrees []git.Worktree, known, ready []beads.Bead, pulls []forge.PR) []Candidate {
	prTitles := make(map[string]string, len(pulls))
	for _, p := range pulls {
		prTitles[strconv.Itoa(p.Number)] = p.Title
	}

	out := make([]Candidate, 0, len(worktrees)+len(pulls)+len(ready))
	// Keyed on the name, which is what a target of any kind is retyped as, so one
	// place to work is offered once however it was found.
	open := make(map[string]bool, len(worktrees))
	for _, w := range worktrees {
		t := e.targetAt(w, known)
		c := Candidate{Target: t, Open: true, path: w.Path}
		open[t.Name] = true
		switch t.Kind {
		case KindBead:
			c.bead = record(known, t)
			c.Label = c.bead.Title
		case KindPR:
			c.Label = prTitles[t.ID]
		}
		out = append(out, c)
	}

	for _, p := range pulls {
		t := e.prTarget(strconv.Itoa(p.Number))
		if open[t.Name] {
			continue
		}
		out = append(out, Candidate{Target: t, Label: p.Title})
	}

	for _, b := range ready {
		// A ready id names a bead outright, so it skips the PR heuristics in Resolve.
		if !worktreeName.MatchString(b.ID) {
			continue
		}
		t := Target{Kind: KindBead, ID: b.ID, Name: b.ID}
		if open[t.Name] {
			continue
		}
		// From the ready listing rather than the known one, so what vetting reads is
		// the snapshot that called the bead ready in the first place.
		out = append(out, Candidate{Target: t, Label: b.Title, ready: true, bead: b})
	}
	return out
}

// located is where the worktree a target already has sits, or "" for a target
// without one. Several still matching is a repository arranged by hand: the
// shortest branch takes it, so it is settled on the branch and never on git's
// listing order.
func (e Env) located(t Target, worktrees []git.Worktree, known []beads.Bead) string {
	var found []git.Worktree
	for _, w := range worktrees {
		if e.matches(t, w, known) {
			found = append(found, w)
		}
	}
	if len(found) == 0 {
		return ""
	}
	return slices.MinFunc(found, func(a, b git.Worktree) int {
		return cmp.Or(cmp.Compare(len(a.Branch), len(b.Branch)), cmp.Compare(a.Branch, b.Branch))
	}).Path
}

// prTarget names a pull request by the branch its worktree checks out, which is
// what a row shows and what the user retypes.
func (e Env) prTarget(id string) Target {
	return Target{Kind: KindPR, ID: id, Name: e.Config.Branch.PullRequest(id)}
}

func (e Env) worktreePath(name string) string {
	return filepath.Join(e.Repo, e.Config.Worktree.Dir(), name)
}

// inspectAt gathers the state of a candidate the listing already answered for,
// so that neither git nor bd is asked again. An empty path is a target without a
// worktree; a zero bead one the listing could not name.
func (e Env) inspectAt(c Candidate) State {
	// Only creating a worktree needs a directory chosen for it; an existing one is
	// entered where git says it is, whatever the setting says today.
	s := State{Target: c.Target, Path: e.worktreePath(c.Target.Name), Ready: c.ready}

	if c.path != "" {
		s.Exists, s.Path = true, c.path
	}

	// A worktree that already exists is re-entered rather than created, and only
	// creating it needs the ticket, so bd is left unasked on the common path.
	if s.Exists || c.Target.Kind != KindBead {
		return s
	}
	if c.bead.ID != "" {
		s.Bead = c.bead
		return s
	}

	// The listing could not name this one: bd was unreachable, or the id is not
	// one it knows, which is the error a bad ticket name has to produce.
	b, err := beads.Show(e.Repo, c.Target.ID)
	if err != nil {
		s.TicketErr = err
		return s
	}
	s.Bead = b
	return s
}

// vet reports why the bead cannot be worked, or "" if it can. bd is asked only
// for an open bead nothing has already vouched for: a status that answers the
// question, and a listing that reported the bead ready, both cost no query.
func (e Env) vet(b beads.Bead, ready bool) (string, error) {
	if b.Status == "open" && !ready {
		list, err := beads.Ready(e.Repo)
		if err != nil {
			return "", err
		}
		ready = slices.ContainsFunc(list, func(r beads.Bead) bool { return r.ID == b.ID })
	}
	return vetBead(b, ready), nil
}

// vetBead reports why the bead cannot be worked, or "" if it can. ready says
// whether bd currently considers it unblocked; it is consulted only for open
// beads, so callers may pass false for any other status.
func vetBead(b beads.Bead, ready bool) string {
	switch {
	case b.Status == "deferred":
		return fmt.Sprintf("%s is unrefined; refine it first with /refine %s", b.ID, b.ID)
	case b.Status == "closed":
		return fmt.Sprintf("%s is already closed", b.ID)
	case b.Type == "epic":
		return fmt.Sprintf("%s is an epic; start one of its children (bd show %s --children)", b.ID, b.ID)
	// Ahead of the criteria check: /refine would move the bead out of the status
	// that is the actual blocker.
	case b.Status != "open" && b.Status != "in_progress":
		return fmt.Sprintf("%s is %s, not workable", b.ID, b.Status)
	case strings.TrimSpace(b.AcceptanceCriteria) == "":
		return fmt.Sprintf("%s has no acceptance criteria; refine it first with /refine %s", b.ID, b.ID)
	case b.Status == "open" && !ready:
		return fmt.Sprintf("%s is blocked by an open dependency", b.ID)
	}
	return ""
}
