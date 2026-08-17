package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/shim"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// A row completes to what the user would retype, and says what it is in the
// column beside it.
func TestCompletions(t *testing.T) {
	got := completions([]work.Candidate{
		{Place: worktree.Place{Source: "beads", ID: "bd-1", Name: "bd-1", Label: "Do a thing"}, Open: true},
		{Place: worktree.Place{Source: "github", ID: "7", Name: "pr-7", Label: "Review this"}},
		{Place: worktree.Place{Source: "plain", ID: "/elsewhere", Name: "spike"}, Open: true},
	})
	want := []string{"bd-1\tDo a thing", "pr-7\tReview this", "spike"}
	if !slices.Equal(got, want) {
		t.Errorf("completions() = %q; want %q", got, want)
	}
}

// The bare position offers the verbs and nothing else: the identifier is go's
// own completion now, and a file name is never one either.
func TestCompleteBarePosition(t *testing.T) {
	listed := []work.Candidate{
		{Place: worktree.Place{Source: "beads", ID: "bd-1", Name: "bd-1", Label: "Do a thing"}},
		{Place: worktree.Place{Source: "github", ID: "7", Name: "pr-7", Label: "Review this"}},
	}
	out := complete(t, front{reach: offering[opens]{list: stub(listed, nil)}}, "")
	offered := names(rows(out))
	for _, verb := range []string{"go", "switch", "add", "remove", "move", "list", "init"} {
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

// A tab press after a verb offers that verb's own listing and nothing else: not
// another verb's rows, and never a file name. Past the position it takes, which
// for move is the destination, it offers nothing at all.
func TestEachVerbCompletesFromItsOwnListing(t *testing.T) {
	own := []work.Candidate{
		{Place: worktree.Place{Source: "beads", ID: "bd-1", Name: "bd-1", Label: "Do a thing"}, Open: true},
		{Place: worktree.Place{Source: "plain", ID: "/elsewhere", Name: "spike"}, Open: true},
	}
	want := []string{"bd-1\tDo a thing", "spike"}

	for _, verb := range []string{"go", "switch", "add", "remove", "move"} {
		t.Run(verb, func(t *testing.T) {
			out := complete(t, completing(verb, own), verb, "")
			if !slices.Equal(rows(out), want) {
				t.Errorf("completing %s gave %q; want %q", verb, rows(out), want)
			}
			assertNoFileComp(t, out)

			past := complete(t, completing(verb, own), verb, "bd-1", "")
			if got := rows(past); got != nil {
				t.Errorf("completing past %s's position gave %q; want nothing", verb, got)
			}
			assertNoFileComp(t, past)
		})
	}

	// A repository that will not answer costs the rows, not the shell.
	silent := complete(t, front{reach: offering[opens]{list: stub(nil, errCancelled)}}, "go", "")
	if got := rows(silent); got != nil {
		t.Errorf("completing go on a silent repository gave %q; want nothing", got)
	}
	assertNoFileComp(t, silent)
}

// completing is a front whose verbs all offer rows of somebody else's but the
// one named, so what a verb offers names the listing it read.
func completing(verb string, own []work.Candidate) front {
	elsewhere := stub([]work.Candidate{{Place: worktree.Place{Source: "github", ID: "7", Name: "pr-7", Label: "Review this"}}}, nil)
	var f front
	offers := lists(&f)
	for _, list := range offers {
		*list = elsewhere
	}
	*offers[verb] = stub(own, nil)
	return f
}

// A tab press after list offers nothing: it takes no argument, and a file name
// is not one either.
func TestCompleteList(t *testing.T) {
	listed := []work.Candidate{{Place: worktree.Place{Source: "beads", ID: "bd-1", Name: "bd-1", Label: "Do a thing"}, Open: true}}

	var f front
	for _, list := range lists(&f) {
		*list = stub(listed, nil)
	}
	out := complete(t, f, "list", "")
	if got := rows(out); got != nil {
		t.Errorf("completing list gave %q; want nothing", got)
	}
	assertNoFileComp(t, out)
}

// Completion offers the declared flags, so a flag work no longer declares is a
// flag it no longer offers.
func TestGoneFlags(t *testing.T) {
	f := command(stubVersion, wired(), front{}).Flags()
	for _, name := range []string{"model", "effort", "delete", "force", "create"} {
		if f.Lookup(name) != nil {
			t.Errorf("--%s is still declared", name)
		}
	}
}

// init prints the shell's own function and then its completions, the one line
// that sources it installing both. The shell it names completes no further.
func TestInit(t *testing.T) {
	for _, c := range []struct{ shell, function, registers string }{
		{"fish", shim.Fish, "complete -c work"},
		{"bash", shim.Bash, "complete -o default -F __start_work work"},
		{"zsh", shim.Bash, "compdef _work work"},
	} {
		t.Run(c.shell, func(t *testing.T) {
			var out strings.Builder
			if err := execute(t, []string{"init", c.shell}, &out, front{}); err != nil {
				t.Fatalf("work init %s: %v", c.shell, err)
			}
			if !strings.Contains(out.String(), c.registers) {
				t.Errorf("work init %s printed %q; want a %s completion script", c.shell, out.String(), c.shell)
			}
			if !strings.HasPrefix(out.String(), c.function) {
				t.Errorf("work init %s printed %q; want the shim in front of it", c.shell, out.String())
			}
			// Cobra writes each script in two variants; init prints the one with titles.
			if strings.Contains(out.String(), cobra.ShellCompNoDescRequestCmd) {
				t.Errorf("work init %s printed a script that asks for no descriptions", c.shell)
			}
		})
	}
}

// A tab press on the shell offers the ones work prints and nothing else, and a
// word past it completes nothing: init takes the one shell.
func TestCompleteInit(t *testing.T) {
	out := complete(t, front{}, "init", "")
	want := []string{"bash\tbash shell integration", "fish\tfish shell integration", "zsh\tzsh shell integration"}
	if !slices.Equal(rows(out), want) {
		t.Errorf("completing init gave %q; want %q", rows(out), want)
	}
	assertNoFileComp(t, out)

	second := complete(t, front{}, "init", "fish", "")
	if got := rows(second); got != nil {
		t.Errorf("completing a second shell after init gave %q; want nothing", got)
	}
	assertNoFileComp(t, second)
}

// cobra's own completion command is gone, so work init fish is the one door and
// the word is a worktree name like any other. Only running it tells: the tree
// carries that command only once Execute ran.
func TestNoCompletionCommand(t *testing.T) {
	var target string
	err := execute(t, []string{"completion"}, io.Discard, front{reach: runs(func(_ options, id string) (worktree.Handoff, error) {
		target = id
		return worktree.Handoff{}, nil
	})})
	if err != nil || target != "completion" {
		t.Errorf(`work completion = %q, %v; want the worktree named completion`, target, err)
	}
}

// Each verb's completion offers exactly what its picker offers, row for row and
// in the same order: a tab press and a screen read the one listing that verb
// owns, so neither puts up a choice the other would leave out.
func TestEachVerbCompletesWhatItsPickerOffers(t *testing.T) {
	repo := testenv.InitRepo(t)
	dir := config.Default().Worktree.Dir()
	for _, name := range []string{"one", "two", "three"} {
		testenv.Git(t, repo, "worktree", "add", "-b", name, filepath.Join(repo, dir, name))
	}
	// Standing in a worktree, so each verb has its own reason to leave a row out.
	t.Chdir(filepath.Join(repo, dir, "one"))

	putUp := filepath.Join(t.TempDir(), "listing")
	// Cancelled, so the rows are all that comes of the picker and no verb goes on to
	// act on one.
	testenv.Stubs(t, testenv.Stub{Name: "fzf", Shell: "cat > " + putUp + "\n", Exits: 1})

	f := standing(spare{})

	for verb, pick := range pickers(f) {
		t.Run(verb, func(t *testing.T) {
			if err := os.Remove(putUp); err != nil && !os.IsNotExist(err) {
				t.Fatalf("clear the listing: %v", err)
			}
			if err := pick(); !errors.Is(err, errCancelled) {
				t.Fatalf("work %s = %v; want the picker dismissed", verb, err)
			}
			offered, completed := picked(t, putUp), names(rows(complete(t, f, verb, "")))
			if len(offered) != len(completed) {
				t.Fatalf("work %s put up %q and completes %q; want the one listing", verb, offered, completed)
			}
			for i, name := range completed {
				// A row is rendered for a screen rather than for a shell, so what the two
				// have in common is the name.
				if !strings.Contains(offered[i], name) {
					t.Errorf("work %s put up %q at row %d and completes %q there", verb, offered[i], i, name)
				}
			}
		})
	}
}

// picked is the rows a picker was put up with, read off what fzf was handed: the
// index each row is keyed on is the picker's own and not part of the row.
func picked(t *testing.T, file string) []string {
	t.Helper()
	out, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	var got []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			_, row, _ := strings.Cut(line, "\t")
			got = append(got, row)
		}
	}
	return got
}

// spare is the resolver these cases are wired with: it answers for whatever
// worktree git reports, as [last] does, and offers one place that has none, so
// add has a row of its own to put up.
type spare struct{ last }

func (spare) Name() string { return "spare" }

func (spare) Offer() ([]worktree.Place, error) {
	return []worktree.Place{{ID: "spare", Name: "spare", Label: "no worktree yet"}}, nil
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
