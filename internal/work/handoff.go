package work

import (
	"cmp"
	"errors"
	"os"
	"os/exec"
	"syscall"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/git"
)

// Handoff is the last thing work does: replace itself with a command running
// inside the worktree.
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

// Shell renders the command an existing worktree is entered with.
func (e Env) Shell(s State, o Options) ([]string, error) {
	return e.Config.Open.Shell(values(s, o))
}

// Editor renders the command the worktree is opened in. An editor the
// environment does not name leaves the default command with nothing to run,
// which is how it is refused: a shell has a fallback every machine has, an
// editor has none worth opening someone's work in.
func (e Env) Editor(s State, o Options) ([]string, error) {
	return e.Config.Open.Editor(values(s, o))
}

// Diff renders the command the worktree's own work is shown with. The base is
// read here rather than in values, being open.diff's value alone.
func (e Env) Diff(s State, o Options) ([]string, error) {
	l := values(s, o)
	base, err := git.Base(s.Path)
	if err != nil {
		return nil, err
	}
	l.Base = base
	return e.Config.Open.Diff(l)
}

// Launch renders the command a fresh worktree opens on, which the target's kind
// chooses between.
func (e Env) Launch(s State, o Options) ([]string, error) {
	l := values(s, o)
	if s.Target.Kind == KindPR {
		l.Number = s.Target.ID
		return e.Config.Agent.StartPullRequest(l)
	}
	l.ID, l.Title = s.Target.ID, s.Bead.Title
	return e.Config.Agent.StartTicket(l)
}

// Resume renders the command that returns to the conversation the worktree
// carries, whatever kind of target it holds.
func (e Env) Resume(s State, o Options) ([]string, error) {
	return e.Config.Agent.ResumeSession(values(s, o))
}

// values are what every command renders with, whatever it is for. The
// environment is read here rather than per command, so which key may place what
// it named is the settings' allowlist to say and nothing else's.
func values(s State, o Options) config.Launch {
	return config.Launch{
		Name: s.Target.Name, Dir: s.Path, Model: o.Model, Effort: o.Effort,
		Shell:  cmp.Or(os.Getenv("SHELL"), "/bin/sh"),
		Editor: cmp.Or(os.Getenv("VISUAL"), os.Getenv("EDITOR")),
	}
}
