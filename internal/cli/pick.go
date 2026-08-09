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

// pickFrom puts one listing in front of the picker.
func pickFrom(list func() ([]work.Candidate, error), none string) (work.Candidate, error) {
	candidates, err := list()
	if err != nil {
		return work.Candidate{}, err
	}
	if len(candidates) == 0 {
		return work.Candidate{}, errors.New(none)
	}
	return pick(candidates)
}

// pick offers a listing and returns the candidate chosen. It is the first of
// the two questions a moment can carry.
func pick(candidates []work.Candidate) (work.Candidate, error) {
	i, err := choose(labels(candidates), "work> ")
	if err != nil {
		return work.Candidate{}, err
	}
	return candidates[i], nil
}

// ask is the second question: which of the actions work says apply the worktree
// opens on. An action reads as the flag naming it, there being nothing else it
// is called.
func ask(offer []work.Action) (work.Action, error) {
	rows := make([]string, len(offer))
	for i, a := range offer {
		rows[i] = string(a)
	}
	i, err := choose(rows, "open> ")
	if err != nil {
		return work.ActionUnnamed, err
	}
	return offer[i], nil
}

// choose puts one question through fzf and returns the row chosen. The row index
// is the key, so nothing has to be parsed back out of the label.
func choose(rows []string, prompt string) (int, error) {
	keyed := make([]string, len(rows))
	for i, r := range rows {
		keyed[i] = fmt.Sprintf("%d\t%s", i, r)
	}

	fzf := exec.Command("fzf", "--ansi", "--height", "40%", "--reverse",
		"--delimiter", "\t", "--with-nth", "2..", "--prompt", prompt)
	fzf.Stdin = strings.NewReader(strings.Join(keyed, "\n") + "\n")
	fzf.Stderr = os.Stderr
	out, err := fzf.Output()
	if err != nil {
		// fzf exits 1 with no match and 130 when interrupted; anything else, a
		// missing binary above all, is a failure the user has to be told about.
		var exit *exec.ExitError
		if errors.As(err, &exit) && (exit.ExitCode() == 1 || exit.ExitCode() == 130) {
			return 0, errCancelled
		}
		return 0, fmt.Errorf("fzf: %w", err)
	}
	field, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\t")
	i, err := strconv.Atoi(field)
	if err != nil || i < 0 || i >= len(rows) {
		return 0, errCancelled
	}
	return i, nil
}

// column is where the titles line up: behind the widest name that has one. A
// worktree name is ASCII by construction, so its length is its width.
func column(candidates []work.Candidate) int {
	width := 0
	for _, c := range candidates {
		// An untitled row is not padded, so it does not set the column either.
		if c.Label != "" {
			width = max(width, len(c.Target.Name))
		}
	}
	return width
}

// labels renders the rows the picker offers.
func labels(candidates []work.Candidate) []string {
	width := column(candidates)
	out := make([]string, len(candidates))
	for i, c := range candidates {
		out[i] = label(c, width)
	}
	return out
}

// label renders one candidate, making the ones with a worktree stand out
// because re-entry is the common case. A row is titled by whichever adapter
// names that kind, and goes untitled where none answered.
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

	name, about := c.Target.Name, c.Label
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
