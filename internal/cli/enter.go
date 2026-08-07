package cli

import "github.com/JHK/work-cli/internal/work"

// enter resolves the target, asks work to bring its worktree into being, and
// hands the terminal over to what came back.
func enter(o options, target string) error {
	env, err := work.Open(".")
	if err != nil {
		return err
	}

	e, err := entry(env, target, work.Options{
		Shell: o.shell, Editor: o.editor, NoClaim: o.noClaim, Model: o.model, Effort: o.effort,
	})
	if err != nil {
		return err
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
