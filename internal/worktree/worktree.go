// Package worktree holds the vocabulary the core and the systems behind its two
// seams both speak: the place a resolver makes of an identifier, the worktree an
// action is handed, the command it opens on, and the one answer either seam
// gives that the core acts on rather than reports.
//
// It is a leaf, so an implementation speaks the core's vocabulary without
// importing the core, and the interfaces are declared where they are consumed.
package worktree

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// Place is one place to work, as the resolver that owns it describes it. The
// core reads Name and Branch, needing a directory to make and a branch to check
// out; the rest is for whoever draws it or is handed it.
type Place struct {
	// Source is the resolver that answered for it, under the name it goes by. The
	// core stamps it from the resolver it asked, so a resolver leaves it alone and
	// the name a system goes by is spelled once, by [System.Name].
	Source string
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

// None reports whether there is no worktree in hand. git always names a path, so a
// resolver shown none is being asked about an identifier alone.
func (o Open) None() bool { return o.Path == "" }

// Tree is a worktree that exists, which is the only thing an action is handed.
type Tree struct {
	Place
	Path    string
	Created bool // this run made it, rather than found it

	// By is the resolver that answered for the place. An action wanting more than
	// a Place carries declares the interface it needs and asserts this to it, so
	// what a tracker can tell an agent is between those two and reaches the core as
	// neither an import nor a switch.
	By System
}

// System is a resolver or an action under the name it goes by: the name a
// [Place] is sourced to, an [action] key holds, and a flag spells.
type System interface {
	Name() string
}

// Drawn is a system that says how a screen marks the rows it answers for. How a
// row reads is then the implementation's, and a system that says nothing is
// drawn unmarked rather than drawn by a front end that knows who it is.
type Drawn interface {
	System

	// Icon is the mark, one column wide.
	Icon() string
}

// Flagged is a system a flag on the command line spells: the name it answers to
// and the line --help shows for it. Which way the flag runs is which seam the
// system sits on, one action opening the worktree and every other one running:
// an opener's flag names it, and an action's declines it.
//
// A system that spells none is reached by the settings and the screen alone.
type Flagged interface {
	System

	Flag() (name, usage string)
}

// Claimant is a system that knows its own identifiers by the way they are
// spelled: a pull request URL is a forge's whether or not the forge can be
// reached, where no tracker can tell a ticket id from a typo without asking the
// tracker.
//
// It is the one question a system the settings left out is asked, so answering it
// may reach nothing that switching the system off was there to stop. A system that
// could only answer by asking claims nothing and declares no such thing: an
// identifier that would merely have been left to it is refused as one nothing
// answers for rather than attributed to it.
type Claimant interface {
	System

	// Claims is the place this system would make of an identifier spelled as one of
	// its own, and false where it is not spelled as one.
	Claims(id string) (Place, bool)
}

// Values are what a command renders with, keyed by the name a template places
// rather than by a field: which values exist is what the systems wired together
// happen to know between them, and not something the code settles in advance. A
// name nothing supplied is a command that cannot run; one supplied empty is a
// command element that drops out.
type Values map[string]string

// Merge takes in what another source supplied. The first source asked owns a
// name, so the resolver that answered for the place is asked ahead of the ambient
// ones and its answer is the one a command sees.
func (v Values) Merge(other Values) {
	for name, value := range other {
		if _, taken := v[name]; !taken {
			v[name] = value
		}
	}
}

// Source is a system that knows values a command may need. Every source is asked
// and the answers are merged, so a value reaches a command from whichever system
// knows it without either of them naming the other: what the beads resolver can
// tell a session about a ticket is the same arrangement as what the environment
// can tell one about an editor.
//
// A worktree with no path is one that does not exist yet. A source that can only
// answer from inside a worktree supplies nothing for it, and the command it would
// have fed is judged on the values that did arrive.
type Source interface {
	Supply(t Tree) (Values, error)
}

// ErrUnknown is the one answer a resolver gives that the core acts on rather
// than reports: an identifier or an open worktree this resolver does not answer
// for, which the next resolver is then asked about. Every other error stops the
// run, a tracker that cannot be reached above all.
var ErrUnknown = errors.New("no system answers for it")

// ErrAbsent is the same bit on the far seam: the tool this action would hand the
// worktree to is not there. The screen leaves such an action out, and naming one
// outright is refused before anything is created. Every other error is a tool
// that ran and failed, which is reported as it stands.
//
// Unlike [ErrUnknown], which the core reads and never reports, this one reaches
// the user wherever a flag named the action: an action carries it with [Absent]
// rather than by wrapping, so its own reason is what is read.
var ErrAbsent = errors.New("nothing here to hand the worktree to")

// Absent carries that bit on a refusal of the action's own, which is what the
// user is told: the sentinel is what the core reads and never a prose prefix on
// the reason the action gave.
func Absent(err error) error { return absent{err} }

type absent struct{ err error }

func (a absent) Error() string        { return a.err.Error() }
func (a absent) Is(target error) bool { return target == ErrAbsent }
func (a absent) Unwrap() error        { return a.err }

// Handoff is what a worktree opens on: work replaces itself with this command,
// running inside the worktree. Rendering one is free of consequence; running it
// is the last thing the process does.
type Handoff struct {
	Dir string
	Run []string
}

// Exec hands the terminal over. It returns only on failure.
func (h Handoff) Exec() error {
	if len(h.Run) == 0 {
		return errors.New("nothing to run")
	}
	// Resolved before the chdir, so a failure here leaves the process where it
	// started.
	bin, err := exec.LookPath(h.Run[0])
	if err != nil {
		return err
	}
	if err := os.Chdir(h.Dir); err != nil {
		return err
	}
	return syscall.Exec(bin, h.Run, os.Environ())
}
