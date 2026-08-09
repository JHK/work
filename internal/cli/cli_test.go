package cli

import (
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/work"
)

// The bare form arrives at switch as it was typed, whatever stands in the first
// position: a name, a flag, or nothing. Which flag names which action is
// TestSwitchFlags' to say, the two forms reaching the one command.
func TestCommandFlags(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		want   options
		target string
	}{
		{"nothing at all", nil, options{}, ""},
		{"an identifier alone", []string{"bd-1"}, options{}, "bd-1"},
		{"a flag before the identifier", []string{"--shell", "bd-1"}, options{shell: true}, "bd-1"},
		{"a flag in place of one", []string{"--shell"}, options{shell: true}, ""},
		// --no-claim says nothing about which command opens, so it combines with each.
		{"a shell that does not claim", []string{"--shell", "--no-claim", "bd-1"}, options{shell: true, noClaim: true}, "bd-1"},
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.action(); got != tt.want {
				t.Errorf("%+v names %s; want %s", tt.opts, got, tt.want)
			}
		})
	}
}

// switch is the bare form spelled out, so it takes the same argument and
// carries the same flags, and the verb reaches the names the bare form cannot.
func TestSwitchFlags(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		want   options
		target string
	}{
		{"an identifier of its own", []string{"switch", "bd-1"}, options{}, "bd-1"},
		{"no identifier", []string{"switch"}, options{}, ""},
		{"named before the identifier", []string{"switch", "--shell", "bd-1"}, options{shell: true}, "bd-1"},
		{"named after the identifier", []string{"switch", "bd-1", "--shell"}, options{shell: true}, "bd-1"},
		{"in the editor", []string{"switch", "--editor", "bd-1"}, options{editor: true}, "bd-1"},
		{"diffed", []string{"switch", "--diff", "bd-1"}, options{diff: true}, "bd-1"},
		{"handed to the agent", []string{"switch", "--agent", "bd-1"}, options{agent: true}, "bd-1"},
		{"asking what to open on", []string{"switch", "--ask", "bd-1"}, options{ask: true}, "bd-1"},
		{"declining the claim", []string{"switch", "--no-claim", "bd-1"}, options{noClaim: true}, "bd-1"},
		// Every flag stands without an identifier too, the picker naming one.
		{"an action for the picker's target", []string{"switch", "--editor"}, options{editor: true}, ""},
		{"declining the picker's claim", []string{"switch", "--no-claim"}, options{noClaim: true}, ""},
		// The verbs shadow these names in the bare position; switch reaches them.
		{"a worktree named init", []string{"switch", "init"}, options{}, "init"},
		{"a worktree named add", []string{"switch", "add"}, options{}, "add"},
		{"a worktree named remove", []string{"switch", "remove"}, options{}, "remove"},
		{"a worktree named list", []string{"switch", "list"}, options{}, "list"},
		{"a worktree named switch", []string{"switch", "switch"}, options{}, "switch"},
		{"a worktree named config", []string{"switch", "config"}, options{}, "config"},
		{"a worktree named help", []string{"switch", "help"}, options{}, "help"},
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

// The bare form is a dispatch, not a second registration: every flag it takes
// is declared once, on the verb it reaches.
func TestOpenOnFlagsAreSwitchs(t *testing.T) {
	root := command(stubVersion, front{})
	sw, _, err := root.Find([]string{"switch"})
	if err != nil {
		t.Fatalf("finding switch: %v", err)
	}
	for _, name := range []string{"agent", "shell", "editor", "diff", "ask", "no-claim"} {
		if root.Flags().Lookup(name) != nil {
			t.Errorf("--%s is still declared on the root", name)
		}
		if sw.Flags().Lookup(name) == nil {
			t.Errorf("switch does not declare --%s", name)
		}
	}
}

// add is a verb of its own, so it takes the name in the argument position and
// carries the open-on flags wherever those sit.
func TestAddFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want options
	}{
		{"a name of its own", []string{"add", "scratch"}, options{}},
		{"named before the name", []string{"add", "--shell", "scratch"}, options{shell: true}},
		{"named after the name", []string{"add", "scratch", "--shell"}, options{shell: true}},
		{"in the editor", []string{"add", "--editor", "scratch"}, options{editor: true}},
		{"diffed", []string{"add", "--diff", "scratch"}, options{diff: true}},
		{"handed to the agent", []string{"add", "--agent", "scratch"}, options{agent: true}},
		{"asking what to open on", []string{"add", "--ask", "scratch"}, options{ask: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got options
			var name string
			err := execute(t, tt.args, io.Discard, front{add: func(o options, n string) error {
				got, name = o, n
				return nil
			}})
			if err != nil {
				t.Fatalf("Execute(%q): %v", tt.args, err)
			}
			if got != tt.want || name != "scratch" {
				t.Errorf("Execute(%q) = %+v, %q; want %+v, %q", tt.args, got, name, tt.want, "scratch")
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

// config runs nothing of its own: dump is the sub-verb, and what it prints goes
// to stdout rather than to the front end.
func TestConfigDump(t *testing.T) {
	var out strings.Builder
	err := execute(t, []string{"config", "dump"}, &out, front{dump: func(w io.Writer) error {
		_, err := io.WriteString(w, dumped)
		return err
	}})
	if err != nil {
		t.Fatalf("Execute(config dump): %v", err)
	}
	if out.String() != dumped {
		t.Errorf("Execute(config dump) printed %q; want %q", out.String(), dumped)
	}
}

// The verb with no sub-verb says what it carries instead of dumping, a dump
// being one thing config could go on to do rather than the thing it is.
func TestConfigWithoutASubVerb(t *testing.T) {
	var out strings.Builder
	err := execute(t, []string{"config"}, &out, front{})
	if err != nil {
		t.Fatalf("Execute(config): %v", err)
	}
	if !strings.Contains(out.String(), "dump") {
		t.Errorf("Execute(config) printed %q; want the sub-verbs", out.String())
	}
}

// list is a verb of its own, and the only one that prints rather than opening
// something. It prints the worktrees remove's listing offers, no more.
func TestListPrints(t *testing.T) {
	worktrees := []work.Candidate{
		{Target: work.Target{Kind: work.KindBead, ID: "bd-1", Name: "bd-1"}, Label: "Do a thing", Open: true},
		{Target: work.Target{Kind: work.KindPlain, ID: "/elsewhere", Name: "spike"}, Open: true},
	}
	elsewhere := []work.Candidate{{Target: work.Target{Kind: work.KindPR, ID: "7", Name: "pr-7"}, Label: "Review this"}}

	var out strings.Builder
	err := execute(t, []string{"list"}, &out, front{worktrees: stub(worktrees, nil), candidates: stub(elsewhere, nil)})
	if err != nil {
		t.Fatalf("work list: %v", err)
	}
	// spike carries no title, so it neither pads nor widens the column.
	want := "bd-1  Do a thing\nspike\n"
	if out.String() != want {
		t.Errorf("work list printed %q; want %q", out.String(), want)
	}

	// A repository that will not answer is one line on stderr and exit 1.
	if err := execute(t, []string{"list"}, io.Discard, front{worktrees: stub(nil, errCancelled)}); err == nil {
		t.Error("work list on a silent repository: want an error")
	}
}

// A row states what to retype and, where an adapter named one, the title, lined
// up behind the widest name that has one.
func TestLines(t *testing.T) {
	got := lines([]work.Candidate{
		{Target: work.Target{Kind: work.KindBead, ID: "bd-longer", Name: "bd-longer"}, Label: "Other", Open: true},
		{Target: work.Target{Kind: work.KindPR, ID: "7", Name: "pr-7"}, Label: "Review this", Open: true},
		// A plain worktree is named by its branch and says nothing more.
		{Target: work.Target{Kind: work.KindPlain, ID: "/elsewhere", Name: "spike"}, Open: true},
		// An untitled name is not padded, so it does not set the column either.
		{Target: work.Target{Kind: work.KindBead, ID: "bd-untitled-and-longest", Name: "bd-untitled-and-longest"}, Open: true},
	})
	want := []string{
		"bd-longer  Other",
		"pr-7       Review this",
		"spike",
		"bd-untitled-and-longest",
	}
	if !slices.Equal(got, want) {
		t.Errorf("lines() = %q; want %q", got, want)
	}
}

// With nothing open there is nothing to print, which is not an error.
func TestLinesOfNothing(t *testing.T) {
	if got := lines(nil); len(got) != 0 {
		t.Errorf("lines(nil) = %q; want nothing", got)
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
		{"two identifiers on switch", []string{"switch", "bd-1", "bd-2"}},
		{"two actions on switch at once", []string{"switch", "bd-1", "--shell", "--editor"}},
		{"a verb's flag on switch", []string{"switch", "bd-1", "--force"}},
		// Creating is a verb now, and the name it takes is not optional.
		{"--create is gone", []string{"scratch", "--create"}},
		{"add with no name", []string{"add"}},
		{"adding two worktrees at once", []string{"add", "scratch", "other"}},
		// add opens something but has no ticket, so it declines no claim.
		{"declining a claim on add", []string{"add", "scratch", "--no-claim"}},
		{"two actions on add at once", []string{"add", "scratch", "--shell", "--editor"}},
		// Removing is a verb now, and --force went with it.
		{"--delete is gone", []string{"scratch", "--delete"}},
		{"--force is gone from the root", []string{"scratch", "--force"}},
		// remove declares --force and nothing else, so a root flag is unknown to it.
		{"a root flag on remove", []string{"remove", "scratch", "--shell"}},
		{"removing two worktrees at once", []string{"remove", "scratch", "other"}},
		// Listing takes no argument: a name to filter on is switch's.
		{"list with a name", []string{"list", "scratch"}},
		{"a root flag on list", []string{"list", "--shell"}},
		{"unknown flag", []string{"bd-1", "--turbo"}},
		// config carries sub-verbs, and dump takes nothing.
		{"config with a sub-verb it does not carry", []string{"config", "bogus"}},
		{"config dump with an argument", []string{"config", "dump", "extra"}},
		{"a verb's flag on config dump", []string{"config", "dump", "--force"}},
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

const (
	stubVersion = "v0.0.0-test"
	dumped      = "the configuration\n"
)

// execute puts args through the tree the way [Execute] does, dispatch included,
// standing in for whatever the case left unnamed: a verb it did not expect to
// run fails the test rather than passing silently.
func execute(t *testing.T, args []string, out io.Writer, f front) error {
	t.Helper()
	if f.enter == nil {
		f.enter = func(options, string) error { t.Error("entered a worktree"); return nil }
	}
	if f.add == nil {
		f.add = func(options, string) error { t.Error("added a worktree"); return nil }
	}
	if f.remove == nil {
		f.remove = func(bool, string) error { t.Error("removed a worktree"); return nil }
	}
	if f.dump == nil {
		f.dump = func(io.Writer) error { t.Error("dumped the configuration"); return nil }
	}
	if f.candidates == nil {
		f.candidates = stub(nil, nil)
	}
	if f.worktrees == nil {
		f.worktrees = stub(nil, nil)
	}
	cmd := command(stubVersion, f)
	cmd.SetOut(out)
	cmd.SetErr(io.Discard)
	return run(cmd, args)
}
