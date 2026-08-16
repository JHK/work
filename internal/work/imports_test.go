package work

import (
	"go/build"
	"maps"
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

// TestCoreReachesNothingElse guards R3 of docs/rules/package-boundaries.md: an
// implementation the core names is a capability living in the file that should
// be the most stable.
func TestCoreReachesNothingElse(t *testing.T) {
	ctx := build.Default
	ctx.UseAllFiles = true // a build tag is not a way out of the rule
	pkg, err := ctx.ImportDir(".", 0)
	if err != nil {
		t.Fatalf("read the core's imports: %v", err)
	}
	for _, path := range pkg.Imports {
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
	pkgs := testenv.Packages(t, "../..")

	read := map[string]bool{}
	for _, path := range slices.Sorted(maps.Keys(pkgs)) {
		seam, from := system(path)
		if from == "" {
			continue
		}
		read[seam] = true
		pkg := pkgs[path]
		// Test files are held to the rule too, so all three compilations are read.
		for _, imported := range slices.Concat(pkg.Imports, pkg.TestImports, pkg.XTestImports) {
			if _, to := system(imported); to != "" && to != from {
				t.Errorf("%s imports %s; a system reaches no other system's package", path, imported)
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
