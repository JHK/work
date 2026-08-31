package testenv_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// answer is what one question put to a stand-in came back with.
type answer struct {
	Code int
	Out  string
	Err  string
}

// asks puts one question to the stand-in on PATH, standing where the test says,
// and hands back what came back on each stream and in the exit status.
func asks(t *testing.T, dir, name string, args ...string) answer {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var out, said strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &said
	code := 0
	if err := cmd.Run(); err != nil {
		exit := new(exec.ExitError)
		require.ErrorAs(t, err, &exit, "%s %s", name, strings.Join(args, " "))
		code = exit.ExitCode()
	}
	return answer{Out: out.String(), Err: said.String(), Code: code}
}

// Each reply is taken for the question carrying the words it names, and nothing
// else.
func TestAStandInAnswersByWhatItWasAsked(t *testing.T) {
	testenv.Stubs(t, testenv.Stub{Name: "bd", Replies: []testenv.Reply{
		{To: []string{"list"}, Says: "[]\n"},
		{To: []string{"update", "--claim"}},
		{To: []string{"show"}, Grumbles: "bead not found\n", Exits: 1},
	}})
	here := t.TempDir()

	for _, tt := range []struct {
		name string
		args []string
		want answer
	}{
		{"a listing", []string{"list", "--all", "--json"}, answer{Out: "[]\n"}},
		{"a claim", []string{"update", "one", "--claim"}, answer{}},
		{"a bead nothing knows", []string{"show", "one"}, answer{Err: "bead not found\n", Code: 1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			testenv.Equal(t, tt.want, asks(t, here, "bd", tt.args...), "the stand-in answered the wrong reply")
		})
	}
}

// A question no reply matches is refused, naming the stand-in and what it was
// asked, rather than answered with silence a case would read as success.
func TestAQuestionNoReplyMatchesIsRefusedNamingIt(t *testing.T) {
	ran := testenv.Stubs(t, testenv.Stub{Name: "bd", Replies: []testenv.Reply{
		{To: []string{"list"}, Says: "[]\n"},
	}})

	got := asks(t, t.TempDir(), "bd", "worktree", "create", "one")

	require.NotZero(t, got.Code, "a question nothing answers came back a success")
	require.Contains(t, got.Err, "bd", "the refusal does not name the stand-in")
	require.Contains(t, got.Err, "worktree create one", "the refusal does not name what was asked")
	require.Empty(t, got.Out, "a refused question answered on stdout")
	testenv.Equal(t, []string{"bd worktree create one"}, ran(), "a refused question went unrecorded")
}

// A reply runs a command where the stand-in stands, so a create step lands a
// worktree on disk without a dispatch script written by hand.
func TestAReplyRunsACommandWhereItStands(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Stubs(t, testenv.Stub{Name: "bd", Replies: []testenv.Reply{
		{To: []string{"worktree", "create"}, Shell: `git worktree add "$3" -b "$5"`},
	}})
	path := filepath.Join(repo, "trees", "one")

	got := asks(t, repo, "bd", "worktree", "create", path, "--branch", "one")

	require.Zero(t, got.Code, "creating the worktree failed: %s", got.Err)
	require.DirExists(t, path, "the reply ran but no worktree landed")
	require.Equal(t, "one", testenv.Git(t, path, "rev-parse", "--abbrev-ref", "HEAD"), "the worktree landed on the branch the reply named")
}

// A command a reply runs that fails takes the stand-in down with it, rather than
// answering over a step that never happened.
func TestAStandInFailsWithTheCommandItsReplyRan(t *testing.T) {
	testenv.Stubs(t, testenv.Stub{Name: "bd", Replies: []testenv.Reply{
		{To: []string{"worktree"}, Shell: "false", Says: "created\n"},
	}})

	got := asks(t, t.TempDir(), "bd", "worktree", "create")

	require.NotZero(t, got.Code, "the stand-in answered for a command that failed")
	require.Empty(t, got.Out, "the stand-in answered over a step that never happened")
}

// A stand-in that names no replies answers everything the same way, which is
// every stub that stands only to keep a tool off the machine's PATH.
func TestAStandInWithNoRepliesAnswersEverythingTheSameWay(t *testing.T) {
	testenv.Stubs(t, testenv.Stub{Name: "gh", Says: "{}\n", Exits: 3})
	here := t.TempDir()

	for _, args := range [][]string{{"pr", "view", "7"}, {"auth", "status"}} {
		testenv.Equal(t, answer{Out: "{}\n", Code: 3}, asks(t, here, "gh", args...), "gh "+strings.Join(args, " ")+" was answered differently")
	}
}
