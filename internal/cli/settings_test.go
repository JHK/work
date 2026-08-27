package cli

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/shim"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// directory is a settings body naming where worktrees go.
func directory(dir string) string {
	return fmt.Sprintf("[worktree]\ndirectory = %q\n", dir)
}

// compiledIn is what a dump names as the source of a key no file set.
const compiledIn = "the compiled-in default"

// documented is every setting work reads, spelled as
// docs/references/configuration.md spells it and grouped under the table it sits
// in, which no case may derive from the code.
var documented = []string{
	"worktree.directory",
	"branch.ticket",
	"branch.pull-request",
	"action.create",
	"action.enter",
	"github.enabled",
	"beads.enabled",
	"mise.enabled",
	"claude.enabled",
	"claude.start-ticket",
	"claude.start-pull-request",
	"claude.start-session",
	"claude.resume-session",
}

// dumped is a printed configuration read back: the keys in the order they were
// printed, the source named above each and the value under it. A key is held by
// the dotted name a settings file spells, a leaf alone not being unique with
// `enabled` standing in every system's table.
type dumped struct {
	text  string
	keys  []string
	from  map[string]string
	value map[string]string
}

// dumping types work config dump in that session and reads back what it printed.
func dumping(t *testing.T, s *session) dumped {
	t.Helper()
	r := s.run("config", "dump")
	r.came(t, result{}, besides("Out"))

	d := dumped{text: r.Out, from: map[string]string{}, value: map[string]string{}}
	table, source := "", ""
	for line := range strings.SplitSeq(r.Out, "\n") {
		switch {
		case strings.HasPrefix(line, "["):
			table = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
		case strings.HasPrefix(line, "# "):
			source = strings.TrimPrefix(line, "# ")
		case strings.Contains(line, " = "):
			leaf, value, _ := strings.Cut(line, " = ")
			name := table + "." + leaf
			d.keys = append(d.keys, name)
			d.from[name], d.value[name] = source, value
		}
	}
	return d
}

// A setting the dump leaves out is one a reader could never see, and a key
// spelled otherwise than a settings file spells it is one nothing would load
// back: docs/references/cli.md#config.
func TestConfigDumpNamesEverySetting(t *testing.T) {
	s := repository(t)

	testenv.Equal(t, documented, dumping(t, s).keys, "the dump names other settings than the reference does")
}

// The file, by the path it was read at, or the compiled-in default where the
// file left it alone.
func TestConfigDumpNamesWhereEachKeyCameFrom(t *testing.T) {
	s := repository(t)
	file := s.settings(directory("trees") + trackerOn)

	from := dumping(t, s).from

	// A system nothing named says it is off and where that comes from, so a dump
	// shows every system's state and not only the ones a file spoke for.
	want := map[string]string{
		"worktree.directory": file,
		"beads.enabled":      file,
		"branch.ticket":      compiledIn,
		"claude.enabled":     compiledIn,
	}
	got := map[string]string{}
	for key := range want {
		got[key] = from[key]
	}
	testenv.Equal(t, want, got, "a key came from the wrong place")
}

// What the dump prints is a settings file: a machine whose whole configuration
// is that text names the same settings, whether a key came from the file or from
// the defaults.
func TestConfigDumpLoadsBack(t *testing.T) {
	s := repository(t)
	// A quote and a tab survive the printing, being written as TOML escapes.
	s.settings(directory("trees") + "[branch]\nticket = \"{{.ID}}\"\npull-request = \"review/{{.Number}}\"\n" +
		"[action]\nenter = \"claude\"\n[claude]\nenabled = true\nstart-session = [\"claude\", \"--name=\\\"{{.Name}}\\\"\", \"a\\tb\"]\n")

	first := dumping(t, s)
	// A machine whose whole configuration is what the first dump printed.
	s.settings(first.text)

	testenv.Equal(t, first.value, dumping(t, s).value, "the dump loads back as another configuration")
}

