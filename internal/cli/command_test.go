package cli

import (
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
)

// cutting is the agent as a stand-in that brackets each argument it was handed,
// so a case reads where the block was cut rather than the words it rendered.
var cutting = testenv.Stub{Name: "claude", Shell: `printf '[%s]' "$@"`}

// Each non-blank line of a block is one argument, trimmed, whatever whitespace
// it carries and however deep it is written.
func TestEachLineOfABlockIsOneArgument(t *testing.T) {
	s := repository(t, cutting)
	s.settings(systemsOn("claude") + commandBlock("claude", "  --name=one two", "\ta b c"))

	r := s.hands("add", "scratch")

	r.came(t, result{Out: "[--name=one two][a b c]", Asked: []string{"claude --name=one two a b c"}})
}

// guarding is a whole command behind one conditional on the source, which is
// what a chain meant for the tracker's worktrees alone is written as.
func guarding(source string) string {
	return commandBlock(`{{if eq .Source "`+source+`"}}`, "claude", "--name={{.ID}}", "/start {{.ID}}", "{{end}}")
}

// One conditional opening the block and closing it guards every line between:
// the whole command runs where it holds.
func TestOneConditionalGuardsEveryLineOfABlock(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, []string{"claude"}, guarding("beads"), cutting)

	r := s.hands("add", "bd-1")

	r.came(t, result{Out: "[--name=bd-1][/start bd-1]",
		Asked: append(worked("bd-1", s.at("bd-1"), "bd-1-do-a-thing"), "claude --name=bd-1 /start bd-1")})
}

// A worktree the guard renders false for is created and its ticket claimed, then
// handed back the way one nothing opens on is.
func TestAWorktreeWhoseGuardRendersFalseIsHandedBack(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, []string{"claude"}, guarding("github"))

	r := s.run("add", "bd-1")

	r.came(t, result{Answered: s.at("bd-1"), Asked: worked("bd-1", s.at("bd-1"), "bd-1-do-a-thing")})
}

// Only the shell a script line names splits it, and squote hands that shell one
// word whatever the value holds.
func TestACommandQuotesAValueIntoAShellString(t *testing.T) {
	awkward := with(doable, func(t *ticket) { t.Title = `Do a $HOME thing it's "own" way` })
	s := tracking(t, []ticket{awkward}, []ticket{awkward}, []string{"claude"},
		commandBlock("sh", "-c", "claude {{.Subject | squote}}"), cutting)

	r := s.hands("add", "bd-1")

	subject := "bd-1: " + awkward.Title
	r.came(t, result{Out: "[" + subject + "]",
		Asked: append(worked("bd-1", s.at("bd-1"), "bd-1-do-a-home-thing-it-s-own-way"), "claude "+subject)})
}

// An empty value quotes to a word all the same, so a script written that way
// never loses a word to it.
func TestAnEmptyValueQuotesToAnEmptyWord(t *testing.T) {
	untitled := with(doable, func(b *ticket) { b.Title = "" })
	s := tracking(t, []ticket{untitled}, []ticket{untitled}, []string{"claude"},
		commandBlock("sh", "-c", "claude {{.Title | squote}} last"), cutting)

	r := s.hands("add", "bd-1")

	r.came(t, result{Out: "[][last]", Asked: append(worked("bd-1", s.at("bd-1"), "bd-1"), "claude  last")})
}
