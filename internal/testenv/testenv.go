// Package testenv is the ground the tests stand on: throwaway repositories, a
// settings home of the test's own, the files put in either, and stand-ins for
// the tools a test would otherwise reach on PATH, with whatever the machine
// running the tests keeps of its own kept out of all four.
package testenv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/run"
)

// Main runs a package's tests against a settings home nobody has written to and
// a git that reads no configuration but the repository's own, so a test that
// never thinks about the machine it runs on cannot be answered by it.
//
// It is the whole process it isolates, which is what puts it here rather than at
// each site that runs git: the code under test spawns git itself, through
// internal/run, and inherits this environment as the tests below do.
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
	// The runner's own git config would otherwise decide what git does:
	// commit.gpgsign makes an empty commit depend on a working key, and
	// status.showUntrackedFiles on whether a worktree refuses to be removed.
	set("GIT_CONFIG_GLOBAL", os.DevNull)
	set("GIT_CONFIG_SYSTEM", os.DevNull)
	// An identity, because a repository reading no config has none to commit under.
	set("GIT_AUTHOR_NAME", "t")
	set("GIT_AUTHOR_EMAIL", "t@t")
	set("GIT_COMMITTER_NAME", "t")
	set("GIT_COMMITTER_EMAIL", "t@t")
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

// Git runs one git command and hands back its stdout, for the callers that are
// asking git something rather than telling it something. It reaches git the way
// the code under test does, through the environment Main left.
func Git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := run.Output(dir, "git", args...)
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return out
}

// Stub is a stand-in for one tool: the name it answers to, what it answers
// with, and the status it exits with. Shell is the way out for a stub that has
// to do more than answer, run after it has recorded and before it answers.
type Stub struct {
	Name  string
	Says  string
	Exits int
	Shell string
}

// Stubs puts a stand-in for each tool on PATH ahead of whatever the machine
// running the tests has installed, and hands back what they were asked to run:
// one line per invocation, the tool and its arguments, in the order they ran.
// Reaching for a tool is then a line in that log rather than the tool itself.
func Stubs(t *testing.T, stubs ...Stub) func() []string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	for _, s := range stubs {
		// What the stub answers with goes in a file of its own rather than into
		// the script, so that quotes and newlines in it reach the caller as they
		// were written.
		says := ""
		if s.Says != "" {
			answer := filepath.Join(dir, "says-"+s.Name)
			Write(t, answer, s.Says)
			says = fmt.Sprintf("cat %q\n", answer)
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
			// A tool called with no arguments records its name and the empty rest
			// of the line, which is not what it was asked.
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
