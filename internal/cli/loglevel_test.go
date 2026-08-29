package cli

import (
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/JHK/work-cli/internal/testenv"
)

// picking stands a shell in a repository where one run says something at every
// level, the settings among what it reads.
func picking(t *testing.T) *session {
	t.Helper()
	s := repository(t, testenv.Stub{Name: "fzf", Says: "0\tscratch\n"},
		testenv.Stub{Name: "bd", Replies: []testenv.Reply{{To: []string{"list"}, Says: "[]"}}})
	s.settings(on("beads"))
	s.opened("scratch")
	return s
}

// under is the values the run said that message under, and nothing where it
// never said it.
func (r result) under(message string) map[string]string {
	for _, one := range r.records {
		if one.Message == message {
			return one.Values
		}
	}
	return nil
}

// What each level lets a reader see: what refused a run and what a system came
// back short of always, what it reached for at info, what it read at debug.
func TestTheLogLevelLetsThrough(t *testing.T) {
	tests := []struct {
		name string
		flag []string
		want []slog.Level
	}{
		{"no flag", nil, []slog.Level{slog.LevelWarn}},
		{"warn", []string{"--log-level", "warn"}, []slog.Level{slog.LevelWarn}},
		{"info", []string{"--log-level", "info"}, []slog.Level{slog.LevelInfo, slog.LevelWarn}},
		{"debug", []string{"--log-level=debug"}, []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := picking(t)

			r := s.run(slices.Concat(tt.flag, []string{"go"})...)

			r.came(t, result{Answered: s.at("scratch"), Asked: []string{listed, vetted, putUp}}, apart, atOnce)
			testenv.Equal(t, tt.want, r.levels(), "the levels a reader was given")
		})
	}
}

// levels is the distinct levels a run said at, least first.
func (r result) levels() []slog.Level {
	var out []slog.Level
	for _, one := range r.records {
		if !slices.Contains(out, one.Level) {
			out = append(out, one.Level)
		}
	}
	slices.Sort(out)
	return out
}

// info is a line per external command work spawned, in the order they ran, each
// naming the tool and the arguments work put to it, the tool itself unchanged.
func TestInfoSaysEveryCommandWorkSpawned(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "bd", Says: "[]"})
	s.settings(on("beads"))
	wt := s.at("scratch")

	r := s.run("--log-level", "info", "add", "scratch")

	r.came(t, result{Answered: wt, Asked: []string{listed}}, apart)
	// The tools in the order they ran, a run of one counted once: how many questions
	// work puts to git is internal/git's business, not the flag's.
	testenv.Equal(t, []string{"git", "bd", "git"}, slices.Compact(tools(r.Informed)),
		"the tools work spawned, in the order it reached them")
	require.Contains(t, r.Informed, listed, "the tracker work reached for")
	require.Contains(t, r.Informed, "git worktree add -q "+wt+" -b scratch", "the worktree work made")
}

// tools is the tool each line named, which is its first word.
func tools(said []string) []string {
	out := make([]string, 0, len(said))
	for _, line := range said {
		bin, _, _ := strings.Cut(line, " ")
		out = append(out, bin)
	}
	return out
}

// What a reader would otherwise guess at rides the debug record as values rather
// than being written into its sentence, spelled as work config dump spells them.
func TestDebugNamesWhatWorkRead(t *testing.T) {
	tests := []struct {
		name  string
		typed func(*session, ...string) result
	}{
		{"typed in this process", (*session).run},
		{"typed in a child", (*session).hands},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := picking(t)

			r := tt.typed(s, "--log-level=debug", "go")

			require.Equal(t, s.Repo, r.under("work opened a repository")["repository"], "debug named no repository")
			force := r.under("the settings in force")
			require.Equal(t, defaultDir, force["worktree.directory"], "debug named no worktree directory")
			require.Equal(t, `["beads"]`, force["systems"], "debug named no tracker in force")
		})
	}
}

// A level work does not spell is refused rather than read as the nearest one,
// and the run it was typed on does not happen.
func TestALevelWorkDoesNotSpellIsRefused(t *testing.T) {
	// --help and --version are answered before any verb runs, and refuse it too.
	for _, alongside := range [][]string{{"add", "scratch"}, {"--help"}, {"--version"}} {
		t.Run(strings.Join(alongside, " "), func(t *testing.T) {
			s := picking(t)

			r := s.run(slices.Concat([]string{"--log-level=shouty"}, alongside)...)

			r.refused(t, `invalid argument "shouty" for "--log-level" flag`, "work says at warn, info, debug")
		})
	}
}

// One letter that could name either the version or the level names neither: -v
// and -l reach no flag, and the version keeps the spelling it always had.
func TestNeitherTheVersionNorTheLevelTakesAShorthand(t *testing.T) {
	s := repository(t)

	for _, letter := range []string{"v", "l"} {
		refused := s.run("-" + letter)

		refused.came(t, result{Code: 1, Errored: []string{"unknown shorthand flag: '" + letter + "' in -" + letter}})
	}
	s.run("--version").came(t, result{Out: versionLine})
}

// pflag names the values and the default, and no root flag runs past eighty
// columns.
func TestTheLogLevelReadsLikeTheRootFlagsBesideIt(t *testing.T) {
	s := repository(t)

	r := s.run("--help")

	r.came(t, result{}, besides("Out"))
	require.Regexp(t, `--log-level warn\|info\|debug\s+say what work reached for \(default warn\)`,
		r.Out, "the line a reader is given for the flag")
	for line := range strings.SplitSeq(r.Out, "\n") {
		if written.MatchString(line) {
			require.LessOrEqual(t, len(line), 80, "a root flag wraps at eighty columns: %q", line)
		}
	}
}

// The command a worktree opens on is the last thing work reaches for, and work
// replaces itself with it, so only a run watched from outside sees the line.
func TestInfoNamesTheCommandTheWorktreeOpensOn(t *testing.T) {
	// One nothing else in the run spawns, so only the handoff can have said it.
	s := tracking(t, []ticket{doable}, []ticket{doable}, []string{"claude"},
		claudeTable+"command = [\"git\", \"--version\"]\n")

	r := s.hands("--log-level", "info", "add", "bd-1")

	r.came(t, result{Asked: worked("bd-1", s.at("bd-1"), "bd-1-do-a-thing")}, apart, besides("Out"))
	require.Equal(t, "git --version", r.Informed[len(r.Informed)-1],
		"the command the terminal went to was not the last line on stderr")
}
