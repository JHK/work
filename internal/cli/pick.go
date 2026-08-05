package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/JHK/work-cli/internal/work"
)

const (
	highlight = "\x1b[1;92m"
	reset     = "\x1b[0m"
)

// pick offers what the repository has to work on and returns the target chosen.
// It stands in until the screen replaces it.
func pick(env work.Env) (work.Target, error) {
	candidates, err := env.Candidates()
	if err != nil {
		return work.Target{}, err
	}
	if len(candidates) == 0 {
		return work.Target{}, errors.New("no worktrees or ready beads")
	}

	// The row index is the key, so nothing has to be parsed back out of the label.
	rows := make([]string, len(candidates))
	for i, c := range candidates {
		rows[i] = fmt.Sprintf("%d\t%s", i, label(c))
	}

	fzf := exec.Command("fzf", "--ansi", "--height", "40%", "--reverse",
		"--delimiter", "\t", "--with-nth", "2..", "--prompt", "work> ",
		"--select-1", "--exit-0")
	fzf.Stdin = strings.NewReader(strings.Join(rows, "\n") + "\n")
	fzf.Stderr = os.Stderr
	out, err := fzf.Output()
	if err != nil {
		// fzf exits 1 with no match and 130 when interrupted; anything else, a
		// missing binary above all, is a failure the user has to be told about.
		var exit *exec.ExitError
		if errors.As(err, &exit) && (exit.ExitCode() == 1 || exit.ExitCode() == 130) {
			return work.Target{}, errCancelled
		}
		return work.Target{}, fmt.Errorf("fzf: %w", err)
	}
	field, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\t")
	i, err := strconv.Atoi(field)
	if err != nil || i < 0 || i >= len(candidates) {
		return work.Target{}, errCancelled
	}
	return candidates[i].Target, nil
}

// label renders a candidate, making the ones with a worktree stand out because
// re-entry is the common case.
func label(c work.Candidate) string {
	if !c.Open {
		return "  " + c.Label
	}
	about := c.Label
	if c.Target.Kind == work.KindPR {
		about = "PR review"
	}
	return fmt.Sprintf("%s⎇ %s%s  ·  %s", highlight, c.Target.Name, reset, about)
}
