package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// A shell that never sourced the function named no file, so the worktree is
// printed instead: the invocation still says where the work is.
func TestAShellWithoutTheIntegrationIsPrintedTheWorktree(t *testing.T) {
	s := repository(t)
	path := s.opened("scratch")
	s.Shim = ""

	r := s.run("switch", "scratch")

	r.came(t, result{Out: path + "\n"})
}

// A terminal reading that path is a person, who hears of the one install step
// that was skipped: internal/shim.
func TestAPersonReadingThePathIsToldOfTheIntegration(t *testing.T) {
	s := repository(t)
	s.opened("scratch")
	s.Shim = ""

	r := s.reads(testenv.Terminal(t), "switch", "scratch")

	r.came(t, result{Warned: []string{"the shell integration is not sourced, so your shell stays where it is; see work init --help"}})
}

// A file the shell named that work cannot write is a refusal rather than a
// silent print: the shell would otherwise stay where it was with nothing said.
func TestAFileTheShellNamedThatCannotBeWrittenIsRefused(t *testing.T) {
	s := repository(t)
	s.opened("scratch")
	s.Shim = filepath.Join(t.TempDir(), "nowhere", "answer")

	r := s.run("switch", "scratch")

	r.refused(t, s.Shim)
}

// A worktree that opens on a command the machine does not have is refused, and
// the shell is left standing where it typed: internal/worktree.
func TestAWorktreeOpeningOnACommandThatIsNotThereIsRefused(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, []string{"claude"},
		claudeTable+"command = [\"no-such-binary-xyz\"]\n")

	r := s.run("add", "bd-1")

	r.came(t, result{Code: 1, Asked: worked("bd-1", s.at("bd-1"), "bd-1-do-a-thing")}, apart)
	r.saying(t, "no-such-binary-xyz")
	here, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, s.Dir, here, "the refused handoff left the process elsewhere")
}

// A worktree that opens on a command hands the terminal to it, running inside
// that worktree, and work says nothing of its own on the way past.
func TestAWorktreeThatOpensOnACommandHandsItTheTerminal(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, []string{"claude"}, "",
		testenv.Stub{Name: "claude", Shell: "git rev-parse --show-toplevel"})

	r := s.hands("add", "bd-1")

	r.came(t, result{Out: s.at("bd-1") + "\n", Asked: append(worked("bd-1", s.at("bd-1"), "bd-1-do-a-thing"),
		ticketSessionOn("bd-1", "Do a thing"))})
}

// The file names one invocation, so what the terminal goes to must not inherit
// it: a work run inside that command would answer into a shell done waiting.
func TestTheCommandTheTerminalGoesToDoesNotInheritTheShellsFile(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, []string{"claude"}, "",
		testenv.Stub{Name: "claude", Shell: `printf '[%s]' "$WORK_CD_FILE"`})

	r := s.hands("add", "bd-1")

	r.came(t, result{Out: "[]", Asked: append(worked("bd-1", s.at("bd-1"), "bd-1-do-a-thing"),
		ticketSessionOn("bd-1", "Do a thing"))})
}

// With no identifier the picker stands in for one, over the worktrees open less
// the one the shell stands in.
func TestSwitchWithNoIdentifierTakesThePickersRow(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "fzf", Says: "0\tscratch\n"})
	path := s.opened("scratch")

	r := s.run("switch")

	r.came(t, result{Answered: path, Asked: []string{putUp}})
}

// Nothing comes into being on the way back in, so the ticket a worktree was made
// for is not claimed again: the tracker is asked only what names the worktree.
func TestSwitchClaimsNothingOnTheWayBackIn(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, nil, "")
	path := s.openedOn("bd-1", "bd-1-do-a-thing")

	r := s.run("switch", "bd-1")

	r.came(t, result{Answered: path, Asked: []string{listed}})
}

// switch only enters, so what it cannot enter it refuses: the refusal names add
// for a place with no worktree, and names no verb for a name nothing answers for.
func TestSwitchRefuses(t *testing.T) {
	tests := []struct {
		name string
		// tracked wires the tracker, for a row whose identifier is a ticket.
		tracked bool
		args    []string
		said    string
	}{
		{"a ticket with no worktree open", true, []string{"switch", "bd-1"},
			"bd-1 has no worktree open; work add bd-1 makes one"},
		{"a name nothing answers for", false, []string{"switch", "typo"},
			`nothing answers for "typo"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, asked := repository(t), []string(nil)
			if tt.tracked {
				s = tracking(t, []ticket{doable}, []ticket{doable}, nil, "")
				asked = []string{listed}
			}

			r := s.run(tt.args...)

			r.came(t, result{Code: 1, Errored: []string{tt.said}, Asked: asked})
		})
	}
}

// A worktree git still lists but nobody can enter is refused: nothing execs on
// this path, so an answer nobody checked would be a shell left where it was.
func TestSwitchRefusesAWorktreeThatIsNoLongerThere(t *testing.T) {
	s := repository(t)
	path := s.opened("scratch")
	require.NoError(t, os.RemoveAll(path), "empty the worktree")

	r := s.run("switch", "scratch")

	r.refused(t, path)
}