// A value a settings file may not hold stops the command that read it, naming
// the key at fault and refusing in work's own words:
// docs/references/configuration.md.
func TestASettingsFileWorkWillNotRead(t *testing.T) {
	tests := []struct{ name, body, want string }{
		{"an unknown key", "[worktree]\ndirectry = \"trees\"\n", "unknown setting"},
		{"an unknown table", "[nonsense]\nkey = 1\n", "unknown setting"},
		{"a value of the wrong type", "[worktree]\ndirectory = 3\n", "directory"},
		{"a key spelled in another case", "[worktree]\nDirectory = \"trees\"\n", "unknown setting"},
		{"a table spelled in another case", "[Worktree]\ndirectory = \"trees\"\n", "unknown setting"},
		{"a directory outside the repository", directory("../trees"), "not a directory inside"},
		{"an absolute directory", directory("/tmp/trees"), "not a directory inside"},
		{"the repository root, unnamed", directory(""), "not a directory inside"},
		{"the repository root, as a dot", directory("."), "not a directory inside"},
		{"the repository root, with a trailing slash", directory("./"), "not a directory inside"},
		{"the repository root, by traversal", directory("trees/.."), "not a directory inside"},
		{"git's own directory", directory(".git"), "git's own directory"},
		{"a directory under git's own", directory(".git/worktrees"), "git's own directory"},
		{"a pattern that does not parse", "[branch]\nticket = \"{{.ID\"\n", "branch.ticket"},
		{"a pattern that is not a string", "[branch]\nticket = 3\n", "ticket"},
		{"a ticket pattern without its id", "[branch]\nticket = \"feature/{{.Slug}}\"\n", "places no {{.ID}}"},
		// The pattern places the id, but only where a ticket with no slug would not
		// reach, and that ticket's branch would then stand for every ticket.
		{"an id only some tickets reach", "[branch]\nticket = \"{{with .Slug}}{{$.ID}}-{{.}}{{end}}\"\n", "places no {{.ID}}"},
		{"a pull request pattern without its number", "[branch]\npull-request = \"pr-{{.ID}}\"\n", "{{.Number}}"},
		{"a branch opening with a dash", "[branch]\nticket = \"-{{.ID}}\"\n", "dash"},
		{"a system work does not have", "[linear]\nenabled = true\n", "unknown setting"},
		{"an unknown action key", "[action]\nopen = \"shell\"\n", "unknown setting"},
		{"an action nothing goes by", "[action]\ncreate = \"launcher\"\n", "is not an action"},
		{"an action named for its flag", "[action]\nenter = \"--shell\"\n", "is not an action"},
		{"the unnamed action, which is no action", "[action]\nenter = \"unnamed\"\n", "is not an action"},
		{"an action that is not a string", "[action]\ncreate = 3\n", "create"},
		{"an action in another case", "[action]\nenter = \"Shell\"\n", "is not an action"},
		// An action work ships, of a system this file left out: docs/references/configuration.md#actions.
		{"an action nothing here is on for", "[action]\nenter = \"claude\"\n", "is not an action"},
		// A file written before a rename is told the new spelling rather than that
		// what it names is unknown.
		{"an action under the name it used to go by", "[action]\ncreate = \"agent\"\n", `"agent" is now "claude"`},
		{"a table under the name it used to go by", "[agent]\nstart-ticket = [\"claude\"]\n", "the [agent] table is now [claude]"},
		{"an unknown command key", "[claude]\nstart = [\"claude\"]\n", "unknown setting"},
		{"a command that is not a list", "[claude]\nstart-ticket = \"claude\"\n", "list of command line arguments"},
		{"a list of something other than strings", "[claude]\nstart-ticket = [1, 2]\n", "list of command line arguments"},
		{"a template that does not parse", "[claude]\nstart-ticket = [\"claude\", \"{{.ID\"]\n", "claude.start-ticket"},
		// Every [claude] key is judged, whichever of them a file names.
		{"a ticket command naming nothing", "[claude]\nstart-ticket = []\n", "claude.start-ticket: names no command"},
		{"a pull request command naming nothing", "[claude]\nstart-pull-request = []\n", "claude.start-pull-request: names no command"},
		{"a session command naming nothing", "[claude]\nstart-session = []\n", "claude.start-session: names no command"},
		{"a resumed command naming nothing", "[claude]\nresume-session = []\n", "claude.resume-session: names no command"},
		{"a value the key does not have", "[claude]\nstart-ticket = [\"claude\", \"{{.Number}}\"]\n", "{{.Title}}"},
		{"a value no key has", "[claude]\nresume-session = [\"claude\", \"{{.Branch}}\"]\n", "claude.resume-session"},
		// The two work once placed itself, and now has no more than any other name.
		{"a model or an effort", "[claude]\nstart-ticket = [\"claude\", \"--model={{.Model}}\", \"--effort={{.Effort}}\"]\n", "claude.start-ticket"},
		// Only the arm a target with a session reaches names it.
		{"a value named inside a branch", "[claude]\nresume-session = [\"claude\", \"{{with .Session}}{{$.Branch}}{{end}}\"]\n", "claude.resume-session"},
		// The whole table went with the commands that were under it: the shell action
		// hands back the worktree now, and the editor and the diff are gone.
		{"the table of commands to open on", "[open]\nshell = [\"fish\"]\neditor = [\"vi\", \"{{.Dir}}\"]\n", "unknown setting open"},
		{"the editor action", "[action]\ncreate = \"editor\"\n", "action.create"},
		{"the diff action", "[action]\nenter = \"diff\"\n", "action.enter"},
		{"the screen, which was never a command", "[action]\ncreate = \"ask\"\n", "action.create"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			file := s.settings(tt.body)

			r := s.run("config", "dump")

			r.refused(t, tt.want, file)
		})
	}
}

