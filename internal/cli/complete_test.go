package cli

import (
	"io"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/work"
)

// A row completes to what the user would retype, and says what it is in the
// column beside it.
func TestCompletions(t *testing.T) {
	got := completions([]work.Candidate{
		{Target: work.Target{Kind: work.KindBead, ID: "bd-1", Name: "bd-1"}, Label: "Do a thing", Open: true},
		{Target: work.Target{Kind: work.KindPR, ID: "7", Name: "pr-7"}, Label: "Review this"},
		{Target: work.Target{Kind: work.KindPlain, ID: "/elsewhere", Name: "spike"}, Open: true},
	})
	want := []string{"bd-1\tDo a thing", "pr-7\tReview this", "spike"}
	if !slices.Equal(got, want) {
		t.Errorf("completions() = %q; want %q", got, want)
	}
}

// The bare position offers the verbs and nothing else: the identifier is
// switch's own completion now, and a file name is never one either.
func TestCompleteBarePosition(t *testing.T) {
	listed := []work.Candidate{
		{Target: work.Target{Kind: work.KindBead, ID: "bd-1", Name: "bd-1"}, Label: "Do a thing"},
		{Target: work.Target{Kind: work.KindPR, ID: "7", Name: "pr-7"}, Label: "Review this"},
	}
	out := complete(t, front{candidates: stub(listed, nil)}, "")
	offered := names(rows(out))
	for _, verb := range []string{"switch", "add", "remove", "list", "init"} {
		if !slices.Contains(offered, verb) {
			t.Errorf("completing the bare position gave %q; want a %s row", offered, verb)
		}
	}
	for _, absent := range []string{"bd-1", "pr-7"} {
		if slices.Contains(offered, absent) {
			t.Errorf("completing the bare position gave %q; want no %s row", offered, absent)
		}
	}
	assertNoFileComp(t, out)
}

// A tab press after switch offers what the picker offers, and never a file
// name: not on a repository that will not answer, and not on a second word,
// there being only one identifier.
func TestCompleteSwitch(t *testing.T) {
	listed := []work.Candidate{
		{Target: work.Target{Kind: work.KindBead, ID: "bd-1", Name: "bd-1"}, Label: "Do a thing"},
		{Target: work.Target{Kind: work.KindPlain, ID: "/elsewhere", Name: "spike"}, Open: true},
	}
	worktrees := []work.Candidate{{Target: work.Target{Kind: work.KindPR, ID: "7", Name: "pr-7"}, Label: "Review this"}}

	out := complete(t, front{candidates: stub(listed, nil), worktrees: stub(worktrees, nil)}, "switch", "")
	want := []string{"bd-1\tDo a thing", "spike"}
	if !slices.Equal(rows(out), want) {
		t.Errorf("completing switch gave %q; want %q", rows(out), want)
	}
	assertNoFileComp(t, out)

	// There is one identifier, so a second word completes to nothing.
	second := complete(t, front{candidates: stub(listed, nil)}, "switch", "bd-1", "")
	if got := rows(second); got != nil {
		t.Errorf("completing a second identifier after switch gave %q; want nothing", got)
	}
	assertNoFileComp(t, second)

	// A repository that will not answer costs the rows, not the shell.
	silent := complete(t, front{candidates: stub(nil, errCancelled)}, "switch", "")
	if got := rows(silent); got != nil {
		t.Errorf("completing switch on a silent repository gave %q; want nothing", got)
	}
	assertNoFileComp(t, silent)
}

// A tab press after remove offers the worktrees and nothing else: not the
// tickets and pull requests the identifier completes to, and not a file name.
func TestCompleteRemove(t *testing.T) {
	worktrees := []work.Candidate{
		{Target: work.Target{Kind: work.KindBead, ID: "bd-1", Name: "bd-1"}, Label: "Do a thing", Open: true},
		{Target: work.Target{Kind: work.KindPlain, ID: "/elsewhere", Name: "spike"}, Open: true},
	}
	elsewhere := []work.Candidate{{Target: work.Target{Kind: work.KindPR, ID: "7", Name: "pr-7"}, Label: "Review this"}}

	out := complete(t, front{worktrees: stub(worktrees, nil), candidates: stub(elsewhere, nil)}, "remove", "")
	want := []string{"bd-1\tDo a thing", "spike"}
	if !slices.Equal(rows(out), want) {
		t.Errorf("completing remove gave %q; want %q", rows(out), want)
	}
	assertNoFileComp(t, out)
}

