package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/JHK/work-cli/internal/run"
	"github.com/JHK/work-cli/internal/work"
)

const (
	highlight = "\x1b[1;92m"
	reset     = "\x1b[0m"

	openMark = "⎇"

	prompt = "work> "
)

// listing is what a verb has to offer: where its rows come from, and the words
// it has for having none.
type listing struct {
	saidWhenEmpty string
	rows          func(work.Env) ([]work.Candidate, []error, error)
}

// The listing each verb offers. remove and move share the rows and not the
// words.
var (
	workable  = listing{"nothing to work on", work.Env.Candidates}
	enterable = listing{"no worktree to switch to", rowsAlone(work.Env.Enterable)}
	addable   = listing{"nothing left to add", work.Env.Addable}
	removable = listing{"no worktree to remove", rowsAlone(work.Env.Removable)}
	movable   = listing{"no worktree to move", rowsAlone(work.Env.Removable)}
)

// rowsAlone puts a source that names no refusals in the shape one that names
// them takes.
func rowsAlone(list func(work.Env) ([]work.Candidate, error)) func(work.Env) ([]work.Candidate, []error, error) {
	return func(env work.Env) ([]work.Candidate, []error, error) {
		rows, err := list(env)
		return rows, nil, err
	}
}

// targeted is the place a verb was given, or the one its listing hands over where
// it was given none.
func targeted(env work.Env, l listing, target string, resolve func(string) (work.Candidate, error)) (work.Candidate, error) {
	if target != "" {
		return resolve(target)
	}
	rows, _, err := l.rows(env)
	return pickFrom(l.saidWhenEmpty, rows, err)
}

// pickFrom puts one listing in front of the picker, refusing one left with no
// rows in the words its verb has for having none.
func pickFrom(saidWhenEmpty string, candidates []work.Candidate, err error) (work.Candidate, error) {
	if err != nil {
		return work.Candidate{}, err
	}
	if len(candidates) == 0 {
		return work.Candidate{}, errors.New(saidWhenEmpty)
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

// putThrough runs fzf under the flags every screen shares. fzf exits 1 with no
// match and 130 when interrupted; anything else, a missing binary above all, is
// a failure.
func putThrough(stdin string, args ...string) (string, error) {
	fzf := run.Command("", "fzf", append([]string{"--height", "40%", "--reverse", "--prompt", prompt}, args...)...)
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
// argument a verb was given only half of.
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

// columnWidth is where the titles line up: behind the widest name that has one. A
// worktree name is ASCII by construction, so its length is its width.
func columnWidth(candidates []work.Candidate) int {
	width := 0
	for _, c := range candidates {
		if c.Label != "" {
			width = max(width, len(c.Name))
		}
	}
	return width
}

func labels(candidates []work.Candidate) []string {
	width := columnWidth(candidates)
	out := make([]string, len(candidates))
	for i, c := range candidates {
		out[i] = label(c, width)
	}
	return out
}

// label renders one candidate, making the ones with a worktree stand out. A row
// goes untitled where the resolver that answered for it named no title.
func label(c work.Candidate, width int) string {
	mark := " "
	if c.Open {
		mark = openMark
	}

	name, about := c.Name, c.Label
	if about != "" {
		name = fmt.Sprintf("%-*s", width, name)
	}
	row := mark + " " + c.Icon + " " + name
	if c.Open {
		row = highlight + row + reset
	}
	if about != "" {
		row += "  ·  " + about
	}
	return row
}
