package cli

import (
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
)

// The agent is on the machine and the integrations leave it out, so the ticket's
// creation is handed back and the agent is asked nothing:
// docs/references/configuration.md#opening-on-a-session.
func TestACreationIsHandedBackWhereTheAgentIsNotNamed(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, nil, "", testenv.Stub{Name: "claude"})
	path := s.at("bd-1")

	r := s.run("add", "bd-1")

	r.came(t, result{Answered: path, Asked: worked("bd-1", path, "bd-1-do-a-thing")})
}

// A creation nothing answered for has nothing to open a session on: the default
// command renders no command line for a name of the user's own, so carry is
// handed back though the agent is named.
func TestACreationNothingAnsweredForIsHandedBack(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "claude"})
	s.settings(integrationsOn("claude"))
	s.dirty()

	r := s.run("carry", "fresh")

	r.came(t, result{Answered: s.at("fresh")})
}

// switch only ever enters, so no creation reaches it: the worktree is handed
// back though the agent is named.
func TestSwitchIsHandedBackThoughTheAgentIsNamed(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "claude"})
	s.settings(integrationsOn("claude"))
	path := s.opened("scratch")

	r := s.run("switch", "scratch")

	r.came(t, result{Answered: path})
}

// go's two moments: it opens a session on the worktree it created, and hands
// that same worktree back on the way in again.
func TestGoOpensASessionOnlyOnTheWorktreeItCreated(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, []string{"claude"}, "", testenv.Stub{Name: "claude"})
	path := s.at("bd-1")

	made := s.hands("go", "bd-1")

	made.came(t, result{Asked: append(worked("bd-1", path, "bd-1-do-a-thing"),
		ticketSessionOn("bd-1", "Do a thing"))})

	again := s.run("go", "bd-1")

	again.came(t, result{Answered: path, Asked: []string{listed}})
}

// carry brings a worktree into being like any other verb: a command that
// renders for a name of the user's own opens a session on what it made.
func TestCarryOpensOnASessionTheCommandRendersFor(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "claude"})
	s.settings(integrationsOn("claude") + commandBlock("claude", "--name={{.Name}}"))
	s.dirty()

	r := s.hands("carry", "carried")

	r.came(t, result{Asked: []string{"claude --name=carried"}})
}
