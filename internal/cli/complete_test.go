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

// The identifier completes to what the picker would offer, and never to a file
// name: not on a repository that will not answer, and not on a second word,
// there being only one identifier.
func TestCompleteIdentifier(t *testing.T) {
	listed := []work.Candidate{
		{Target: work.Target{Kind: work.KindBead, ID: "bd-1", Name: "bd-1"}, Label: "Do a thing"},
		{Target: work.Target{Kind: work.KindPR, ID: "7", Name: "pr-7"}, Label: "Review this"},
	}
	tests := []struct {
		name   string
		args   []string
		list   func() ([]work.Candidate, error)
		want   []string
		absent []string
	}{
		{"the listing", []string{""}, stub(listed, nil), []string{"bd-1\tDo a thing", "pr-7\tReview this"}, nil},
		// The subcommands still complete; only the listing's rows are lost.
		{"a repository that will not answer", []string{""}, stub(nil, errCancelled), []string{"init\tPrint the shell integration to source"}, []string{"bd-1\tDo a thing"}},
		{"a second identifier", []string{"bd-1", ""}, stub(listed, nil), nil, []string{"bd-1\tDo a thing", "pr-7\tReview this"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := complete(t, tt.list, tt.args...)
			for _, w := range tt.want {
				if !slices.Contains(rows(out), w) {
					t.Errorf("completing %q gave %q; want a %q row", tt.args, out, w)
				}
			}
			for _, a := range tt.absent {
				if slices.Contains(rows(out), a) {
					t.Errorf("completing %q gave %q; want no %q row", tt.args, out, a)
				}
			}
			assertNoFileComp(t, out)
		})
	}
}

// A flag value completes to what it accepts, and to nothing where the values
// are not work's to know. Neither falls back to file names.
func TestCompleteFlags(t *testing.T) {
	out := complete(t, stub(nil, nil), "--effort", "")
	for _, e := range efforts {
		if !slices.Contains(rows(out), e) {
			t.Errorf("completing --effort gave %q; want a %q row", out, e)
		}
	}
	assertNoFileComp(t, out)

	// The agent behind --model is about to be configurable, so nothing is offered.
	out = complete(t, stub(nil, nil), "--model", "")
	if len(rows(out)) != 0 {
		t.Errorf("completing --model gave %q; want nothing", out)
	}
	assertNoFileComp(t, out)
}

// The flag states the values it completes, so neither list can drift.
func TestEffortUsage(t *testing.T) {
	usage := command(stubVersion, nil, stub(nil, nil)).Flags().Lookup("effort").Usage
	if !strings.Contains(usage, strings.Join(efforts, "|")) {
		t.Errorf("--effort usage %q does not state %q", usage, efforts)
	}
}

// init prints the script, and the shell it names completes no further.
func TestInitFish(t *testing.T) {
	var out strings.Builder
	err := runTo([]string{"init", "fish"}, &out, func(options, string) error {
		t.Error("entered a worktree despite init")
		return nil
	})
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
	assertNoFileComp(t, complete(t, stub(nil, nil), "init", "fish", ""))
}

// cobra's own completion command is gone, so work init fish is the one door.
// Only running it tells: the tree carries that command only once Execute ran.
func TestNoCompletionCommand(t *testing.T) {
	err := run([]string{"completion", "fish"}, func(options, string) error {
		t.Error("entered a worktree despite completion")
		return nil
	})
	if err == nil {
		t.Error("work still offers a completion command")
	}
}

func stub(candidates []work.Candidate, err error) func() ([]work.Candidate, error) {
	return func() ([]work.Candidate, error) { return candidates, err }
}

// complete asks the command what it would offer, the way a shell does.
func complete(t *testing.T, list func() ([]work.Candidate, error), args ...string) string {
	t.Helper()
	var out strings.Builder
	cmd := command(stubVersion, func(options, string) error {
		t.Error("ran despite completing")
		return nil
	}, list)
	cmd.SetArgs(append([]string{"__complete"}, args...))
	cmd.SetOut(&out)
	// The generated script discards stderr, so what a shell reads is stdout alone.
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("__complete %q: %v", args, err)
	}
	return out.String()
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
