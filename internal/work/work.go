// Package work is the core beneath every front end: it resolves an identifier
// to a target, reports what that target's worktree already carries, and takes
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

// Sessions lists the agent sessions a directory carries. The default
// implementation reads Claude Code's undocumented transcript layout.
type Sessions interface {
	List(dir string) ([]sessions.Session, error)
}

// Env holds the repository work operates on and the adapters it reaches
// through.
type Env struct {
	Repo     string
	Sessions Sessions
}

// Open finds the repository containing dir and wires the default adapters.
func Open(dir string) (Env, error) {
	repo, err := git.Root(dir)
	if err != nil {
		return Env{}, err
	}
	return Env{Repo: repo, Sessions: sessions.Claude{}}, nil
}

// State is everything knowable about a target on entry, less what an existing
// worktree makes moot. Every field is best effort, and the errors stay apart so
// a caller can ask whether the adapter it needs answered.
type State struct {
	Target      Target
	Path        string // where the worktree is, or where it would be created
	Exists      bool
	Sessions    []sessions.Session
	Bead        beads.Bead
	SessionsErr error // the session history could not be read
	TicketErr   error // bd could not answer for this target
}

// worktreesDir is where every target's worktree lives, relative to the repo.
const worktreesDir = ".worktrees"

var (
	prURL = regexp.MustCompile(`^(?:[a-z]+://[^/]+/)?[^/\s]+/[^/\s]+/pull/([0-9]+)(?:[/?#].*)?$`)
	// The name becomes a directory under .worktrees and an argument to bd and git,
	// so it may not traverse, and may not open with a dash.
	worktreeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
)

// prPrefix marks a worktree as holding a pull request rather than a bead.
const prPrefix = "pr-"

// Resolve maps an identifier to a target. A bare number or an
// .../<owner>/<repo>/pull/<n> URL is a pull request; anything else is a bead id.
func Resolve(arg string) (Target, error) {
	if arg == "" {
		return Target{}, errors.New("no target given")
	}

	n := ""
	if _, err := strconv.ParseUint(arg, 10, 32); err == nil {
		n = arg
	} else if m := prURL.FindStringSubmatch(arg); m != nil {
		n = m[1]
	} else if rest, ok := strings.CutPrefix(arg, prPrefix); ok {
		// The worktree's own name, so that what the picker shows can be retyped.
		n = rest
	}
	if n != "" {
		// Canonicalise, so that 7 and 007 name one worktree and one refspec.
		i, err := strconv.ParseUint(n, 10, 32)
		if err != nil || i == 0 {
			return Target{}, fmt.Errorf("%q is not a pull request number", arg)
		}
		return prTarget(strconv.FormatUint(i, 10)), nil
	}

	if !worktreeName.MatchString(arg) {
		return Target{}, fmt.Errorf("%q is not a usable worktree name", arg)
	}
	return Target{Kind: KindBead, ID: arg, Name: arg}, nil
}

// targetAt reads the target a worktree stands for off the branch it has checked
// out. ids are the bead ids bd knows; without them a bead's worktree is merely a
// plain one, which still lists.
func targetAt(w git.Worktree, ids []string) Target {
	// Only a branch Resolve would name itself is a pull request, so that pr-007,
	// which it canonicalises to pr-7, does not stand for a worktree elsewhere.
	if t, err := Resolve(w.Branch); err == nil && t.Kind == KindPR && t.Name == w.Branch {
		return t
	}
	if id := beadID(w.Branch, ids); id != "" {
		return Target{Kind: KindBead, ID: id, Name: id}
	}
	return Target{Kind: KindPlain, ID: w.Path, Name: cmp.Or(w.Branch, filepath.Base(w.Path))}
}

// beadID names the bead a branch belongs to: the longest known id owning it, so
// that a branch of one-two's does not fall to one.
func beadID(branch string, ids []string) string {
	best := ""
	for _, id := range ids {
		if len(id) > len(best) && owns(id, branch) {
			best = id
		}
	}
	return best
}

// owns reports whether a branch is a bead's. Ids and title slugs both contain
// dashes, so the branch is matched against the id and its separator rather than
// parsed apart.
func owns(id, branch string) bool {
	return branch == id || strings.HasPrefix(branch, id+"-")
}

