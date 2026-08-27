package work

import (
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
)

// root is where the module sits, from the directory these tests run in.
const root = "../.."

// reachable is every package the core may import directly: the vocabulary both
// sides of the seams speak, git because worktrees are git's, and the settings.
var reachable = []string{
	testenv.Module + "/internal/config",
	testenv.Module + "/internal/git",
	testenv.Module + "/internal/worktree",
}

// TestCoreReachesNothingElse guards R3 of docs/rules/package-boundaries.md: an
// implementation the core names is a capability living in the file that should
// be the most stable.
func TestCoreReachesNothingElse(t *testing.T) {
	for _, path := range testenv.Listed(t, root, "-f", `{{join .Imports "\n"}}`, "./internal/work") {
		if standard(path) || slices.Contains(reachable, path) {
			continue
		}
		t.Errorf("the core imports %s; it may reach only the standard library and %s",
			path, strings.Join(reachable, ", "))
	}
}

// seams are the two trees the implementations live in, as the prefix the
// packages of each are imported under.
var seams = []string{
	testenv.Module + "/internal/action/",
	testenv.Module + "/internal/resolve/",
}

// TestNoSystemReachesAnother guards R4 of docs/rules/package-boundaries.md: what
// one system does is its own, and an implementation that named another would put
// the second one's work into the first one's answer.
func TestNoSystemReachesAnother(t *testing.T) {
	read := map[string]bool{}
	// Test files are held to the rule too, so all three compilations are read.
	for _, line := range testenv.Listed(t, root, "-f",
		`{{.ImportPath}}{{range .Imports}} {{.}}{{end}}{{range .TestImports}} {{.}}{{end}}{{range .XTestImports}} {{.}}{{end}}`, "./...") {
		paths := strings.Fields(line)
		seam, from := system(paths[0])
		if from == "" {
			continue
		}
		read[seam] = true
		for _, imported := range paths[1:] {
			if _, to := system(imported); to != "" && to != from {
				t.Errorf("%s imports %s; a system reaches no other system's package", paths[0], imported)
			}
		}
	}
	// A prefix gone stale matches nothing, which would leave its whole tree
	// unread and the test passing.
	for _, seam := range seams {
		if !read[seam] {
			t.Errorf("no package sits under %s; the rule was read over nothing there", seam)
		}
	}
}

// system names the seam an import path sits behind and the system it belongs to,
// both empty for a path behind neither. A system is one directory under a seam,
// with whatever it holds, so its own packages read as one rather than as peers.
func system(path string) (seam, name string) {
	for _, s := range seams {
		if rest, ok := strings.CutPrefix(path, s); ok {
			under, _, _ := strings.Cut(rest, "/")
			return s, s + under
		}
	}
	return "", ""
}

// standard reports whether an import path names a standard library package.
// Every other path begins with a module path, whose first element is a domain.
func standard(path string) bool {
	root, _, _ := strings.Cut(path, "/")
	return !strings.Contains(root, ".")
}
