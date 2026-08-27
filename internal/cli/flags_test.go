package cli

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/wiring"
	"github.com/JHK/work-cli/internal/worktree"
	"github.com/stretchr/testify/require"
)

// openingVerbs hand a worktree to something; creatingVerbs are the ones among
// them that can bring one into being.
var (
	openingVerbs  = []string{"go", "switch", "add"}
	creatingVerbs = []string{"go", "add"}
)

// The systems the settings wired spell the flags, and --help is where a reader
// is given the set: docs/references/cli.md#open-on-flags.
func TestTheWiredSystemsSpellTheFlagsHelpPrints(t *testing.T) {
	s := repository(t)
	// The two systems that spell a flag at all: an opener named by one, an action
	// declined by one.
	s.settings(claudeOn + trackerOn)

	r := s.run("go", "--help")

	r.came(t, result{}, besides("Out"))
	for _, want := range []string{
		`--claude\s+hand the worktree to claude\b`,
		`--shell\s+stand in the worktree\b`,
		`--no-claim\s+create the worktree without claiming the ticket\b`,
	} {
		require.Regexp(t, want, r.Out, "--help prints no flag for a system the settings wired")
	}
}

// Every system the wiring has, on the verbs that can run it and on no other:
// an opener names what a worktree opens on, and an action is called off only
// where one can come into being. Nothing spells a flag on the root, which the
// bare form would then take ahead of go. R2 of docs/rules/command-grammar.md.
func TestWhereEachSystemsFlagIsDeclared(t *testing.T) {
	s := repository(t)
	s.settings(claudeOn + trackerOn + miseOn + forgeOn)
	cfg, err := config.Load()
	require.NoError(t, err, "the settings")
	sys := wiring.Wire(s.Repo, s.Repo, cfg)

	// Every command that carries a flag at all, so one landing where the system
	// cannot run is caught beside one missing where it can.
	declared := map[string][]string{}
	for _, args := range [][]string{nil, {"go"}, {"switch"}, {"add"}, {"remove"}, {"move"},
		{"list"}, {"init"}, {"config"}, {"config", "dump"}, {"config", "edit"}} {
		declared[strings.Join(args, " ")] = s.prints(args...)
	}
	for _, op := range sys.Openers {
		flag, ok := spells(op)
		require.True(t, ok, "%s spells no flag, so nothing on the command line reaches it", op.Name())
		declaredOn(t, declared, flag, openingVerbs)
	}
	for _, a := range sys.Actions {
		if flag, ok := spells(a); ok {
			declaredOn(t, declared, flag, creatingVerbs)
			continue
		}
		// An action with no flag of its own runs whenever a worktree does, so its name
		// is nothing to type.
		declaredOn(t, declared, a.Name(), nil)
	}
}

// declaredOn fails the case unless that flag is offered by each of these verbs
// and by no other command that carries one, the root among them.
func declaredOn(t *testing.T, declared map[string][]string, flag string, verbs []string) {
	t.Helper()
	for where, flags := range declared {
		require.Equal(t, slices.Contains(verbs, where), slices.Contains(flags, flag),
			"work %s --help offers %v; want --%s among them: %v", where, flags, flag, slices.Contains(verbs, where))
	}
}

// spells is the flag a system answers to, and whether it spells one at all.
func spells(s worktree.System) (string, bool) {
	f, ok := s.(worktree.Flagged)
	if !ok {
		return "", false
	}
	flag, _ := f.Flag()
	return flag, true
}

// written is how --help writes a flag, which is the one place a reader is given
// the set: a flag declared and hidden is written nowhere.
var written = regexp.MustCompile(`(?m)^\s+(?:-\w, )?--([\w-]+)`)

// prints is the flags a command offers a reader, which is what --help lists.
func (s *session) prints(args ...string) []string {
	s.t.Helper()
	r := s.run(append(args, "--help")...)
	r.came(s.t, result{}, besides("Out"))
	var flags []string
	for _, m := range written.FindAllStringSubmatch(r.Out, -1) {
		flags = append(flags, m[1])
	}
	return flags
}

