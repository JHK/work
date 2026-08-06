package work

import (
	"cmp"
	"errors"
	"os"
	"os/exec"
	"syscall"
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

// Launch describes the session to hand off to.
type Launch struct {
	Resume string // session id to resume; empty starts a fresh session
	Name   string // session name, for a fresh session
	Prompt string
	Model  string
	Effort string
}

// SessionLaunch is the session a fresh worktree opens with: named for the
// target, opened on the skill that works it.
func (s State) SessionLaunch(model, effort string) Launch {
	l := Launch{Model: model, Effort: effort}
	switch s.Target.Kind {
	case KindPR:
		l.Name = s.Title()
		l.Prompt = "/code-review " + s.Target.ID
	default:
		l.Name = s.Target.ID + ": " + s.Title()
		l.Prompt = "/start " + s.Target.ID
	}
	return l
}

// Argv builds the command that starts the session.
func (l Launch) Argv() []string {
	argv := []string{"claude", "--permission-mode", "auto"}
	if l.Resume != "" {
		argv = append(argv, "--resume", l.Resume)
	} else if l.Name != "" {
		argv = append(argv, "--name", l.Name)
	}
	if l.Model != "" {
		argv = append(argv, "--model", l.Model)
	}
	if l.Effort != "" {
		argv = append(argv, "--effort", l.Effort)
	}
	if l.Prompt != "" {
		argv = append(argv, l.Prompt)
	}
	return argv
}
