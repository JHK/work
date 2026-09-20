package module

import (
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
)

// reachable is every package the core may import directly: the vocabulary both
// sides of the seams speak, git because worktrees are git's, and the settings.
var reachable = []string{
	testenv.Module + "/internal/config",
	testenv.Module + "/internal/git",
	testenv.Module + "/internal/worktree",
}

// R3 of docs/rules/package-boundaries.md: an implementation the core names is a
// capability living in the file that should be the most stable.
func TestCoreReachesNothingElse(t *testing.T) {
	for _, path := range testenv.Listed(t, root, "-f", `{{join .Imports "\n"}}`, "./internal/work") {
		if standard(path) || slices.Contains(reachable, path) {
			continue
		}
		t.Errorf("the core imports %s; it may reach only the standard library and %s",
			path, strings.Join(reachable, ", "))
	}
}

// R8 of docs/rules/package-boundaries.md: every package speaks the vocabulary,
// so a word reaching one package would carry that package into all of them.
func TestTheVocabularyReachesNothing(t *testing.T) {
	for _, path := range testenv.Listed(t, root, "-f", `{{join .Imports "\n"}}`, "./internal/worktree") {
		if standard(path) {
			continue
		}
		t.Errorf("the vocabulary imports %s; it may reach only the standard library", path)
	}
}

// integrations is the tree the integrations live in, as the prefix the packages
// of each are imported under.
const integrations = testenv.Module + "/internal/integration/"

// R4 of docs/rules/package-boundaries.md: what one integration does is its own,
// and an implementation that named another would put the second one's work into
// the first one's answer.
func TestNoIntegrationReachesAnother(t *testing.T) {
	// The tree is listed rather than the module, so a prefix gone stale is a go list
	// that fails rather than a rule read over nothing. Test files are held to the
	// rule too, so all three compilations are read.
	for _, line := range testenv.Listed(t, root, "-f",
		`{{.ImportPath}}{{range .Imports}} {{.}}{{end}}{{range .TestImports}} {{.}}{{end}}{{range .XTestImports}} {{.}}{{end}}`,
		"./internal/integration/...") {
		paths := strings.Fields(line)
		from := integration(paths[0])
		if from == "" {
			continue
		}
		for _, imported := range paths[1:] {
			if to := integration(imported); to != "" && to != from {
				t.Errorf("%s imports %s; an integration reaches no other integration's package", paths[0], imported)
			}
		}
	}
}

// integration names the integration an import path belongs to, empty for the
// tree's own root, which is what two or more of them share. An integration is
// one directory under the tree, with whatever it holds, so its own packages read
// as one rather than as peers.
func integration(path string) string {
	rest, ok := strings.CutPrefix(path, integrations)
	if !ok {
		return ""
	}
	under, _, _ := strings.Cut(rest, "/")
	return under
}

// standard reports whether an import path names a standard library package.
// Every other path begins with a module path, whose first element is a domain.
func standard(path string) bool {
	root, _, _ := strings.Cut(path, "/")
	return !strings.Contains(root, ".")
}
