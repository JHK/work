// Package testenv is the ground the tests stand on: throwaway repositories, a
// settings home of the test's own, the files put in either, and stand-ins for
// the tools a test would otherwise reach on PATH.
package testenv

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/run"
)

// Main runs a package's tests against a settings home nobody has written to and
// a git that reads no configuration but the repository's own. It isolates the
// whole process, so the code under test inherits it too.
//
// A package whose tests reach the settings or git declares one TestMain reaching
// this, which TestEveryTestPackageReachingGitOrTheSettingsIsIsolated holds to.
func Main(m *testing.M) {
	dir, err := os.MkdirTemp("", "work-testenv")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	set := func(k, v string) {
		if err := os.Setenv(k, v); err != nil {
			panic(err)
		}
	}
	set("XDG_CONFIG_HOME", dir)
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
	// What work says of the tools it reached for belongs on a terminal, not in a
	// test run; a test that reads one takes it with [Warnings].
	run.Warnings = io.Discard
	m.Run()
}

// Home hands back a settings home the test owns, for the tests that put a
// configuration in one rather than only needing the user's kept out.
func Home(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

// InitRepo hands back a repository holding one empty commit on main.
func InitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	Git(t, dir, "init", "-b", "main")
	Git(t, dir, "commit", "--allow-empty", "-m", "root")
	return dir
}

// Git runs one git command and hands back its stdout, reaching git the way the
// code under test does.
func Git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := run.Output(dir, "git", args...)
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return out
}

// Warnings hands back what work has said of the tools it reached for, which
// [Main] otherwise keeps off a test run.
func Warnings(t *testing.T) func() string {
	t.Helper()
	var said strings.Builder
	was := run.Warnings
	run.Warnings = &said
	t.Cleanup(func() { run.Warnings = was })
	return said.String
}

// Stub is a stand-in for one tool: the name it answers to, what it answers with
// on stdout and on stderr, and the status it exits with. Shell is the way out
// for a stub that has to do more than answer, run after it has recorded and
// before it answers.
type Stub struct {
	Name     string
	Says     string
	Grumbles string
	Exits    int
	Shell    string
}

// Stubs puts a stand-in for each tool on PATH ahead of whatever the machine has
// installed, and hands back what they were asked to run: one line per
// invocation, the tool and its arguments, in the order they ran. One that
// answers reads its answer with cat, so a test emptying PATH first gets a stub
// that can exit but not speak.
func Stubs(t *testing.T, stubs ...Stub) func() []string {
	t.Helper()
	// A tool an earlier case found missing is out for the rest of the process, which
	// would leave the stand-ins put here unasked.
	run.Forget()
	t.Cleanup(run.Forget)
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	for _, s := range stubs {
		// In a file rather than in the script, so quotes and newlines reach the caller
		// as they were written.
		says := ""
		if s.Says != "" {
			answer := filepath.Join(dir, "says-"+s.Name)
			Write(t, answer, s.Says)
			says = fmt.Sprintf("cat %q\n", answer)
		}
		if s.Grumbles != "" {
			answer := filepath.Join(dir, "grumbles-"+s.Name)
			Write(t, answer, s.Grumbles)
			says += fmt.Sprintf("cat %q >&2\n", answer)
		}
		record := fmt.Sprintf("printf '%%s %%s\\n' %q \"$*\" >> %q\n", s.Name, log)
		body := "#!/bin/sh\n" + record + s.Shell + says + fmt.Sprintf("exit %d\n", s.Exits)
		if err := os.WriteFile(filepath.Join(dir, s.Name), []byte(body), 0o755); err != nil {
			t.Fatalf("write %s: %v", s.Name, err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	return func() []string {
		out, err := os.ReadFile(log)
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("read %s: %v", log, err)
		}
		var ran []string
		for line := range strings.SplitSeq(string(out), "\n") {
			// A tool called with no arguments records its name and an empty rest.
			if line = strings.TrimSpace(line); line != "" {
				ran = append(ran, line)
			}
		}
		return ran
	}
}

// Write puts a file where the test says, making the directories above it.
func Write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