// matches reports whether a worktree is the one this target stands for,
// wherever it happens to sit. A ticket is keyed on the branch; a plain worktree,
// having no ticket to be named by, is only itself.
func (t Target) matches(w git.Worktree) bool {
	switch t.Kind {
	case KindPR:
		return w.Branch == t.Name
	case KindPlain:
		return git.SameDir(w.Path, t.ID)
	default:
		return owns(t.ID, w.Branch)
	}
}

// Candidate is one place worth offering: a target, how it reads, and what the
// listing already settled, which entering it takes rather than ask again.
type Candidate struct {
	Target Target
	Label  string // bead or PR title, empty where no adapter named it
	Open   bool   // a worktree for it already exists

	path  string // where that worktree sits
	ready bool   // bd listed this bead as ready to work
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
	return candidates(worktrees, known, ready, pulls), nil
}

// candidates assembles the list from what the adapters answered, each of which
// may have answered nothing.
func candidates(worktrees []git.Worktree, known, ready []beads.Bead, pulls []forge.PR) []Candidate {
	ids := make([]string, len(known))
	titles := make(map[string]string, len(known))
	for i, b := range known {
		ids[i] = b.ID
		titles[b.ID] = b.Title
	}
	prTitles := make(map[string]string, len(pulls))
	for _, p := range pulls {
		prTitles[strconv.Itoa(p.Number)] = p.Title
	}

	out := make([]Candidate, 0, len(worktrees)+len(pulls)+len(ready))
	// Keyed on the name, which is what a target of any kind is retyped as, so one
	// place to work is offered once however it was found.
	open := make(map[string]bool, len(worktrees))
	for _, w := range worktrees {
		t := targetAt(w, ids)
		c := Candidate{Target: t, Open: true, path: w.Path}
		open[t.Name] = true
		switch t.Kind {
		case KindBead:
			c.Label = titles[t.ID]
		case KindPR:
			c.Label = prTitles[t.ID]
		}
		out = append(out, c)
	}

	for _, p := range pulls {
		t := prTarget(strconv.Itoa(p.Number))
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
		out = append(out, Candidate{Target: t, Label: b.Title, ready: true})
	}
	return out
}

// locate finds the worktree a target already has, wherever it sits.
func (e Env) locate(t Target) (string, bool) {
	worktrees, err := git.Linked(e.Repo)
	if err != nil {
		return "", false
	}
	for _, w := range worktrees {
		if t.matches(w) {
			return w.Path, true
		}
	}
	return "", false
}

func prTarget(id string) Target {
	return Target{Kind: KindPR, ID: id, Name: prPrefix + id}
}

func worktreePath(repo, name string) string {
	return filepath.Join(repo, worktreesDir, name)
}

// Inspect gathers the state of a target without changing anything.
func (e Env) Inspect(t Target) State {
	path, _ := e.locate(t)
	return e.inspectAt(t, path)
}

// inspectAt is Inspect for a target whose worktree has already been found, so
// that git is not asked again. An empty path is a target without one.
func (e Env) inspectAt(t Target, path string) State {
	// Only creating a worktree needs a directory chosen for it; an existing one is
	// entered where git says it is, .worktrees or not.
	s := State{Target: t, Path: worktreePath(e.Repo, t.Name)}

	if path != "" {
		s.Exists, s.Path = true, path
		if e.Sessions == nil {
			s.SessionsErr = errors.New("no session adapter")
		} else if list, err := e.Sessions.List(path); err != nil {
			s.SessionsErr = err
		} else {
			s.Sessions = list
		}
	}

	// A worktree that already exists is re-entered rather than created, and only
	// creating it needs the ticket, so bd is left unasked on the common path.
	if t.Kind == KindBead && !s.Exists {
		b, err := beads.Show(e.Repo, t.ID)
		if err != nil {
			s.TicketErr = err
			return s
		}
		s.Bead = b
	}
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

// Title names the target for a session and for display.
func (s State) Title() string {
	if s.Target.Kind == KindPR {
		return "PR #" + s.Target.ID
	}
	return s.Bead.Title
}
