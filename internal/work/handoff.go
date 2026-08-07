package work

import (
	"cmp"
	"errors"
	"os"
	"os/exec"
	"syscall"

	"github.com/JHK/work-cli/internal/config"
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

// Shell names the interactive shell to drop into.
func Shell() []string {
	return []string{cmp.Or(os.Getenv("SHELL"), "/bin/sh")}
}

// Editor names the editor to open the worktree in. An unset one is refused
// rather than guessed at: a shell has a fallback every machine has, an editor
// has none worth opening someone's work in.
func Editor(dir string) ([]string, error) {
	// Whatever the editor makes of the terminal it is handed is its own business,
	// so a terminal and a GUI editor are invoked alike.
	editor := cmp.Or(os.Getenv("VISUAL"), os.Getenv("EDITOR"))
	if editor == "" {
		return nil, errors.New("no editor to open: set $VISUAL or $EDITOR")
	}
	return []string{editor, dir}, nil
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

// values are what every command renders with, whatever it is for.
func values(s State, o Options) config.Launch {
	return config.Launch{Name: s.Target.Name, Dir: s.Path, Model: o.Model, Effort: o.Effort}
}
