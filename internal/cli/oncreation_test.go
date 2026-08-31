package cli

import (
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
)

// A creation under a verb claude.on-creation names opens a session, and every
// other creation is handed back: docs/references/configuration.md. go is the
// other verb the key names, and takes an identifier a system answers for.
func TestWhatAVerbsCreationOpensOn(t *testing.T) {
	tests := []struct {
		name, body      string
		systemsBesideBd []string
		opensASession   bool
	}{
		{"the default, which names add", "", []string{"claude"}, true},
		{"the key naming go alone", claudeTable + "on-creation = [\"go\"]\n", []string{"claude"}, false},
		{"the key naming nothing", claudeTable + "on-creation = []\n", []string{"claude"}, false},
		{"the agent left out of the systems list", claudeTable + "on-creation = [\"add\"]\n", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tracking(t, []ticket{doable}, []ticket{doable}, tt.systemsBesideBd, tt.body,
				testenv.Stub{Name: "claude"})
			path := s.at("bd-1")
			asked := worked("bd-1", path, "bd-1-do-a-thing")

			// A run that opens a session replaces the process, so it is watched from a child.
			if tt.opensASession {
				s.hands("add", "bd-1").came(t, result{Asked: append(asked, ticketSessionOn("bd-1", "Do a thing"))})
				return
			}
			s.run("add", "bd-1").came(t, result{Answered: path, Asked: asked})
		})
	}
}

// A creation nothing answered for has nothing to open a session on, so it is
// handed back whatever claude.on-creation names.
func TestACreationNothingAnsweredForIsHandedBack(t *testing.T) {
	tests := []struct{ name, body, verb string }{
		{"add, which the key names by default", systemsOn("claude"), "add"},
		{"carry, where the key names it", agentOn + "on-creation = [\"carry\"]\n", "carry"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t, testenv.Stub{Name: "claude"})
			s.settings(tt.body)
			// carry refuses a checkout with nothing in hand.
			if tt.verb == "carry" {
				s.dirty()
			}

			r := s.run(tt.verb, "fresh")

			r.came(t, result{Answered: s.at("fresh")})
		})
	}
}

// switch only ever enters, so no key reaches it: naming every verb there is
// still leaves it handing the worktree back.
func TestSwitchIsHandedBackWhateverTheKeyNames(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "claude"})
	s.settings(agentOn + "on-creation = [\"add\", \"carry\", \"go\"]\n")
	path := s.opened("scratch")

	r := s.run("switch", "scratch")

	r.came(t, result{Answered: path})
}

// The one key over both of go's moments: it opens a session on the worktree it
// created, and hands that same worktree back on the way in again.
func TestGoOpensASessionOnlyOnTheWorktreeItCreated(t *testing.T) {
	s := tracking(t, []ticket{doable}, []ticket{doable}, []string{"claude"}, "", testenv.Stub{Name: "claude"})
	path := s.at("bd-1")

	made := s.hands("go", "bd-1")

	made.came(t, result{Asked: append(worked("bd-1", path, "bd-1-do-a-thing"),
		ticketSessionOn("bd-1", "Do a thing"))})

	again := s.run("go", "bd-1")

	again.came(t, result{Answered: path, Asked: []string{listed}})
}