// The settings are read before a verb does anything, so a file work will not
// read stops it where it stands rather than halfway through.
func TestASettingsFileWorkWillNotReadStopsEveryVerb(t *testing.T) {
	commands := [][]string{
		{"go", "scratch"}, {"switch", "scratch"}, {"add", "other"}, {"carry", "other"},
		{"remove", "scratch"}, {"move", "scratch", "other"}, {"list"}, {"config", "dump"},
		// A file that would not load wired nothing, so the flag a system spells is
		// missing too: what the reader hears is still the settings.
		{"go", "--claude", "scratch"},
	}
	for _, args := range commands {
		typed := strings.Join(args, " ")
		t.Run(typed, func(t *testing.T) {
			s := repository(t)
			path := s.opened("scratch")
			file := s.settings("[worktree]\ndirectry = \"trees\"\n")

			r := s.run(args...)

			r.refused(t, "unknown setting", file)
			require.True(t, s.hasWorktree(path), "the worktree the verb acted on is gone")
			require.NoDirExists(t, s.at("other"), "the verb made a worktree before reading the settings")
		})
	}
}

// The containment check is not lexical, and belongs to no one directory: a
// repository is cloned with its symlinks whatever the settings say, and a
// worktree may not land outside it.
func TestAWorktreeDirectorySymlinkedOutOfTheRepositoryIsRefused(t *testing.T) {
	tests := []struct{ name, dir, body string }{
		{"the compiled-in default", defaultDir, ""},
		{"a directory the file names", "trees", directory("trees")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			outside := t.TempDir()
			require.NoError(t, os.Symlink(outside, filepath.Join(s.Repo, tt.dir)))
			if tt.body != "" {
				s.settings(tt.body)
			}

			r := s.run("add", "scratch")

			r.refused(t, "resolves outside")
			require.NoDirExists(t, filepath.Join(outside, "scratch"), "work wrote outside the repository")
		})
	}
}

