package cli

import (
	"io"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/work"
)

func TestCommandFlags(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		want   options
		target string
	}{
		{"no identifier", []string{"--shell"}, options{shell: true}, ""},
		{"flags before the identifier", []string{"--shell", "bd-1"}, options{shell: true}, "bd-1"},
		{"an editor with no identifier", []string{"--editor"}, options{editor: true}, ""},
		{"an editor on a named target", []string{"--editor", "bd-1"}, options{editor: true}, "bd-1"},
		// --no-claim says nothing about which command opens, so it combines with each.
		{"no claim", []string{"--no-claim", "bd-1"}, options{noClaim: true}, "bd-1"},
		{"a shell that does not claim", []string{"--shell", "--no-claim", "bd-1"}, options{shell: true, noClaim: true}, "bd-1"},
		{"an editor that does not claim", []string{"--editor", "--no-claim", "bd-1"}, options{editor: true, noClaim: true}, "bd-1"},
		{"an agent on a named target", []string{"--agent", "bd-1"}, options{agent: true}, "bd-1"},
		{"an agent with no identifier", []string{"--agent"}, options{agent: true}, ""},
		{"a diff on a named target", []string{"--diff", "bd-1"}, options{diff: true}, "bd-1"},
		{"a diff that does not claim", []string{"--diff", "--no-claim", "bd-1"}, options{diff: true, noClaim: true}, "bd-1"},
		{"no claim from the picker", []string{"--no-claim"}, options{noClaim: true}, ""},
		{"ask on a named target", []string{"--ask", "bd-1"}, options{ask: true}, "bd-1"},
		{"ask with no identifier", []string{"--ask"}, options{ask: true}, ""},
		{"ask without claiming", []string{"--ask", "--no-claim", "bd-1"}, options{ask: true, noClaim: true}, "bd-1"},
		// --create says nothing about which command opens either, so it combines too.
		{"create", []string{"--create", "scratch"}, options{create: true}, "scratch"},
		{"a created worktree in a shell", []string{"--create", "--shell", "scratch"}, options{create: true, shell: true}, "scratch"},
		{"a created worktree in the editor", []string{"--create", "--editor", "scratch"}, options{create: true, editor: true}, "scratch"},
		{"a created worktree diffed", []string{"--create", "--diff", "scratch"}, options{create: true, diff: true}, "scratch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got options
			var target string
			err := execute(t, tt.args, io.Discard, front{enter: func(o options, id string) error {
				got, target = o, id
				return nil
			}})
			if err != nil {
				t.Fatalf("Execute(%q): %v", tt.args, err)
			}
			if got != tt.want || target != tt.target {
				t.Errorf("Execute(%q) = %+v, %q; want %+v, %q", tt.args, got, target, tt.want, tt.target)
			}
		})
	}
}

// The flags are five, the action they name is one. Which flag sets which field
// is TestCommandFlags' to say; two at once never reach here, cobra having
// refused them.
func TestFlagsNameOneAction(t *testing.T) {
	tests := []struct {
		name string
		opts options
		want work.Action
	}{
		{"nothing named", options{}, work.ActionUnnamed},
		{"agent", options{agent: true}, work.ActionAgent},
		{"shell", options{shell: true}, work.ActionShell},
		{"editor", options{editor: true}, work.ActionEditor},
		{"diff", options{diff: true}, work.ActionDiff},
		{"ask", options{ask: true}, work.ActionAsk},
		{"no claim names none", options{noClaim: true}, work.ActionUnnamed},
		{"create names none", options{create: true}, work.ActionUnnamed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.action(); got != tt.want {
				t.Errorf("%+v names %s; want %s", tt.opts, got, tt.want)
			}
		})
	}
}

// remove is a verb of its own, so it takes the name in the argument position
// and carries --force wherever that sits.
func TestRemoveFlags(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		force  bool
		target string
	}{
		{"a named worktree", []string{"remove", "scratch"}, false, "scratch"},
		{"forced before the name", []string{"remove", "--force", "scratch"}, true, "scratch"},
		{"forced after the name", []string{"remove", "scratch", "--force"}, true, "scratch"},
		// The picker stands in for the name, over the worktrees alone.
		{"no name", []string{"remove"}, false, ""},
		{"forced with no name", []string{"remove", "--force"}, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var force bool
			var target string
			err := execute(t, tt.args, io.Discard, front{remove: func(f bool, id string) error {
				force, target = f, id
				return nil
			}})
			if err != nil {
				t.Fatalf("Execute(%q): %v", tt.args, err)
			}
			if force != tt.force || target != tt.target {
				t.Errorf("Execute(%q) = %v, %q; want %v, %q", tt.args, force, target, tt.force, tt.target)
			}
		})
	}
}

