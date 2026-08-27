package testenv

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/run"
)

// Tools are the stand-ins for everything work reaches beyond git, each failing
// where it is reached at all, so a question put to one is recorded rather than
// answered.
func Tools() []Stub {
	return []Stub{
		{Name: "bd", Exits: 1},
		{Name: "gh", Exits: 1},
		{Name: "mise", Exits: 1},
		{Name: "claude", Exits: 1},
		{Name: "fzf", Exits: 1},
	}
}

// Stub is a stand-in for one tool: the name it answers to, and either the one
// answer it gives whatever it is asked or the [Reply] list it answers each
// question with. Says, Grumbles and Exits are that one answer, on stdout, on
// stderr and in the exit status; Shell is the way out for a stub that has to do
// more than answer, run after it has recorded and before it answers, and a
// command in it that fails takes the stand-in down with it. A stub that names
// replies refuses a question none of them matches, saying what it was asked.
type Stub struct {
	Name     string
	Says     string
	Grumbles string
	Exits    int
	Shell    string
	Replies  []Reply
}

// Reply is one answer a stand-in has ready for a question carrying every word in
// To, said the way a [Stub] says its own. The first reply naming a question
// answers it, however often it is asked.
type Reply struct {
	To       []string
	Says     string
	Grumbles string
	Exits    int
	Shell    string
}

// Stubs puts a stand-in for each tool on PATH ahead of whatever the machine has
// installed, and hands back what they were asked to run: one line per
// invocation, the tool and its arguments, in the order they ran. Where two name
// one tool the last stands, so a set may be handed on with an answer for one of
// them appended. A stand-in is this test binary under the tool's name, so only a
// reply that names a Shell needs anything else on PATH.
func Stubs(t *testing.T, stubs ...Stub) func() []string {
	t.Helper()
	// A tool an earlier case found missing is out for the rest of the process, which
	// would leave the stand-ins put here unasked.
	run.Forget()
	t.Cleanup(run.Forget)
	binary, err := os.Executable()
	if err != nil {
		t.Fatalf("find the test binary: %v", err)
	}
	dir := t.TempDir()
	for _, s := range stubs {
		at := filepath.Join(dir, s.Name)
		// Where two stubs name one tool the last stands, on the stand-in already there.
		if err := os.Symlink(binary, at); err != nil && !os.IsExist(err) {
			t.Fatalf("stand in for %s: %v", s.Name, err)
		}
		Write(t, at+ready, promised(t, s))
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	return func() []string {
		out, err := os.ReadFile(filepath.Join(dir, asked))
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("read %s: %v", asked, err)
		}
		var ran []string
		for line := range strings.SplitSeq(string(out), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				ran = append(ran, line)
			}
		}
		return ran
	}
}

const (
	ready = ".replies" // what a stand-in's replies sit beside its name under
	asked = "log"      // where the stand-ins of one directory record their questions
)

// promised is a stub's replies as its stand-in reads them back, the stub that
// answers the same way whatever it is asked being the one reply naming no
// question.
func promised(t *testing.T, s Stub) string {
	t.Helper()
	replies := s.Replies
	if len(replies) == 0 {
		replies = []Reply{{Says: s.Says, Grumbles: s.Grumbles, Exits: s.Exits, Shell: s.Shell}}
	} else if s.Says != "" || s.Grumbles != "" || s.Exits != 0 || s.Shell != "" {
		t.Fatalf("stub %s answers both the same way and by question; want that one answer put in a reply of its own", s.Name)
	}
	body, _ := json.Marshal(replies)
	return string(body)
}

// standIn answers as the tool this binary was reached under and never comes
// back, where a [Stubs] call left replies beside that name. Anything else is a
// process that goes on to run tests.
func standIn() {
	at, err := exec.LookPath(os.Args[0])
	if err != nil {
		return
	}
	body, err := os.ReadFile(at + ready)
	if err != nil {
		return
	}
	var replies []Reply
	must(json.Unmarshal(body, &replies))
	dir, name, args := filepath.Dir(at), filepath.Base(at), os.Args[1:]
	log, err := os.OpenFile(filepath.Join(dir, asked), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	must(err)
	_, err = fmt.Fprintln(log, run.CommandLine(name, args...))
	must(err)
	must(log.Close())
	for _, r := range replies {
		if r.names(args) {
			os.Exit(r.say(name, args))
		}
	}
	// Exit 2, which none of the tools work reaches means as an answer of its own.
	fmt.Fprintf(os.Stderr, "%s: no reply to: %s\n", name, strings.Join(args, " "))
	os.Exit(2)
}

// names reports whether the question carries every word the reply names.
func (r Reply) names(args []string) bool {
	return !slices.ContainsFunc(r.To, func(word string) bool { return !slices.Contains(args, word) })
}

// say runs what the reply has to run, then answers on each stream and comes back
// with the status it exits under. The command is given the question in its
// positional arguments, and one that fails takes the stand-in down with it.
func (r Reply) say(name string, args []string) int {
	if r.Shell != "" {
		cmd := exec.Command("sh", append([]string{"-c", r.Shell, name}, args...)...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			if cmd.ProcessState == nil {
				must(err)
			}
			return cmd.ProcessState.ExitCode()
		}
	}
	_, _ = io.WriteString(os.Stdout, r.Says)
	_, _ = io.WriteString(os.Stderr, r.Grumbles)
	return r.Exits
}

// must takes a stand-in down where the ground under it failed.
func must(err error) {
	if err != nil {
		panic(err)
	}
}