// docs/references/cli.md#init.
func TestASettingsFileWorkWillNotReadLeavesInitAlone(t *testing.T) {
	s := repository(t)
	s.settings("[worktree]\ndirectry = \"oops\"\n")

	r := s.run("init", "fish")

	r.came(t, result{}, besides("Out"))
	require.Contains(t, r.Out, shim.Fish, "work init fish printed no shell integration")
}

// The flags the command line records are spelled from the settings as read, so a
// system no file switched on is a system no flag names.
func TestASystemTheSettingsLeftOutSpellsNoFlag(t *testing.T) {
	s := repository(t)

	// Refused in the flag parsing, so the worktree the words name need not be there.
	r := s.run("switch", "scratch", "--claude")

	// A file work could read, so the word itself is what is wrong with it. The same
	// flag under a file work refuses hears about the file instead.
	r.came(t, result{Code: 1, Errored: []string{"unknown flag: --claude"}})
}

// A system its table switched on is reached on every seam it fills, and one the
// file left out is reached nowhere: docs/references/systems.md.
func TestEachSystemIsReachedOnlyWhereItsTableSwitchesItOn(t *testing.T) {
	tests := []struct {
		name, body string
		// picking is what the stand-ins were asked to put the picker up with, and
		// creating what they were asked to bring a worktree of a plain name into being.
		picking, creating []string
	}{
		{"the forge lists the pull requests", forgeOn, []string{pullRequests(hosted), putUp}, nil},
		{"the tracker lists the tickets", trackerOn, []string{listed, vetted, putUp}, []string{listed}},
		{"the runner trusts a fresh worktree", miseOn, []string{putUp}, []string{"mise trust"}},
		{"the agent fills neither seam", claudeOn, []string{putUp}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t, tracker("[]", "[]"), testenv.Stub{Name: "gh", Says: "[]"},
				testenv.Stub{Name: "mise"})
			s.settings(tt.body)
			testenv.Git(t, s.Repo, "remote", "add", "origin", hosted)
			// A row for the picker to be put up over, which is what asks each listing.
			s.opened("scratch")

			// Dismissed, so the listing each system was asked for is all that came of it.
			s.run("go").came(t, result{Code: 1, Asked: tt.picking}, atOnce)

			s.run("add", "fresh").came(t, result{Answered: s.at("fresh"), Asked: tt.creating})
		})
	}
}

// The directory the file names, and the compiled-in default where it names none.
func TestWhereAWorktreeLands(t *testing.T) {
	for _, tt := range []struct{ name, dir string }{
		{"no file at all", ""},
		{"the file", "mine"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			if tt.dir != "" {
				s.settings(directory(tt.dir))
			}

			r := s.run("add", "scratch")

			r.came(t, result{Answered: filepath.Join(s.Repo, cmp.Or(tt.dir, defaultDir), "scratch")})
		})
	}
}

// A file that names one key leaves the rest to the compiled-in defaults: a table
// present but empty moves nothing, and the [action] key it left out is still the
// default.
func TestAFileNamingOneKeyLeavesTheRestToTheDefaults(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "claude"})
	s.settings("[worktree]\n" + claudeOn + "[action]\nenter = \"claude\"\n")

	made := s.run("add", "scratch")

	made.came(t, result{Answered: s.at("scratch")})

	entered := s.hands("switch", "scratch")

	entered.came(t, result{Asked: []string{"claude --permission-mode auto --name=scratch"}})
}

// A command the file names replaces the compiled-in one whole rather than
// element by element, and is what the worktree opens on.
func TestACommandInTheFileReplacesTheDefaultWhole(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "claude"})
	s.opened("scratch")
	// Shorter than the default, so one replaced element by element would leave the
	// default's tail behind.
	s.settings(claudeOn + "start-session = [\"claude\", \"{{.Name}}\"]\n")

	r := s.hands("switch", "scratch", "--claude")

	r.came(t, result{Asked: []string{"claude scratch"}})
}