// A flag an action used to answer to is refused with the one it answers to now,
// wherever the set is carried, and offered nowhere: --help is what a reader is
// given the spellings from.
func TestTheRenamedFlagIsRefusedByItsNewName(t *testing.T) {
	s := repository(t)
	s.settings(claudeOn)

	for _, args := range [][]string{{"bd-1", "--agent"}, {"go", "--agent", "bd-1"}, {"switch", "--agent", "bd-1"}, {"add", "--agent", "scratch"}} {
		r := s.run(args...)

		r.refused(t, "--claude")
	}
	for _, verb := range openingVerbs {
		require.NotContains(t, s.prints(verb), "agent", "work %s offers --agent, which is a spelling it no longer takes", verb)
	}
}

// A word the command line does not take is refused rather than read as something
// else: R2 of docs/rules/command-grammar.md over every verb at once.
func TestCommandRejects(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"--start is gone", []string{"bd-1", "--start"}},
		{"--model is gone", []string{"bd-1", "--model", "opus"}},
		{"--effort is gone", []string{"--effort=high", "bd-1"}},
		{"--editor is gone", []string{"bd-1", "--editor"}},
		{"--diff is gone", []string{"bd-1", "--diff"}},
		{"--ask is gone", []string{"bd-1", "--ask"}},
		{"claude and a shell at once", []string{"bd-1", "--claude", "--shell"}},
		{"two identifiers", []string{"bd-1", "bd-2"}},
		{"two identifiers on go", []string{"go", "bd-1", "bd-2"}},
		{"two actions on go at once", []string{"go", "bd-1", "--shell", "--claude"}},
		{"a verb's flag on go", []string{"go", "bd-1", "--force"}},
		{"two identifiers on switch", []string{"switch", "bd-1", "bd-2"}},
		{"two actions on switch at once", []string{"switch", "bd-1", "--shell", "--claude"}},
		{"a verb's flag on switch", []string{"switch", "bd-1", "--force"}},
		// switch only enters, so no action of its own ever runs to be called off.
		{"declining a claim on switch", []string{"switch", "bd-1", "--no-claim"}},
		// Creating is a verb now.
		{"--create is gone", []string{"scratch", "--create"}},
		{"adding two worktrees at once", []string{"add", "scratch", "other"}},
		{"two actions on add at once", []string{"add", "scratch", "--shell", "--claude"}},
		// Removing is a verb now, and --force went with it.
		{"--delete is gone", []string{"scratch", "--delete"}},
		{"--force is gone from the root", []string{"scratch", "--force"}},
		// remove declares --force and nothing else, so a root flag is unknown to it.
		{"a root flag on remove", []string{"remove", "scratch", "--shell"}},
		{"removing two worktrees at once", []string{"remove", "scratch", "other"}},
		// move takes the two positions and no flag at all.
		{"a name, a destination and a third word", []string{"move", "scratch", "settled", "over"}},
		{"a root flag on move", []string{"move", "scratch", "--shell"}},
		{"another verb's flag on move", []string{"move", "scratch", "--force"}},
		// Listing takes no argument: a name to filter on is go's.
		{"list with a name", []string{"list", "scratch"}},
		{"a root flag on list", []string{"list", "--shell"}},
		{"unknown flag", []string{"bd-1", "--turbo"}},
		// The level is a flag of its own now, not a verbosity that raises one, and its
		// value is mandatory: the word behind it is read as that value.
		{"--verbose is gone", []string{"--verbose", "scratch"}},
		{"--verbose at a level is gone", []string{"--verbose=debug", "list"}},
		{"the level with no value", []string{"--log-level"}},
		{"a verb where the level's value goes", []string{"--log-level", "list"}},
		// config carries sub-verbs, and dump takes nothing.
		{"config with a sub-verb it does not carry", []string{"config", "bogus"}},
		{"config dump with an argument", []string{"config", "dump", "extra"}},
		{"a verb's flag on config dump", []string{"config", "dump", "--force"}},
		// edit opens the one file of the user's, and carries no flag at all.
		{"config edit with an argument", []string{"config", "edit", "extra"}},
		{"a verb's flag on config edit", []string{"config", "edit", "--shell"}},
		{"init without a shell", []string{"init"}},
		{"init with a shell work does not print", []string{"init", "tcsh"}},
		{"init with two shells", []string{"init", "bash", "fish"}},
	}
	// One repository for the lot: a word the command line refuses never reaches it.
	s := repository(t)
	s.settings(claudeOn + trackerOn)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := s.run(tt.args...)

			r.came(t, result{Code: 1}, apart)
		})
	}
}
