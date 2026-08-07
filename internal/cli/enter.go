package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/JHK/work-cli/internal/work"
)

// enter resolves the target, asks work to bring its worktree into being, and
// hands the terminal over to what came back.
func enter(o options, target string) error {
	env, err := work.Open(".")
	if err != nil {
		return err
	}

	opts := work.Options{Shell: o.shell, Editor: o.editor, NoClaim: o.noClaim, Model: o.model, Effort: o.effort}
	e, err := entry(env, target, opts)
	if err != nil {
		return err
	}
	// Only the shell is landed in with a prompt to paste into; a command has the
	// terminal to itself.
	if e.Shell {
		report(env, e.State, opts)
	}
	return e.Handoff.Exec()
}

// entry brings the target's worktree into being: an identifier names one
// outright, and without one the picker hands over a candidate.
func entry(env work.Env, target string, o work.Options) (work.Entry, error) {
	if target == "" {
		c, err := pick(env)
		if err != nil {
			return work.Entry{}, err
		}
		return env.EnterCandidate(c, o)
	}
	t, err := env.Resolve(target)
	if err != nil {
		return work.Entry{}, err
	}
	return env.Enter(t, o)
}

// report names what the worktree already carries, so the shell you land in
// starts with its conversations one paste away. Returning to one is a single
// line: the agent is run in the worktree, and picks the conversation up from
// there.
func report(env work.Env, s work.State, o work.Options) {
	if s.SessionsErr != nil {
		fmt.Fprintln(os.Stderr, "work: could not read session history:", s.SessionsErr)
		return
	}
	if len(s.Sessions) == 0 {
		fmt.Println("No prior Claude sessions here.")
		return
	}
	for _, sess := range s.Sessions {
		fmt.Println(" ", sess.Title)
	}
	resume, err := env.Resume(s, o)
	if err != nil {
		fmt.Fprintln(os.Stderr, "work:", err)
		return
	}
	fmt.Println("Return to the most recent with:", strings.Join(resume, " "))
}
