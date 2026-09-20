package cli

import (
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

// documented is every setting work reads, spelled as
// docs/references/configuration.md spells it and grouped under the table it sits
// in, which no case may derive from the code.
var documented = []string{
	"integrations",
	"worktree.directory",
	"beads.branch",
	"claude.command",
}

// dumped is a printed configuration read back: the text, and the keys in the
// order printed, each under the whole name a settings file spells.
type dumped struct {
	text string
	keys []string
}

// dumping types work config dump in that session and reads back what it printed.
func dumping(t *testing.T, s *session) dumped {
	t.Helper()
	r := s.run("config", "dump")
	r.came(t, result{}, besides("Out"))

	return dumped{text: r.Out, keys: keysIn(r.Out)}
}

// keysIn is the keys a printed configuration names, in the order printed.
func keysIn(text string) []string {
	var keys []string
	table, inBlock := "", false
	for line := range strings.SplitSeq(text, "\n") {
		switch {
		// What stands between a block's quotes is one value, not keys or tables of its
		// own.
		case inBlock:
			inBlock = line != quotes
		case strings.HasPrefix(line, "["):
			table = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
		case strings.Contains(line, " = "):
			leaf, value, _ := strings.Cut(line, " = ")
			name := leaf
			if table != "" {
				name = table + "." + leaf
			}
			keys = append(keys, name)
			inBlock = value == quotes
		}
	}
	return keys
}

// A setting the dump leaves out is one a reader could never see, and a key
// spelled otherwise than a settings file spells it is one nothing would load
// back: docs/references/cli.md#config.
func TestConfigDumpNamesEverySetting(t *testing.T) {
	s := repository(t)

	testenv.Equal(t, documented, dumping(t, s).keys, "the dump names other settings than the reference does")
}

// What the dump prints is a settings file: a machine whose whole configuration
// is that text names the same settings, whether a key came from the file or from
// the defaults.
func TestConfigDumpLoadsBack(t *testing.T) {
	tests := []struct{ name, body string }{
		{"the compiled-in defaults", ""},
		// A quote and a tab survive the printing, the block holding both as written.
		{"a file naming every key", integrationsOn("claude") + commandBlock("claude", "--name=\"{{.Name}}\"", "a\tb") +
			directory("{{.Repo}}/trees") +
			"[beads]\nbranch = \"{{.ID}}\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			s.settings(tt.body)

			first := dumping(t, s)
			// A machine whose whole configuration is what the first dump printed.
			s.settings(first.text)

			require.Equal(t, first.text, dumping(t, s).text, "the dump loads back as another configuration")
		})
	}
}

