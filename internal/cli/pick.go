package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/run"
	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
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
func targeted(env work.Env, l listing, target string, resolve func(worktree.ID) (work.Candidate, error)) (work.Candidate, error) {
	if target != "" {
		return resolve(worktree.ID(target))
	}
	rows, _, err := l.rows(env)
	return pickFrom(env, l.saidWhenEmpty, rows, err)
}

// pickFrom puts one listing in front of the picker, refusing one left with no
// rows in the words its verb has for having none.
func pickFrom(env work.Env, saidWhenEmpty string, candidates []work.Candidate, err error) (work.Candidate, error) {
	if err != nil {
		return work.Candidate{}, err
	}
	if len(candidates) == 0 {
		return work.Candidate{}, errors.New(saidWhenEmpty)
	}
	i, err := choose(labels(env.Repo, candidates))
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

func whereabouts(repo worktree.Repo, c work.Candidate) string {
	if c.Main {
		return "(main worktree)"
	}
	if c.Path() == "" {
		return ""
	}
	where := writtenPath(repo, c.Path())
	// The tail is the name the row already carries, so what is left to say is the
	// directory the worktree sits in.
	if rest, endsWithName := strings.CutSuffix(where, string(filepath.Separator)+string(c.Name)); endsWithName {
		where = rest + string(filepath.Separator)
	}
	return "(" + where + ")"
}

// writtenPath is one worktree's path as a reader reads it, $HOME written as ~.
func writtenPath(repo worktree.Repo, path worktree.Path) string {
	if rel, ok := git.RelativeTo(path, worktree.Path(repo)); ok {
		return filepath.Join(filepath.Base(string(repo)), rel)
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if rel, ok := git.RelativeTo(path, worktree.Path(home)); ok {
			return filepath.Join("~", rel)
		}
	}
	return string(path)
}

// columns is where the fields behind the mark start. A field counts a row only
// where something behind it has to line up.
type columns struct{ name, where int }

func labels(repo worktree.Repo, candidates []work.Candidate) []string {
	var width columns
	where := make([]string, len(candidates))
	for i, c := range candidates {
		where[i] = whereabouts(repo, c)
		if where[i] != "" || c.Label != "" {
			width.name = max(width.name, utf8.RuneCountInString(string(c.Name)))
		}
		if c.Label != "" {
			width.where = max(width.where, utf8.RuneCountInString(where[i]))
		}
	}
	out := make([]string, len(candidates))
	for i, c := range candidates {
		out[i] = label(c, where[i], width)
	}
	return out
}

// label renders one candidate, making the ones with a worktree stand out. A row
// goes untitled where the resolver that answered for it named no title.
func label(c work.Candidate, where string, width columns) string {
	mark := " "
	if c.Open {
		mark = openMark
	}

	row := mark + " " + c.Icon + " " + padded(string(c.Name), width.name) + "  " + padded(where, width.where)
	tail := ""
	if about := string(c.Label); about != "" {
		tail = "  ·  " + about
	} else {
		row = strings.TrimRight(row, " ")
	}
	if c.Open {
		row = highlight + row + reset
	}
	return row + tail
}

func padded(text string, width int) string {
	return text + strings.Repeat(" ", max(0, width-utf8.RuneCountInString(text)))
}
