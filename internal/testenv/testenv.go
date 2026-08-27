// Package testenv is the ground the tests stand on: throwaway repositories, a
// settings home of the test's own, the files put in either, and stand-ins for
// the tools a test would otherwise reach on PATH. Importing it is what isolates
// a process, so the code under test sees none of the machine it runs on either:
// docs/rules/test-isolation.md.
package testenv

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/JHK/work-cli/internal/run"
)

// init puts the process on a settings home and a home directory nobody has
// written to, on a git that reads no configuration but the repository's own,
// and on a log that says nothing at all. A process reached under a stand-in's
// name answers as that tool here and runs no test.
func init() {
	standIn()
	slog.SetDefault(slog.New(slog.DiscardHandler))
	set := func(k, v string) {
		if err := os.Setenv(k, v); err != nil {
			panic(err)
		}
	}
	// A child of a test binary stands where the case that spawned it left it.
	if dir := os.Getenv(inherited); dir != "" {
		ground = dir
		return
	}
	dir, err := os.MkdirTemp("", "work-testenv")
	if err != nil {
		panic(err)
	}
	ground, owned = dir, dir
	set(inherited, dir)
	settings := filepath.Join(dir, "settings")
	if err := os.Mkdir(settings, 0o755); err != nil {
		panic(err)
	}
	set("XDG_CONFIG_HOME", settings)
	home := filepath.Join(dir, "home")
	if err := os.Mkdir(home, 0o755); err != nil {
		panic(err)
	}
	set("HOME", home)
	// A shell that ran the tests from inside work named a file of its own in
	// shim.CDFile, which a front end under test would answer into.
	set("WORK_CD_FILE", "")
	// The runner's own git config would otherwise decide what git does: gpgsign on
	// an empty commit, showUntrackedFiles on whether a worktree can be removed.
	set("GIT_CONFIG_GLOBAL", os.DevNull)
	set("GIT_CONFIG_SYSTEM", os.DevNull)
	// git translates its refusals, and a test reading one reads the English.
	set("LC_ALL", "C")
	// An identity, because a repository reading no config has none to commit under.
	set("GIT_AUTHOR_NAME", "t")
	set("GIT_AUTHOR_EMAIL", "t@t")
	set("GIT_COMMITTER_NAME", "t")
	set("GIT_COMMITTER_EMAIL", "t@t")
}

// Main takes the ground away once a package's tests are done with it. A package
// importing this one declares func TestMain(m *testing.M) { testenv.Main(m) },
// which is what leaves nothing behind in the temporary directory.
func Main(m *testing.M) {
	defer func() {
		if owned != "" {
			_ = os.RemoveAll(owned)
		}
	}()
	m.Run()
}

// inherited names the ground to a child of a test binary.
const inherited = "WORK_TESTENV_GROUND"

// ground is the directory this process holds what its tests have in common in,
// and owned that same directory where this process is the one that made it.
var ground, owned string

// template is the repository [InitRepo] copies: one empty commit on main, built
// once for the process. A repository this new names no path of its own, so a
// copy of it stands wherever it is put.
var template = sync.OnceValues(func() (string, error) {
	dir := filepath.Join(ground, "template")
	if err := os.Mkdir(dir, 0o755); err != nil {
		return "", err
	}
	for _, args := range [][]string{{"init", "-b", "main"}, {"commit", "--allow-empty", "-m", "root"}} {
		if _, err := run.Output(dir, "git", args...); err != nil {
			return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
	}
	// The samples git seeds are most of what each copy would carry, and nothing
	// runs them.
	if err := os.RemoveAll(filepath.Join(dir, ".git", "hooks")); err != nil {
		return "", err
	}
	return dir, nil
})

// Home hands back a settings home the test owns, for the tests that put a
// configuration in one rather than only needing the user's kept out.
func Home(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

// Settings puts a configuration in a settings home the test owns, and hands
// back the file it was written to. Each call takes a home of its own.
func Settings(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(Home(t), "work", "config.toml")
	Write(t, path, body)
	return path
}

// InitRepo hands back a repository holding one empty commit on main.
func InitRepo(t *testing.T) string {
	t.Helper()
	src, err := template()
	if err != nil {
		t.Fatalf("testenv.InitRepo: %v", err)
	}
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(src)); err != nil {
		t.Fatalf("testenv.InitRepo: %v", err)
	}
	return dir
}

// absolute stops the test where the path a helper was handed is not absolute,
// naming that helper.
func absolute(t testing.TB, helper, path string) {
	t.Helper()
	if !filepath.IsAbs(path) {
		t.Fatalf("%s: %q is not an absolute path", helper, path)
	}
}

// Git runs one git command in that directory and hands back its stdout,
// reaching git the way the code under test does. A directory that is not
// absolute is refused.
func Git(t testing.TB, dir string, args ...string) string {
	t.Helper()
	absolute(t, "testenv.Git", dir)
	out, err := run.Output(dir, "git", args...)
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return out
}

// Terminal hands back a writer that is a terminal, for a case turning on what
// the reader is, the case skipped where the machine opens none. Nothing drains
// it, so a case sending more than a line or two through it blocks.
func Terminal(t testing.TB) io.Writer {
	t.Helper()
	f, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("no pseudo-terminal to open: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// Write puts a file where the test says, making the directories above it. A
// path that is not absolute is refused.
func Write(t testing.TB, path, body string) {
	t.Helper()
	absolute(t, "testenv.Write", path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
