// Package worktree holds the vocabulary the core and the integrations behind
// its two seams both speak: the place a resolver makes of an identifier, the
// worktree an action is handed, and the command it opens on.
package worktree

import (
	"cmp"
	"errors"
	"path/filepath"
)

// The words the vocabulary is made of, one defined type each.
type (
	// IntegrationName is the name a resolver or an action goes by.
	IntegrationName string

	// ID is what the resolver that owns a place calls it.
	ID string

	// Name is what a row shows, what the user retypes, and the directory.
	Name string

	// Branch is what a worktree checks out.
	Branch string

	// Label is the title a row carries beside its name.
	Label string

	// Repo is a repository's main worktree.
	Repo string

	// Path is where a worktree sits.
	Path string

	// ValueName is a name a command template places.
	ValueName string
)

// GitSource is what a place no integration answered for is sourced to: a name
// of the user's own, or a worktree described out of git alone.
const GitSource IntegrationName = "git"

// Place is one place to work, as the resolver that owns it describes it. The
// core reads Name and Branch; the rest is for whoever draws it or is handed it.
type Place struct {
	Source IntegrationName // the resolver that answered for it, stamped by the core
	ID     ID
	Name   Name
	Branch Branch // what its worktree checks out, named by Prepare where creating names it
	Label  Label  // empty where nothing named one
}

// Open is a worktree the repository already has, as a resolver is shown one:
// enough to say whose it is, and no more.
type Open struct {
	Path   Path
	Branch Branch // empty where the worktree is detached
}

// None reports whether there is no worktree in hand, the resolver being asked
// about an identifier alone.
func (o Open) None() bool { return o.Path == "" }

// Name is what a worktree goes by where nothing behind it names it: the branch
// it has checked out, or its directory where it is detached.
func (o Open) Name() Name {
	return Name(cmp.Or(string(o.Branch), filepath.Base(string(o.Path))))
}

// Tree is a worktree that exists, which is the only thing an action is handed.
type Tree struct {
	Place
	Path    Path
	Created bool // this run made it, rather than found it

	// Values are what a command for this worktree renders with, assembled once
	// before the actions run.
	Values Values

	// By is the resolver that answered for the place. An action wanting more than a
	// Place carries declares the interface it needs and asserts this to it.
	By Integration
}

// Integration is a resolver or an action under the name it goes by, which is
// the name a [Place] is sourced to.
type Integration interface {
	Name() IntegrationName
}

// Values are what a command renders with, keyed by the name a template places
// rather than by a field. A name nothing supplied renders empty, which is a
// command element that drops out.
type Values map[ValueName]string

// Merge takes in another set of values, leaving every name already set alone: the
// first to set a name owns it.
func (v Values) Merge(other Values) {
	for name, value := range other {
		if _, taken := v[name]; !taken {
			v[name] = value
		}
	}
}

// The names a command may place.
const (
	SourceValue  ValueName = "Source"
	IDValue      ValueName = "ID"
	TitleValue   ValueName = "Title"
	NameValue    ValueName = "Name"
	DirValue     ValueName = "Dir"
	SubjectValue ValueName = "Subject"
)

// ValueNames are those names, in the order a listing of them reads. A name
// outside them is never supplied.
func ValueNames() []ValueName {
	return []ValueName{SourceValue, IDValue, TitleValue, NameValue, DirValue, SubjectValue}
}

// Supplier is an integration that knows values the core does not hold, the
// core's own names winning where both name one. It is asked once, of a worktree
// that exists, and ahead of the assembly, so [Tree.Values] is empty there.
type Supplier interface {
	Integration

	Supply(t Tree) (Values, error)
}

// ErrUnknown means a resolver does not answer for the identifier or open
// worktree it was shown, and the next resolver is asked about it. Every other
// error stops the run.
var ErrUnknown = errors.New("no integration answers for it")

// Handoff is what a worktree opens on: work replaces itself with this command,
// running inside the worktree. One naming no command is the worktree itself,
// which the front end answers with rather than running.
type Handoff struct {
	Dir Path
	Run []string
}

// Directory reports whether the answer is the worktree and nothing to run in it.
func (h Handoff) Directory() bool { return len(h.Run) == 0 }
