package cli

import (
	"cmp"
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

	openMark = "⎇"
	// unmarked keeps the column a resolver that named no icon leaves empty, so the
	// names line up.
	unmarked = " "

	// prompt is what the one question fzf is put reads as.
	prompt = "work> "
)

// offered is the place a verb was given, or the one its picker hands over where
// it was given none. nothing is how that verb says it has none to offer.
func offered(id, nothing string, list func() ([]work.Candidate, []error, error), resolve func(string) (work.Candidate, error)) (work.Candidate, error) {
	if id != "" {
		return resolve(id)
	}
	rows, refused, err := list()
	report(os.Stderr, refused)
	return pickFrom(nothing, rows, err)
}

// openWorktree is the worktree a verb that acts on one was named: an identifier
// resolves to it, and without one the picker offers what that verb can act on.
func openWorktree(env work.Env, target, nothing string, list func() ([]work.Candidate, error)) (work.Candidate, error) {
	return offered(target, nothing, func() ([]work.Candidate, []error, error) {
		rows, err := list()
		return rows, nil, err
	}, env.Resolve)
}

// pickFrom puts one listing in front of the picker, refusing one left with no
// rows in the words its verb has for having none.
func pickFrom(nothing string, candidates []work.Candidate, err error) (work.Candidate, error) {
	if err != nil {
		return work.Candidate{}, err
	}
	if len(candidates) == 0 {
		return work.Candidate{}, errors.New(nothing)
	}
	i, err := choose(labels(candidates))
	if err != nil {
		return work.Candidate{}, err
	}
	return candidates[i], nil
}

// choose puts the listing through fzf and returns the row chosen. The row index
// is the key, so nothing has to be parsed back out of the label.
func choose(rows []string) (int, error) {
	keyed := make([]string, len(rows))
	for i, r := range rows {
		keyed[i] = fmt.Sprintf("%d\t%s", i, r)
	}

	out, err := putThrough(strings.Join(keyed, "\n")+"\n",
		"--ansi", "--delimiter", "\t", "--with-nth", "2..")
	if err != nil {
		return 0, err
	}
	field, _, _ := strings.Cut(strings.TrimSpace(out), "\t")
	i, err := strconv.Atoi(field)
	if err != nil || i < 0 || i >= len(rows) {
		return 0, errCancelled
	}
	return i, nil
}

// putThrough runs fzf over a listing under the flags every screen work puts up
// shares, and hands back what it printed whether or not it came back cancelled.
// fzf exits 1 with no match and 130 when interrupted; anything else, a missing
// binary above all, is a failure the user has to be told about.
func putThrough(stdin string, args ...string) (string, error) {
	fzf := exec.Command("fzf", append([]string{"--height", "40%", "--reverse", "--prompt", prompt}, args...)...)
	fzf.Stdin = strings.NewReader(stdin)
	fzf.Stderr = os.Stderr
	out, err := fzf.Output()
	if err == nil {
		return string(out), nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && (exit.ExitCode() == 1 || exit.ExitCode() == 130) {
		return string(out), errCancelled
	}
	return string(out), fmt.Errorf("fzf: %w", err)
}

// ask puts one question with an answer already in it, standing in for the
// argument a verb was given only half of. It is the picker's fzf over a listing
// of nothing, where the query is the answer.
func ask(preset string) (string, error) {
	// An answer matches none of the nothing on offer, so it comes back cancelled and
	// is read off what was printed rather than off the status.
	out, err := putThrough("", "--print-query", "--query", preset)
	if err != nil && !errors.Is(err, errCancelled) {
		return "", err
	}
	answer, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
	if answer == "" {
		return "", errCancelled
	}
	return answer, nil
}

// column is where the titles line up: behind the widest name that has one. A
// worktree name is ASCII by construction, so its length is its width.
func column(candidates []work.Candidate) int {
	width := 0
	for _, c := range candidates {
		// An untitled row is not padded, so it does not set the column either.
		if c.Label != "" {
			width = max(width, len(c.Name))
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

// label renders one candidate, making the ones with a worktree stand out. A row
// goes unmarked or untitled where the resolver that answered for it named neither.
func label(c work.Candidate, width int) string {
	mark, icon := " ", cmp.Or(c.Icon, unmarked)
	if c.Open {
		mark = openMark
	}

	name, about := c.Name, c.Label
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
