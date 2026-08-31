package cli

import (
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/JHK/work-cli/internal/shim"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// The one cobra answers alone, and the one work's own listing answers.
func TestNoPositionCompletesAFileName(t *testing.T) {
	s := repository(t)

	for _, position := range [][]string{{"list", ""}, {"go", ""}} {
		t.Run(position[0], func(t *testing.T) {
			out := s.completes(position...)

			// The directive line the shell acts on, not the prose cobra prints beside it,
			// which a directive carrying other bits also spells.
			want := ":" + strconv.Itoa(int(cobra.ShellCompDirectiveNoFileComp))
			require.Contains(t, strings.Split(strings.TrimSpace(out), "\n"), want, "completion %q may fall back to file names", out)
		})
	}
}

// An identifier is go's own completion.
func TestTheBarePositionCompletesTheVerbsAlone(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, nil, "")
	s.opened("scratch")

	out := s.completes("")

	offered := names(rows(out))
	for _, verb := range []string{"go", "switch", "add", "carry", "remove", "move", "list", "init"} {
		require.Contains(t, offered, verb, "completing the bare position offered no %s row", verb)
	}
	for _, absent := range []string{"bd-1", "scratch"} {
		require.NotContains(t, offered, absent, "completing the bare position offered a %s row", absent)
	}
}

func TestNothingCompletesPastAVerbsPosition(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, nil, "")
	s.opened("spare")

	for _, verb := range []string{"go", "switch", "add", "remove", "move"} {
		t.Run(verb, func(t *testing.T) {
			out := s.completes(verb, "spare", "")

			require.Empty(t, rows(out), "completing past %s's position offered rows", verb)
		})
	}
}

// A repository that will not answer costs the rows, not the shell: a tab press
// still ends well.
func TestCompletingASilentRepository(t *testing.T) {
	s := repository(t)
	s.Dir = t.TempDir()

	out := s.completes("go", "")

	require.Empty(t, rows(out), "completing go outside a repository offered rows")
}

func TestNothingCompletesAfterList(t *testing.T) {
	s := repository(t)
	s.opened("scratch")

	out := s.completes("list", "")

	require.Empty(t, rows(out), "completing list offered rows")
}

// Row for row and in the same order: a tab press and a screen read the one
// listing that verb owns.
func TestEachVerbCompletesWhatItsPickerOffers(t *testing.T) {
	put := putsUp(t)
	// A ticket with no worktree yet, so add has a row of its own to put up.
	spare := with(doable, func(b *ticket) { b.ID = "spare" })
	s := tracking(t, []ticket{spare}, []ticket{spare}, nil, "", put.dismisses())
	s.opened("one")
	// Standing in a worktree, so each verb has its own reason to leave a row out.
	s.Dir = s.opened("stood-in")

	for _, verb := range []string{"go", "switch", "add", "remove", "move"} {
		t.Run(verb, func(t *testing.T) {
			// Dismissed, so the rows are all that comes of the screen.
			r := s.run(verb)
			r.came(t, result{Code: 1}, besides("Asked"))

			out := s.completes(verb, "")

			// A row is rendered for a screen rather than for a shell, so what the two have
			// in common is the name each goes by.
			testenv.Equal(t, retyped(put.rows()), names(rows(out)), "work "+verb+" completes other than what it put up")
		})
	}
}

// Which generator a shell reaches is work's, the words it writes cobra's.
func TestInitPrintsTheFunctionThenTheShellsCompletions(t *testing.T) {
	s := repository(t)
	// The line each script opens with, so a failure names two lines and not two scripts.
	heads := map[string]string{}

	for _, c := range []struct{ shell, function string }{
		{"fish", shim.Fish},
		{"bash", shim.Bash},
		{"zsh", shim.Bash},
	} {
		t.Run(c.shell, func(t *testing.T) {
			r := s.run("init", c.shell)

			r.came(t, result{}, besides("Out"))
			require.True(t, strings.HasPrefix(r.Out, c.function), "work init %s printed the shim behind the completions", c.shell)
			printed := strings.TrimPrefix(r.Out, c.function)
			require.NotEmpty(t, printed, "work init %s printed the shim and no completions", c.shell)
			heads[c.shell], _, _ = strings.Cut(printed, "\n")
		})
	}

	for _, pair := range [][2]string{{"bash", "fish"}, {"bash", "zsh"}, {"fish", "zsh"}} {
		require.NotEqual(t, heads[pair[0]], heads[pair[1]], "work init gave %s and %s the one completion script", pair[0], pair[1])
	}
}

// A word past the shell completes nothing: init takes the one shell.
func TestInitCompletesTheShellsWorkPrintsFor(t *testing.T) {
	s := repository(t)

	out := s.completes("init", "")

	want := []string{"bash\tbash shell integration", "fish\tfish shell integration", "zsh\tzsh shell integration"}
	testenv.Equal(t, want, rows(out), "completing init offered other than the shells work prints for")

	second := s.completes("init", "fish", "")
	require.Empty(t, rows(second), "completing a second shell after init offered rows")
}

// Wherever the flag is typed: the root hands it down to every verb.
func TestTheLogLevelCompletesTheLevelsWorkSaysAt(t *testing.T) {
	s := repository(t)

	for _, before := range [][]string{{}, {"list"}} {
		out := s.completes(append(before, "--log-level", "")...)

		testenv.Equal(t, []string{"warn", "info", "debug"}, rows(out), "completing --log-level offered other than the levels")
	}
}

// completes asks work what it would offer, the way a shell does: a tab press
// answers on stdout, asks each listing behind the verb and says nothing at all.
func (s *session) completes(args ...string) string {
	t := s.t
	t.Helper()
	r := s.run(append([]string{cobra.ShellCompRequestCmd}, args...)...)
	r.came(t, result{}, besides("Out", "Asked"))
	return r.Out
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
