package testenv

import (
	"go/build"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// Module is the module every package of this repository sits under.
const Module = "github.com/JHK/work-cli"

// Packages reads every package of the module, keyed by import path. root is the
// module's root directory, from the directory the test runs in. Imports are read
// with UseAllFiles, so a build tag hides none of them from a caller judging them.
func Packages(t *testing.T, root string) map[string]*build.Package {
	t.Helper()
	ctx := build.Default
	ctx.UseAllFiles = true
	pkgs := map[string]*build.Package{}
	err := filepath.WalkDir(root, func(dir string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		// The worktrees of this same module sit under a dot directory, as the
		// repository itself does.
		if name := d.Name(); dir != root && (strings.HasPrefix(name, ".") || name == "testdata") {
			return fs.SkipDir
		}
		pkg, err := ctx.ImportDir(dir, 0)
		if _, empty := err.(*build.NoGoError); empty {
			return nil
		}
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return err
		}
		pkgs[path.Join(Module, filepath.ToSlash(rel))] = pkg
		return nil
	})
	if err != nil {
		t.Fatalf("read the module's packages: %v", err)
	}
	return pkgs
}
