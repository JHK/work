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
		name, body, verb string
		// session says the run opened a conversation rather than handing the
		// worktree back to the shell.
		session bool
	}{
		{"add, which the default names", on("claude"), "add", true},
		{"carry, which the default leaves out", on("claude"), "carry", false},
		{"carry, where the key names it", agentOn + "on-creation = [\"carry\"]\n", "carry", true},
		{"add, where the key names carry alone", agentOn + "on-creation = [\"carry\"]\n", "add", false},
		{"add, where the key names nothing", agentOn + "on-creation = []\n", "add", false},
		// The key names the verb, and the table it sits in never switched the agent on.
		{"add, with the agent off", "[claude]\non-creation = [\"add\"]\n", "add", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := repository(t, testenv.Stub{Name: "claude"})
			s.settings(tt.body)
			// carry refuses a checkout with nothing in hand.
			if tt.verb == "carry" {
				s.dirty()
			}

			// Only a run that opens a session replaces the process, and only that one
			// has to be watched from a child.
			if tt.session {
				s.hands(tt.verb, "fresh").came(t, result{Asked: []string{sessionOn("fresh")}})
				return
			}
			s.run(tt.verb, "fresh").came(t, result{Answered: s.at("fresh")})
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
