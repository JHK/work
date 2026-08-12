package work

import (
	"go/build"
	"slices"
	"strings"
	"testing"
)

// reachable is every package the core may import directly: the vocabulary both
// sides of the seams speak, git because worktrees are git's, and the settings.
var reachable = []string{
	"github.com/JHK/work-cli/internal/config",
	"github.com/JHK/work-cli/internal/git",
	"github.com/JHK/work-cli/internal/worktree",
}

// TestCoreReachesNothingElse guards the separation
// docs/projects/worktree-switcher-core.md is for: an implementation the core
// names is a capability living in the file that should be the most stable.
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

// standard reports whether an import path names a standard library package.
// Every other path begins with a module path, whose first element is a domain.
func standard(path string) bool {
	root, _, _ := strings.Cut(path, "/")
	return !strings.Contains(root, ".")
}
