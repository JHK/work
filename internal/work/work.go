// Package work is the core beneath every front end: it resolves an identifier
// to a target, reports what that target's worktree already carries, provisions
// what is missing, and hands the shell off to a session.
package work

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/JHK/work-cli/internal/beads"
	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/sessions"
)

// Kind distinguishes the namespaces an identifier can land in.
type Kind int

const (
	KindBead Kind = iota
	KindPR
)

// Target is one resolved place to work.
type Target struct {
	Kind Kind
	ID   string // bead id, or the pull request number
	Name string // worktree directory name
	Path string // absolute worktree path
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
	Exists      bool
	Sessions    []sessions.Session
	Bead        beads.Bead
	Reason      string // why the bead cannot be worked; "" if it can, or is unknown
	SessionsErr error  // the session history could not be read
	TicketErr   error  // bd or gh could not answer for this target
}

// worktreesDir is where every target's worktree lives, relative to the repo.
const worktreesDir = ".worktrees"

var (
	prURL = regexp.MustCompile(`^(?:[a-z]+://[^/]+/)?[^/\s]+/[^/\s]+/pull/([0-9]+)(?:[/?#].*)?$`)
	// The name becomes a directory under .worktrees and an argument to bd, gh and
	// git, so it may not traverse, and may not open with a dash.
	worktreeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
)

// prPrefix marks a worktree as holding a pull request rather than a bead.
const prPrefix = "pr-"

// Resolve maps an identifier to a target. A bare number or an
// .../<owner>/<repo>/pull/<n> URL is a pull request; anything else is a bead id.
func Resolve(repo, arg string) (Target, error) {
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
		return prTarget(repo, strconv.FormatUint(i, 10)), nil
	}

	if !worktreeName.MatchString(arg) {
		return Target{}, fmt.Errorf("%q is not a usable worktree name", arg)
	}
	return Target{Kind: KindBead, ID: arg, Name: arg, Path: worktreePath(repo, arg)}, nil
}

// TargetAt reads back the target a worktree directory stands for, inverting the
// naming Resolve applies.
func TargetAt(repo, path string) (Target, bool) {
	dir := filepath.Join(repo, worktreesDir)
	if filepath.Dir(filepath.Clean(path)) != dir {
		return Target{}, false
	}
	name := filepath.Base(path)
	if rest, ok := strings.CutPrefix(name, prPrefix); ok {
		if _, err := strconv.ParseUint(rest, 10, 32); err == nil {
			return prTarget(repo, rest), true
		}
	}
	if !worktreeName.MatchString(name) {
		return Target{}, false
	}
	return Target{Kind: KindBead, ID: name, Name: name, Path: worktreePath(repo, name)}, true
}

// Candidate is one place worth offering: a target, and how it reads.
type Candidate struct {
	Target Target
	Label  string // bead or PR title, empty when the tracker could not say
	Open   bool   // a worktree for it already exists
}

// Candidates lists what the repository offers to work on: the targets that
// already have worktrees, then the ready beads that do not. A tracker that will
// not answer costs the titles and the ready beads, never the worktrees.
func (e Env) Candidates() ([]Candidate, error) {
	targets, err := e.Targets()
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(targets))
	for _, t := range targets {
		if t.Kind == KindBead {
			ids = append(ids, t.ID)
		}
	}
	titles, _ := beads.Titles(e.Repo, ids)

	out := make([]Candidate, 0, len(targets))
	open := make(map[string]bool, len(targets))
	for _, t := range targets {
		c := Candidate{Target: t, Open: true}
		if t.Kind == KindBead {
			open[t.ID] = true
			c.Label = titles[t.ID]
		}
		out = append(out, c)
	}

	ready, _ := beads.Ready(e.Repo)
	for _, b := range ready {
		if open[b.ID] {
			continue
		}
		// A ready id names a bead outright, so it skips the PR heuristics in Resolve.
		if !worktreeName.MatchString(b.ID) {
			continue
		}
		t := Target{Kind: KindBead, ID: b.ID, Name: b.ID, Path: worktreePath(e.Repo, b.ID)}
		out = append(out, Candidate{Target: t, Label: b.Title})
	}
	return out, nil
}

// Targets lists the targets the repository already has worktrees for.
func (e Env) Targets() ([]Target, error) {
	paths, err := git.Worktrees(e.Repo)
	if err != nil {
		return nil, err
	}
	var out []Target
	for _, p := range paths {
		if t, ok := TargetAt(e.Repo, p); ok {
			out = append(out, t)
		}
	}
	return out, nil
}

func prTarget(repo, id string) Target {
	return Target{Kind: KindPR, ID: id, Name: prPrefix + id, Path: worktreePath(repo, prPrefix+id)}
}

func worktreePath(repo, name string) string {
	return filepath.Join(repo, worktreesDir, name)
}

// Inspect gathers the state of a target without changing anything.
func (e Env) Inspect(t Target) State {
	s := State{Target: t}

	// A directory git does not know as a worktree is a leftover, not a place to
	// work: entering it would put the session on the main checkout's branch.
	if git.IsWorktree(e.Repo, t.Path) {
		s.Exists = true
		if e.Sessions == nil {
			s.SessionsErr = errors.New("no session adapter")
		} else if list, err := e.Sessions.List(t.Path); err != nil {
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
		s.Reason, err = e.vet(b)
		if err != nil {
			s.TicketErr = err
		}
	}
	return s
}

// vet reports why the bead cannot be worked, or "" if it can. The ready set is
// only consulted for open beads, so a status that already answers the question
// costs no query.
func (e Env) vet(b beads.Bead) (string, error) {
	if b.Status != "open" {
		return vetBead(b, nil), nil
	}
	list, err := beads.Ready(e.Repo)
	if err != nil {
		return "", err
	}
	ready := make(map[string]bool, len(list))
	for _, r := range list {
		ready[r.ID] = true
	}
	return vetBead(b, ready), nil
}

// vet reports why the bead cannot be worked, or "" if it can. ready holds the
// ids bd currently considers unblocked; it is consulted only for open beads, so
// callers may leave it nil for any other status.
func vetBead(b beads.Bead, ready map[string]bool) string {
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
	case b.Status == "open" && !ready[b.ID]:
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
