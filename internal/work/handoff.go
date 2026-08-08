package work

import (
	"cmp"
	"errors"
	"os"
	"os/exec"
	"syscall"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/sessions"
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
func (e Env) Shell(s State) ([]string, error) {
	return e.Config.Open.Shell(values(s))
}

// Editor renders the command the worktree is opened in. An editor the
// environment does not name leaves the default command with nothing to run,
// which is how it is refused: a shell has a fallback every machine has, an
// editor has none worth opening someone's work in.
func (e Env) Editor(s State) ([]string, error) {
	return e.Config.Open.Editor(values(s))
}

// Diff renders the command the worktree's own work is shown with. The base is
// read here rather than in values, being open.diff's value alone.
func (e Env) Diff(s State) ([]string, error) {
	l := values(s)
	base, err := git.Base(s.Path)
	if err != nil {
		return nil, err
	}
	l.Base = base
	return e.Config.Open.Diff(l)
}

// Launch renders the command a fresh worktree opens on, which the target's kind
// chooses between.
func (e Env) Launch(s State) ([]string, error) {
	l := values(s)
	switch s.Target.Kind {
	case KindPR:
		l.Number = s.Target.ID
		return e.Config.Agent.StartPullRequest(l)
	case KindPlain:
		// Nothing to prompt a session with: a bare worktree opens on one of its own.
		return e.Config.Agent.StartSession(l)
	}
	l.ID, l.Title = s.Target.ID, s.Bead.Title
	return e.Config.Agent.StartTicket(l)
}

// Agent renders the command an existing worktree is handed to, which what it
// already carries decides: no conversation starts one, a single one is returned
// to, and several reach the agent's own list.
func (e Env) Agent(s State) ([]string, error) {
	// Unset is claude, as an unset command is the compiled-in one.
	agent := e.Conversations
	if agent == nil {
		agent = sessions.Claude{}
	}
	list, err := agent.List(s.Path)
	if err != nil {
		return nil, err
	}
	l := values(s)
	if len(list) == 0 {
		return e.Config.Agent.StartSession(l)
	}
	if len(list) == 1 {
		l.Session = list[0].ID
	}
	return e.Config.Agent.ResumeSession(l)
}

// values are what every command renders with, whatever it is for. The
// environment is read here rather than per command, so which key may place what
// it named is the settings' allowlist to say and nothing else's.
func values(s State) config.Launch {
	return config.Launch{
		Name:   s.Target.Name,
		Dir:    s.Path,
		Shell:  cmp.Or(os.Getenv("SHELL"), "/bin/sh"),
		Editor: cmp.Or(os.Getenv("VISUAL"), os.Getenv("EDITOR")),
	}
}