func TestVersionFlag(t *testing.T) {
	var out strings.Builder
	err := execute(t, []string{"--version"}, &out, front{})
	if err != nil {
		t.Fatalf("Execute(--version): %v", err)
	}
	if !strings.Contains(out.String(), stubVersion) {
		t.Errorf("Execute(--version) printed %q; want the version %q", out.String(), stubVersion)
	}
}

func TestCommandRejects(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"--start is gone", []string{"bd-1", "--start"}},
		{"--model is gone", []string{"bd-1", "--model", "opus"}},
		{"--effort is gone", []string{"--effort=high", "bd-1"}},
		{"a shell and an editor at once", []string{"bd-1", "--shell", "--editor"}},
		{"a shell and a diff at once", []string{"bd-1", "--shell", "--diff"}},
		{"an editor and a diff at once", []string{"bd-1", "--editor", "--diff"}},
		{"an agent and a shell at once", []string{"bd-1", "--agent", "--shell"}},
		{"an agent and an editor at once", []string{"bd-1", "--agent", "--editor"}},
		{"an agent and a diff at once", []string{"bd-1", "--agent", "--diff"}},
		{"asking and naming an action at once", []string{"bd-1", "--ask", "--shell"}},
		{"asking and an agent at once", []string{"bd-1", "--ask", "--agent"}},
		{"two identifiers", []string{"bd-1", "bd-2"}},
		// A worktree with no ticket behind it has no claim to decline.
		{"creating and declining a claim at once", []string{"scratch", "--create", "--no-claim"}},
		// Removing is a verb now, and --force went with it.
		{"--delete is gone", []string{"scratch", "--delete"}},
		{"--force is gone from the root", []string{"scratch", "--force"}},
		// remove declares --force and nothing else, so a root flag is unknown to it.
		{"a root flag on remove", []string{"remove", "scratch", "--shell"}},
		{"removing two worktrees at once", []string{"remove", "scratch", "other"}},
		{"unknown flag", []string{"bd-1", "--turbo"}},
		{"init without a shell", []string{"init"}},
		{"init with a shell work does not print", []string{"init", "bash"}},
		{"init with two shells", []string{"init", "fish", "zsh"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := execute(t, tt.args, io.Discard, front{})
			if err == nil {
				t.Errorf("Execute(%q): want an error", tt.args)
			}
		})
	}
}

// A row states what to retype and what kind of thing it is, and the names line
// up whether or not a worktree exists.
func TestLabels(t *testing.T) {
	bead := func(id, title string, open bool) work.Candidate {
		return work.Candidate{Target: work.Target{Kind: work.KindBead, ID: id, Name: id}, Label: title, Open: open}
	}
	pr := func(n, title string, open bool) work.Candidate {
		return work.Candidate{Target: work.Target{Kind: work.KindPR, ID: n, Name: "pr-" + n}, Label: title, Open: open}
	}

	got := labels([]work.Candidate{
		bead("bd-longer", "Other", true),
		bead("bd-1", "Do a thing", false),
		pr("7", "Review this", true),
		// A plain worktree is named by its branch and says nothing more.
		{Target: work.Target{Kind: work.KindPlain, ID: "/elsewhere", Name: "spike"}, Open: true},
		bead("bd-untitled-and-longest", "", false), // bd could not say
		pr("9", "", false), // nor could gh
	})
	want := []string{
		highlight + "⎇ ◆ bd-longer" + reset + "  ·  Other",
		"  ◆ bd-1       ·  Do a thing",
		highlight + "⎇ ⇄ pr-7     " + reset + "  ·  Review this",
		highlight + "⎇ ◇ spike" + reset,
		"  ◆ bd-untitled-and-longest",
		"  ⇄ pr-9",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d rows; want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %q; want %q", i, got[i], want[i])
		}
	}
}

// The picker offers what exists, so it never stands in for a name --create was
// not given. The env here is not a repository, so anything else would fail
// against git rather than refuse the invocation.
func TestCreateNeedsAName(t *testing.T) {
	_, err := candidate(work.Env{}, options{create: true}, "")
	if err == nil || !strings.Contains(err.Error(), "--create needs a name") {
		t.Errorf("--create with no name = %v; want it refused by name", err)
	}
}

const stubVersion = "v0.0.0-test"

// execute puts args through the tree, standing in for whatever the case left
// unnamed: a verb it did not expect to run fails the test rather than passing
// silently.
func execute(t *testing.T, args []string, out io.Writer, f front) error {
	t.Helper()
	if f.enter == nil {
		f.enter = func(options, string) error { t.Error("entered a worktree"); return nil }
	}
	if f.remove == nil {
		f.remove = func(bool, string) error { t.Error("removed a worktree"); return nil }
	}
	if f.candidates == nil {
		f.candidates = stub(nil, nil)
	}
	if f.worktrees == nil {
		f.worktrees = stub(nil, nil)
	}
	cmd := command(stubVersion, f)
	cmd.SetArgs(args)
	cmd.SetOut(out)
	cmd.SetErr(io.Discard)
	return cmd.Execute()
}
