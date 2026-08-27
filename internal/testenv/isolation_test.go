package testenv_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
)

// root is where the module sits, from the directory a test runs in.
const root = "../.."

// isolation is the package whose initialisation isolates a process, and so the
// one a test binary reaching a door has to pull in.
const isolation = testenv.Module + "/internal/testenv"

// doors are the packages a test reaches the machine running it through: git,
// which reads the runner's config, and the settings and the agent, which read
// the runner's home.
var doors = []string{
	testenv.Module + "/internal/git",
	testenv.Module + "/internal/config",
	testenv.Module + "/internal/action/claude",
}

// TestEveryTestPackageReachingGitOrTheSettingsIsIsolated guards R6 of
// docs/rules/test-isolation.md: every package whose test binary reaches a door
// pulls in testenv, whose initialisation isolates the process.
func TestEveryTestPackageReachingGitOrTheSettingsIsIsolated(t *testing.T) {
	// A test binary is what go list writes under .Name main, and its dependencies are
	// the whole closure, so a package reaching a door through what it imports is
	// judged like one importing the door itself.
	for _, line := range testenv.Listed(t, root, "-test",
		"-f", `{{if eq .Name "main"}}{{.ImportPath}}{{range .Deps}} {{.}}{{end}}{{end}}`, "./...") {
		linked := strings.Fields(line)
		if len(linked) == 0 || !strings.HasSuffix(linked[0], ".test") || slices.Contains(linked, isolation) {
			continue
		}
		for _, door := range doors {
			if slices.Contains(linked, door) {
				t.Errorf("%s reaches %s without pulling in %s; what its tests see is then "+
					"the runner's git config and settings", linked[0], door, isolation)
			}
		}
	}
}