// A command is printed as the block it is written as, the compiled-in one
// included, which is the one form a settings file holds it in.
func TestConfigDumpPrintsTheCommandAsABlock(t *testing.T) {
	s := repository(t)

	got := dumping(t, s).text

	require.Contains(t, got, "\ncommand = "+quotes+"\n", "the compiled-in command was printed as something other than a block")
	require.Contains(t, got, "\nclaude\n", "the block the dump printed names no command to run")
	require.True(t, strings.HasSuffix(got, "\n"+quotes+"\n"), "the block the dump printed is never closed: %q", got)
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
		{"a directory that does not parse", directory("{{.Repo"), "worktree.directory"},
		{"a value the directory does not have", directory("{{.Root}}/trees"), "{{.Repo}}"},
		{"a pattern that does not parse", "[beads]\nbranch = \"{{.ID\"\n", "beads.branch"},
		{"a pattern that is not a string", "[beads]\nbranch = 3\n", "branch"},
		{"a ticket pattern without its id", "[beads]\nbranch = \"feature/{{.Slug}}\"\n", "places no {{.ID}}"},
		// The pattern places the id, but only where a ticket with no slug would not
		// reach, and that ticket's branch would then stand for every ticket.
		{"an id only some tickets reach", "[beads]\nbranch = \"{{with .Slug}}{{$.ID}}-{{.}}{{end}}\"\n", "places no {{.ID}}"},
		{"a branch opening with a dash", "[beads]\nbranch = \"-{{.ID}}\"\n", "dash"},
		// An integration is switched on by the one integrations list, which is the
		// only key that names one.
		{"a name in the list no integration goes by", "integrations = [\"linear\"]\n", "is no integration work has"},
		{"an integration named in another case", "integrations = [\"Beads\"]\n", "is no integration work has"},
		{"integrations that are not a list", "integrations = \"beads\"\n", "integrations"},
		{"an integration that is not a string", "integrations = [3]\n", "integrations"},
		{"a key the agent's table does not have", "[claude]\nstart = [\"claude\"]\n", "unknown setting"},
		{"a command written as a list", "[claude]\ncommand = [\"claude\"]\n", "claude.command"},
		{"a command that is not text", "[claude]\ncommand = 3\n", "claude.command"},
		{"a template that does not parse", "[claude]\ncommand = '''\nclaude {{.ID\n'''\n", "claude.command"},
		{"a command naming nothing", "[claude]\ncommand = '''\n'''\n", "claude.command: names no command"},
		// A guard no source reaches leaves a block that renders for nothing at all.
		{"a command no value renders", "[claude]\ncommand = '''\n{{if eq .Source \"gitlab\"}}claude{{end}}\n'''\n", "claude.command: names no command"},
		{"a value the command does not have", "[claude]\ncommand = '''\nclaude {{.Number}}\n'''\n", "{{.Subject}}"},
		// Only the arm a ticket carrying a title reaches names it.
		{"a value named inside a branch", "[claude]\ncommand = '''\nclaude {{with .Title}}{{$.Number}}{{end}}\n'''\n", "claude.command"},
		// Only the arm a worktree the tracker answered for reaches names it.
		{"a value named inside a source arm", "[claude]\ncommand = '''\nclaude {{if eq .Source \"beads\"}}{{.Number}}{{end}}\n'''\n", "claude.command"},
		{"a command naming a filter that does not exist", "[claude]\ncommand = '''\nclaude {{.Subject | shout}}\n'''\n", "claude.command"},
		{"a branch pattern naming squote", "[beads]\nbranch = \"{{.ID | squote}}\"\n", "beads.branch"},
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
		// A word the command line refuses too: the settings are still what the reader
		// hears, being why the verb could not run at all.
		{"go", "--turbo", "scratch"},
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

func TestAWorktreeLandsWhereASymlinkedDirectoryLeads(t *testing.T) {
	s := repository(t)
	elsewhere := t.TempDir()
	require.NoError(t, os.Symlink(elsewhere, defaultDir(s.Repo)), "symlink the worktree directory")

	r := s.run("add", "scratch")

	r.came(t, result{Answered: filepath.Join(resolved(t, elsewhere), "scratch")})
	require.DirExists(t, filepath.Join(elsewhere, "scratch"), "the worktree is not where the symlink leads")
}

// Creation answers git's own path, as every other verb does, so one worktree is
// one place however it was reached:
// docs/explanation/worktree-identity.md#the-branch-is-the-identity-not-the-path.
func TestCreatingAWorktreeAnswersThePathEnteringItDoes(t *testing.T) {
	for _, verb := range []string{"add", "go"} {
		t.Run(verb, func(t *testing.T) {
			s := repository(t)
			require.NoError(t, os.Symlink(t.TempDir(), defaultDir(s.Repo)), "symlink the worktree directory")

			made := s.run(verb, "scratch")
			entered := s.run("switch", "scratch")

			require.Equal(t, entered.Answered, made.Answered, "work %s answers what work switch does not", verb)
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

// An integration the list names is reached on every seam it fills, and one the
// list leaves out is reached nowhere: docs/references/integrations.md.
func TestEachIntegrationIsReachedOnlyWhereTheListNamesIt(t *testing.T) {
	tests := []struct {
		name, body string
		// picking is what the stand-ins were asked to put the picker up with, and
		// creating what they were asked to bring a worktree of a plain name into being.
		picking, creating []string
	}{
		{"the forge lists the pull requests", integrationsOn("github"), []string{pullRequests(hosted), putUp}, nil},
		{"the tracker lists the tickets", integrationsOn("beads"), []string{listed, vetted, putUp}, []string{listed}},
		{"the runner trusts a fresh worktree", integrationsOn("mise"), []string{putUp}, []string{"mise trust"}},
		// The default command renders nothing for a name of the user's own, so what the
		// agent is asked here is what it is asked at a seam, which is nothing.
		{"the agent fills neither seam", integrationsOn("claude"), []string{putUp}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t, tracker("[]", "[]"), testenv.Stub{Name: "gh", Says: "[]"},
				testenv.Stub{Name: "mise"})
			s.settings(tt.body)
			testenv.Git(t, s.Repo, "remote", "add", "origin", hosted)
			// A row for the picker to be put up over, which is what asks each listing.
			s.opened("scratch")

			// Dismissed, so the listing each integration was asked for is all that came
			// of it.
			s.run("go").came(t, result{Code: 1, Asked: tt.picking}, atOnce)

			s.run("add", "fresh").came(t, result{Answered: s.at("fresh"), Asked: tt.creating})
		})
	}
}

// The directory the file names, wherever on disk it leads.
func TestWhereAWorktreeLands(t *testing.T) {
	tests := []struct {
		name  string
		lands func(s *session) (dir, at string)
	}{
		{"a directory under the repository", func(s *session) (string, string) {
			return "{{.Repo}}/mine", filepath.Join(s.Repo, "mine", "scratch")
		}},
		{"a directory outside the repository", func(s *session) (string, string) {
			outside := s.t.TempDir()
			return filepath.Join(outside, "trees"), filepath.Join(outside, "trees", "scratch")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t)
			dir, at := tt.lands(s)
			s.settings(directory(dir))

			r := s.run("add", "scratch")

			r.came(t, result{Answered: at})
		})
	}
}

// A file that names one key leaves the rest to the compiled-in defaults: a table
// present but empty moves nothing, and the keys it left out still name the
// directory, the verbs and the command.
func TestAFileNamingOneKeyLeavesTheRestToTheDefaults(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, []string{"claude"}, "[worktree]\n",
		testenv.Stub{Name: "claude"})

	r := s.hands("add", "bd-1")

	r.came(t, result{Asked: append(worked("bd-1", s.at("bd-1"), "bd-1-do-a-thing"),
		ticketSessionOn("bd-1", "Do a thing"))})
	require.DirExists(t, s.at("bd-1"), "the worktree did not land where the compiled-in directory names")
}

// A command the file names replaces the compiled-in one whole rather than line
// by line, and is what the worktree opens on.
func TestACommandInTheFileReplacesTheDefaultWhole(t *testing.T) {
	// Shorter than the default, so one replaced line by line would leave the
	// default's tail behind.
	s := tracking(t, []ticket{doable}, []ticket{doable}, []string{"claude"},
		commandBlock("claude", "{{.Name}}"), testenv.Stub{Name: "claude"})

	r := s.hands("add", "bd-1")

	r.came(t, result{Asked: append(worked("bd-1", s.at("bd-1"), "bd-1-do-a-thing"), "claude bd-1")})
}
