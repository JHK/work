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

	beadIcon  = "◆"
	prIcon    = "⇄"
	plainIcon = "◇"
	openMark  = "⎇"
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
	rows := labels(candidates)
	for i, l := range rows {
		rows[i] = fmt.Sprintf("%d\t%s", i, l)
	}

	fzf := exec.Command("fzf", "--ansi", "--height", "40%", "--reverse",
		"--delimiter", "\t", "--with-nth", "2..", "--prompt", "work> ")
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

// labels renders the rows, lining the titles up behind the widest name that has
// one. A worktree name is ASCII by construction, so its length is its width.
func labels(candidates []work.Candidate) []string {
	width := 0
	for _, c := range candidates {
		// An untitled row is not padded, so it does not set the column either.
		if title(c) != "" {
			width = max(width, len(c.Target.Name))
		}
	}
	out := make([]string, len(candidates))
	for i, c := range candidates {
		out[i] = label(c, width)
	}
	return out
}

// title says what a row is about: a PR row says so itself, a bead row is named
// by the tracker, which may not have answered, and a plain worktree has only
// the branch its name already shows.
func title(c work.Candidate) string {
	if c.Target.Kind == work.KindPR {
		return "PR review"
	}
	return c.Label
}

// label renders one candidate, making the ones with a worktree stand out
// because re-entry is the common case.
func label(c work.Candidate, width int) string {
	mark, icon := " ", beadIcon
	if c.Open {
		mark = openMark
	}
	switch c.Target.Kind {
	case work.KindPR:
		icon = prIcon
	case work.KindPlain:
		icon = plainIcon
	}

	name, about := c.Target.Name, title(c)
	if about != "" {
		name = fmt.Sprintf("%-*s", width, name)
	}
	row := mark + " " + icon + " " + name
	if c.Open {
		row = highlight + row + reset
	}
	if about != "" {
		row += "  ·  " + about
	}
	return row
}
