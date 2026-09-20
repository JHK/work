package cli

import (
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
)

// What list prints is one name a line, the main worktree first and a detached one
// under its directory, and no integration is asked for a name or a title.
func TestListPrintsTheWorktreesAndAsksNoIntegration(t *testing.T) {
	s := repository(t)
	// Both integrations on, and every stand-in refuses: one asked would be a line
	// said.
	s.settings(integrationsOn("beads", "github"))
	testenv.Git(t, s.Repo, "branch", "--move", "trunk")
	s.openedOn("worked", "bd-1-do-a-thing")
	s.detached("adrift")

	r := s.run("list")

	r.came(t, result{Out: "trunk\nadrift\nbd-1-do-a-thing\n"})
}

// A repository that will not answer is refused rather than printed as a listing
// with nothing in it: the verb that prints is the one that could read it so.
func TestListOnASilentRepository(t *testing.T) {
	s := repository(t)
	s.Dir = t.TempDir()

	r := s.run("list")

	r.came(t, result{Code: 1}, saidApart)
}
