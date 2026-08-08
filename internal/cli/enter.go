package cli

import "github.com/JHK/work-cli/internal/work"

// enter resolves the target, asks work to bring its worktree into being, and
// hands the terminal over to what came back.
func enter(o options, target string) error {
	env, err := work.Open(".")
	if err != nil {
		return err
	}

	e, err := entry(env, target, work.Options{Action: o.action(), NoClaim: o.noClaim})
	if err != nil {
		return err
	}
	return e.Handoff.Exec()
}

// entry brings the target's worktree into being: an identifier names the place
// to work outright, and without one the picker hands one over.
func entry(env work.Env, target string, o work.Options) (work.Entry, error) {
	var c work.Candidate
	var err error
	if target == "" {
		c, err = pick(env)
	} else {
		c, err = env.Resolve(target)
	}
	if err != nil {
		return work.Entry{}, err
	}
	return env.Enter(c, o)
}
