package testenv_test

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"maps"
	"path/filepath"
	"slices"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// root is where the module sits, from the directory a test runs in.
const root = "../.."

// doors are the packages a test reaches the machine running it through: git,
// which reads the runner's config, the settings, which read the runner's home,
// and testenv, which stands in front of both.
var doors = []string{
	testenv.Module + "/internal/git",
	testenv.Module + "/internal/config",
	testenv.Module + "/internal/testenv",
}

// TestEveryTestPackageReachingGitOrTheSettingsIsIsolated guards the isolation
// testenv.Main writes: it holds for the tests of the package declaring it and
// for nothing else, so a package that reaches a door without declaring it is
// judged against whatever the machine running the tests keeps of its own.
func TestEveryTestPackageReachingGitOrTheSettingsIsIsolated(t *testing.T) {
	pkgs := testenv.Packages(t, root)

	for _, p := range slices.Sorted(maps.Keys(pkgs)) {
		pkg := pkgs[p]
		if len(pkg.TestGoFiles)+len(pkg.XTestGoFiles) == 0 {
			continue
		}
		door, ok := reaches(pkgs, p)
		if !ok || isolated(t, pkg) {
			continue
		}
		t.Errorf("%s reaches %s from its tests without declaring func TestMain(m *testing.M) { testenv.Main(m) }; "+
			"what its tests see is then the runner's git config and settings", p, door)
	}
}

// reaches reports the first door the package's test binary pulls in: what the
// package and its tests import, and what those imports import in turn, because
// the code under test runs git as readily as the test beside it does.
func reaches(pkgs map[string]*build.Package, start string) (string, bool) {
	if slices.Contains(doors, start) {
		return start, true // a door's own tests are the ones about it
	}
	seen := map[string]bool{start: true}
	pkg := pkgs[start]
	queue := slices.Concat(pkg.Imports, pkg.TestImports, pkg.XTestImports)
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if seen[p] {
			continue
		}
		seen[p] = true
		if slices.Contains(doors, p) {
			return p, true
		}
		// Anything outside the module is left where it is: no import path leads from
		// there back to this machine's git.
		if dep, ok := pkgs[p]; ok {
			queue = append(queue, dep.Imports...)
		}
	}
	return "", false
}

// isolated reports whether the package's tests declare a TestMain that reaches
// testenv.Main, wherever in that function it does so: a TestMain with work of
// its own to do first still isolates.
func isolated(t *testing.T, pkg *build.Package) bool {
	t.Helper()
	for _, name := range slices.Concat(pkg.TestGoFiles, pkg.XTestGoFiles) {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(pkg.Dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "TestMain" {
				continue
			}
			if callsMain(fn.Body) {
				return true
			}
		}
	}
	return false
}

// callsMain reports whether the body calls testenv.Main.
func callsMain(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if fn, ok := call.Fun.(*ast.SelectorExpr); ok && fn.Sel.Name == "Main" {
				pkg, ok := fn.X.(*ast.Ident)
				found = found || (ok && pkg.Name == "testenv")
			}
		}
		return !found
	})
	return found
}
