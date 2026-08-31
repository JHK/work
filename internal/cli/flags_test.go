package cli

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// creatingVerbs are the verbs that can bring a worktree into being, sorted as a
// settings file is read against.
var creatingVerbs = []string{"add", "carry", "go"}

// The root's three and --force on the verb that uses it are the whole of the
// command line's flags. Nothing spells one on the root beyond those, which the
// bare form would then take ahead of go. R2 of docs/rules/command-grammar.md.
func TestWhereEachFlagIsDeclared(t *testing.T) {
	s := repository(t)
	// Every system switched on, so a flag a system reached the command line with
	// would show up here.
	s.settings(systemsOn("claude", "beads", "mise", "github"))

	// The help is on every command and the level is handed down to every command;
	// the version is the root's alone, and --force is remove's.
	want := map[string][]string{
		"":            {"help", "log-level", "version"},
		"go":          {"help", "log-level"},
		"switch":      {"help", "log-level"},
		"add":         {"help", "log-level"},
		"carry":       {"help", "log-level"},
		"remove":      {"force", "help", "log-level"},
		"move":        {"help", "log-level"},
		"list":        {"help", "log-level"},
		"init":        {"help", "log-level"},
		"config":      {"help", "log-level"},
		"config dump": {"help", "log-level"},
		"config edit": {"help", "log-level"},
	}
	got := map[string][]string{}
	for where := range want {
		got[where] = s.prints(strings.Fields(where)...)
	}

	testenv.Equal(t, want, got, "a command offers other flags than R2 leaves it")
}

// flagLine is how --help writes a flag, which is the one place a reader is given
// the set: a flag declared and hidden is written nowhere.
var flagLine = regexp.MustCompile(`(?m)^\s+(?:-\w, )?--([\w-]+)`)

// prints is the flags a command offers a reader, which is what --help lists.
func (s *session) prints(args ...string) []string {
	s.t.Helper()
	r := s.run(append(args, "--help")...)
	r.came(s.t, result{}, besides("Out"))
	var flags []string
	for _, m := range flagLine.FindAllStringSubmatch(r.Out, -1) {
		flags = append(flags, m[1])
	}
	return flags
}

// The verbs claude.on-creation is read against are the verbs the command line
// carries, and the ones it may name are the ones a worktree comes into being
// under. Nothing in the compiler holds the two spellings together.
func TestTheSettingsKnowEveryVerbTheCommandLineCarries(t *testing.T) {
	s := repository(t)

	// Cobra's own help command is no verb work declares.
	listed := slices.DeleteFunc(s.available(), func(verb string) bool { return verb == "help" })

	testenv.Equal(t, listed, slices.Sorted(slices.Values(config.Verbs())),
		"the settings read claude.on-creation against other verbs than work --help lists")
	testenv.Equal(t, creatingVerbs, slices.Sorted(slices.Values(config.Creating())),
		"the settings let claude.on-creation name other verbs than the ones that create a worktree")
}

// available is the verbs work --help lists, which is the one place a reader is
// given the set.
func (s *session) available() []string {
	s.t.Helper()
	r := s.run("--help")
	r.came(s.t, result{}, besides("Out"))
	_, listing, ok := strings.Cut(r.Out, "Available Commands:\n")
	require.True(s.t, ok, "work --help lists no commands")
	block, _, _ := strings.Cut(listing, "\n\n")

	var verbs []string
	for line := range strings.SplitSeq(block, "\n") {
		if words := strings.Fields(line); len(words) > 0 {
			verbs = append(verbs, words[0])
		}
	}
	return verbs
}

// A word the command line does not take is refused rather than read as something
// else: R2 of docs/rules/command-grammar.md over every verb at once.
func TestCommandRejects(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"two identifiers", []string{"bd-1", "bd-2"}},
		{"two identifiers on go", []string{"go", "bd-1", "bd-2"}},
		{"a verb's flag on go", []string{"go", "bd-1", "--force"}},
		{"two identifiers on switch", []string{"switch", "bd-1", "bd-2"}},
		{"a verb's flag on switch", []string{"switch", "bd-1", "--force"}},
		{"adding two worktrees at once", []string{"add", "scratch", "other"}},
		// carry takes the one identifier and no listing stands in for it.
		{"carrying two worktrees at once", []string{"carry", "scratch", "other"}},
		{"another verb's flag on carry", []string{"carry", "scratch", "--force"}},
		// remove declares --force and nothing else, so a root flag is unknown to it.
		{"a root flag on remove", []string{"remove", "scratch", "--version"}},
		{"removing two worktrees at once", []string{"remove", "scratch", "other"}},
		// move takes the two positions and no flag at all.
		{"a name, a destination and a third word", []string{"move", "scratch", "settled", "over"}},
		{"a root flag on move", []string{"move", "scratch", "--version"}},
		{"another verb's flag on move", []string{"move", "scratch", "--force"}},
		// Listing takes no argument: a name to filter on is go's.
		{"list with a name", []string{"list", "scratch"}},
		{"a root flag on list", []string{"list", "--version"}},
		{"unknown flag", []string{"bd-1", "--turbo"}},
		// The level's value is mandatory: the word behind it is read as that value.
		{"the level with no value", []string{"--log-level"}},
		{"a verb where the level's value goes", []string{"--log-level", "list"}},
		// config carries sub-verbs, and dump takes nothing.
		{"config with a sub-verb it does not carry", []string{"config", "bogus"}},
		{"config dump with an argument", []string{"config", "dump", "extra"}},
		{"a verb's flag on config dump", []string{"config", "dump", "--force"}},
		// edit opens the one file of the user's, and carries no flag at all.
		{"config edit with an argument", []string{"config", "edit", "extra"}},
		{"a verb's flag on config edit", []string{"config", "edit", "--force"}},
		{"init without a shell", []string{"init"}},
		{"init with a shell work does not print", []string{"init", "tcsh"}},
		{"init with two shells", []string{"init", "bash", "fish"}},
	}
	// One repository for the lot: a word the command line refuses never reaches it.
	s := repository(t)
	s.settings(systemsOn("claude", "beads"))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := s.run(tt.args...)

			r.came(t, result{Code: 1}, saidApart)
		})
	}
}