// A tab press after add offers nothing: the name is new, so no listing has it,
// and a file name is not one either.
func TestCompleteAdd(t *testing.T) {
	listed := []work.Candidate{{Target: work.Target{Kind: work.KindBead, ID: "bd-1", Name: "bd-1"}, Label: "Do a thing"}}

	out := complete(t, front{candidates: stub(listed, nil), worktrees: stub(listed, nil)}, "add", "")
	if got := rows(out); got != nil {
		t.Errorf("completing add gave %q; want nothing", got)
	}
	assertNoFileComp(t, out)
}

// A tab press after list offers nothing: it takes no argument, and a file name
// is not one either.
func TestCompleteList(t *testing.T) {
	listed := []work.Candidate{{Target: work.Target{Kind: work.KindBead, ID: "bd-1", Name: "bd-1"}, Label: "Do a thing", Open: true}}

	out := complete(t, front{candidates: stub(listed, nil), worktrees: stub(listed, nil)}, "list", "")
	if got := rows(out); got != nil {
		t.Errorf("completing list gave %q; want nothing", got)
	}
	assertNoFileComp(t, out)
}

// Completion offers the declared flags, so a flag work no longer declares is a
// flag it no longer offers.
func TestGoneFlags(t *testing.T) {
	f := command(stubVersion, front{}).Flags()
	for _, name := range []string{"model", "effort", "delete", "force", "create"} {
		if f.Lookup(name) != nil {
			t.Errorf("--%s is still declared", name)
		}
	}
}

// init prints the script, and the shell it names completes no further.
func TestInitFish(t *testing.T) {
	var out strings.Builder
	err := execute(t, []string{"init", "fish"}, &out, front{})
	if err != nil {
		t.Fatalf("work init fish: %v", err)
	}
	if !strings.Contains(out.String(), "complete -c work") {
		t.Errorf("work init fish printed %q; want a fish completion script", out.String())
	}
	// Both variants register the same completions; only this one asks for titles.
	if strings.Contains(out.String(), cobra.ShellCompNoDescRequestCmd) {
		t.Error("work init fish printed a script that asks for no descriptions")
	}
	assertNoFileComp(t, complete(t, front{}, "init", "fish", ""))
}

// cobra's own completion command is gone, so work init fish is the one door and
// the word is a worktree name like any other. Only running it tells: the tree
// carries that command only once Execute ran.
func TestNoCompletionCommand(t *testing.T) {
	var target string
	err := execute(t, []string{"completion"}, io.Discard, front{enter: func(_ options, id string) error {
		target = id
		return nil
	}})
	if err != nil || target != "completion" {
		t.Errorf(`work completion = %q, %v; want the worktree named completion`, target, err)
	}
}

func stub(candidates []work.Candidate, err error) func() ([]work.Candidate, error) {
	return func() ([]work.Candidate, error) { return candidates, err }
}

// complete asks the command what it would offer, the way a shell does. The
// generated script discards stderr, so what a shell reads is stdout alone.
func complete(t *testing.T, f front, args ...string) string {
	t.Helper()
	var out strings.Builder
	if err := execute(t, append([]string{"__complete"}, args...), &out, f); err != nil {
		t.Fatalf("__complete %q: %v", args, err)
	}
	return out.String()
}

// names is what each row completes to, the description column dropped: what a
// row says of itself is the command's to word, not this package's.
func names(rows []string) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i], _, _ = strings.Cut(r, "\t")
	}
	return out
}

// rows is the completion output less the directive line cobra ends with.
func rows(out string) []string {
	var got []string
	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		if line != "" && !strings.HasPrefix(line, ":") {
			got = append(got, line)
		}
	}
	return got
}

// assertNoFileComp reads the directive line the shell acts on, not the prose
// cobra prints beside it, which a directive carrying other bits also spells.
func assertNoFileComp(t *testing.T, out string) {
	t.Helper()
	want := ":" + strconv.Itoa(int(cobra.ShellCompDirectiveNoFileComp))
	if !slices.Contains(strings.Split(strings.TrimSpace(out), "\n"), want) {
		t.Errorf("completion %q may fall back to file names", out)
	}
}
