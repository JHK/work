// Package worktree holds the vocabulary the core and the systems behind its two
// seams both speak: the place a resolver makes of an identifier, the worktree an
// action is handed, and the command it opens on.
package worktree

import (
	"cmp"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// Place is one place to work, as the resolver that owns it describes it. The
// core reads Name and Branch; the rest is for whoever draws it or is handed it.
type Place struct {
	Source string // the resolver that answered for it, stamped by the core
	ID     string // what that resolver calls this place
	Name   string // what a row shows, what the user retypes, and the directory
	Branch string // what its worktree checks out, named by Prepare where creating names it
	Label  string // a title for the row, empty where nothing named one
}

// Open is a worktree the repository already has, as a resolver is shown one:
// enough to say whose it is, and no more.
type Open struct {
	Path   string
	Branch string // empty where the worktree is detached
}

// None reports whether there is no worktree in hand, the resolver being asked
// about an identifier alone.
func (o Open) None() bool { return o.Path == "" }

// Name is what a worktree goes by where nothing behind it names it: the branch
// it has checked out, or its directory where it is detached.
func (o Open) Name() string { return cmp.Or(o.Branch, filepath.Base(o.Path)) }

// Tree is a worktree that exists, which is the only thing an action is handed.
type Tree struct {
	Place
	Path    string
	Created bool // this run made it, rather than found it

	// By is the resolver that answered for the place. An action wanting more than a
	// Place carries declares the interface it needs and asserts this to it.
	By System
}

// System is a resolver or an action under the name it goes by: the name a
// [Place] is sourced to, an [action] key holds, and a flag spells.
type System interface {
	Name() string
}

// Drawn is a system that says how a screen marks the rows it answers for. One
// that says nothing is drawn unmarked.
type Drawn interface {
	System

	// Icon is the mark, one column wide.
	Icon() string
}

// Flagged is a system a flag on the command line spells: the name it answers to
// and the line --help shows for it. An opener's flag names it and an action's
// declines it; a system that spells none is reached by the settings alone.
type Flagged interface {
	System

	Flag() (name, usage string)
}

// Claimant is a system that knows its own identifiers by the way they are
// spelled, without reaching anything: it is the one question a system the
// settings left out is asked. A system that could only answer by asking claims
// nothing and does not implement this.
type Claimant interface {
	System

	// Claims is the place this system would make of an identifier spelled as one of
	// its own, and false where it is not spelled as one.
	Claims(id string) (Place, bool)
}

// Values are what a command renders with, keyed by the name a template places
// rather than by a field. A name nothing supplied is a command that cannot run;
// one supplied empty is a command element that drops out.
type Values map[string]string

// Merge takes in what another source supplied, leaving every name already set
// alone: the first source asked owns a name.
func (v Values) Merge(other Values) {
	for name, value := range other {
		if _, taken := v[name]; !taken {
			v[name] = value
		}
	}
}

// Source is a system that knows values a command may need. Every source is asked
// once, of a worktree that exists, and the answers are merged.
type Source interface {
	Supply(t Tree) (Values, error)
}

// ErrUnknown means a resolver does not answer for the identifier or open
// worktree it was shown, and the next resolver is asked about it. Every other
// error stops the run.
var ErrUnknown = errors.New("no system answers for it")

// Handoff is what a worktree opens on: work replaces itself with this command,
// running inside the worktree.
type Handoff struct {
	Dir string
	Run []string
}

// Exec hands the terminal over. It returns only on failure.
func (h Handoff) Exec() error {
	if len(h.Run) == 0 {
		return errors.New("nothing to run")
	}
	// Resolved before the chdir, so a failure leaves the process where it started.
	bin, err := exec.LookPath(h.Run[0])
	if err != nil {
		return err
	}
	if err := os.Chdir(h.Dir); err != nil {
		return err
	}
	return syscall.Exec(bin, h.Run, os.Environ())
}
