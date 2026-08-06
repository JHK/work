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

	var t work.Target
	if target == "" {
		t, err = pick(env)
	} else {
		t, err = work.Resolve(target)
	}
	if err != nil {
		return err
	}

	e, err := env.Enter(t, work.Options{Shell: o.shell, Model: o.model, Effort: o.effort})
	if err != nil {
		return err
	}
	if !e.Launched {
		report(e.State)
	}
	return e.Handoff.Exec()
}

// report surfaces what the worktree already carries, so the shell you land in
// starts with its sessions one paste away.
func report(s work.State) {
	if s.SessionsErr != nil {
		fmt.Fprintln(os.Stderr, "work: could not read session history:", s.SessionsErr)
		return
	}
	if len(s.Sessions) == 0 {
		fmt.Println("No prior Claude sessions here. Start one with:", commandLine(work.Launch{}))
		return
	}
	for _, sess := range s.Sessions {
		fmt.Printf("  %s\n    %s\n", sess.Title, commandLine(work.Launch{Resume: sess.ID}))
	}
}

// commandLine renders a launch as the line the user would type.
func commandLine(l work.Launch) string {
	return strings.Join(l.Argv(), " ")
}
