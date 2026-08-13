package cli

import (
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// opener is one action a worktree opens on, saying no more than the command line
// reads of one. It spells no flag, so it is named by the flag its own name
// spells.
type opener struct{ name string }

func (o opener) Name() string { return o.name }

func (o opener) Open(worktree.Tree, worktree.Values) (worktree.Handoff, error) {
	return worktree.Handoff{}, nil
}

// spelt is an opener that spells the flag it answers to rather than leaving the
// spelling to its name.
type spelt struct {
	opener
	flag, usage string
}

func (s spelt) Flag() (string, string) { return s.flag, s.usage }

// action is one thing a worktree coming into being means. It spells no flag, so
// nothing declines it.
type action struct{ name string }

func (a action) Name() string            { return a.name }
func (a action) Run(worktree.Tree) error { return nil }

// declined is an action a flag calls off.
type declined struct {
	action
	flag, usage string
}

func (d declined) Flag() (string, string) { return d.flag, d.usage }

// wired stands in for what cmd/work hands the front end, spelling the flags the
// command line records. That the binary's own systems spell these is cmd/work's
// to say; what this package answers for is the tree they make. The claude
// spelling is load-bearing here: it is the one an action was renamed to, which
// is where the flag it used to answer to comes from.
func wired() work.Systems {
	return work.Systems{
		Openers: []work.Opener{
			spelt{opener{"claude"}, "claude", "hand the worktree to claude"},
			opener{"shell"},
		},
		Actions: []work.Action{action{"mise"}, declined{action{"beads"}, "no-claim", "do not claim the ticket"}},
	}
}

// same reports whether two readings of the flags name the same actions.
func same(a, b options) bool {
	return a.open == b.open && slices.Equal(a.skip, b.skip)
}

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
		{"a flag before the identifier", []string{"--shell", "bd-1"}, options{open: "shell"}, "bd-1"},
		{"a flag in place of one", []string{"--shell"}, options{open: "shell"}, ""},
		// A flag declining an action says nothing about which command opens, so it
		// combines with each.
		{"a shell that does not claim", []string{"--shell", "--no-claim", "bd-1"}, options{open: "shell", skip: []string{"beads"}}, "bd-1"},
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
			if !same(got, tt.want) || target != tt.target {
				t.Errorf("Execute(%q) = %+v, %q; want %+v, %q", tt.args, got, target, tt.want, tt.target)
			}
		})
	}
}

// The command line spells the systems it was wired with and holds no list of its
// own, so one wired in cmd/work is reachable by a flag without this package
// knowing it exists. An opener's flag names it; an action's declines it.
func TestTheWiredSystemsNameTheFlags(t *testing.T) {
	sys := work.Systems{
		Openers: []work.Opener{opener{"novel"}, spelt{opener{"terse"}, "spelt-out", "a flag of its own"}},
		Actions: []work.Action{action{"quiet"}, declined{action{"noisy"}, "no-noise", "leave it alone"}},
	}

	sw, _, err := command(stubVersion, sys, front{}).Find([]string{"switch"})
	if err != nil {
		t.Fatalf("finding switch: %v", err)
	}
	for _, name := range []string{"novel", "spelt-out", "no-noise"} {
		if sw.Flags().Lookup(name) == nil {
			t.Errorf("switch does not declare --%s", name)
		}
	}
	// The name an opener spelling its own flag goes by, and the two systems that
	// spelled no flag at all: an action with none runs whenever a worktree does.
	for _, name := range []string{"terse", "quiet", "noisy"} {
		if sw.Flags().Lookup(name) != nil {
			t.Errorf("switch declares --%s, which nothing spelled", name)
		}
	}

	tests := []struct {
		args []string
		want options
	}{
		{[]string{"switch", "--novel", "bd-1"}, options{open: "novel"}},
		{[]string{"switch", "--spelt-out", "bd-1"}, options{open: "terse"}},
		{[]string{"switch", "--no-noise", "bd-1"}, options{skip: []string{"noisy"}}},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			var got options
			err := executeOn(t, sys, tt.args, io.Discard, front{enter: func(o options, _ string) error {
				got = o
				return nil
			}})
			if err != nil {
				t.Fatalf("Execute(%q): %v", tt.args, err)
			}
			if !same(got, tt.want) {
				t.Errorf("Execute(%q) = %+v; want %+v", tt.args, got, tt.want)
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
		{"named before the identifier", []string{"switch", "--shell", "bd-1"}, options{open: "shell"}, "bd-1"},
		{"named after the identifier", []string{"switch", "bd-1", "--shell"}, options{open: "shell"}, "bd-1"},
		{"handed to claude", []string{"switch", "--claude", "bd-1"}, options{open: "claude"}, "bd-1"},
		{"declining the claim", []string{"switch", "--no-claim", "bd-1"}, options{skip: []string{"beads"}}, "bd-1"},
		// Every flag stands without an identifier too, the picker naming one.
		{"an action for the picker's target", []string{"switch", "--claude"}, options{open: "claude"}, ""},
		{"declining the picker's claim", []string{"switch", "--no-claim"}, options{skip: []string{"beads"}}, ""},
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
			if !same(got, tt.want) || target != tt.target {
				t.Errorf("Execute(%q) = %+v, %q; want %+v, %q", tt.args, got, target, tt.want, tt.target)
			}
		})
	}
}

// The bare form is a dispatch, not a second registration: every flag it takes
// is declared once, on the verb it reaches.
func TestOpenOnFlagsAreSwitchs(t *testing.T) {
	root := command(stubVersion, wired(), front{})
	sw, _, err := root.Find([]string{"switch"})
	if err != nil {
		t.Fatalf("finding switch: %v", err)
	}
	for _, name := range []string{"claude", "shell", "no-claim"} {
		if root.Flags().Lookup(name) != nil {
			t.Errorf("--%s is still declared on the root", name)
		}
		if sw.Flags().Lookup(name) == nil {
			t.Errorf("switch does not declare --%s", name)
		}
	}
}

// A flag an action used to answer to is refused with the one it answers to now,
// wherever the set is carried, and is offered by nothing.
func TestTheRenamedFlagIsRefusedByItsNewName(t *testing.T) {
	for _, args := range [][]string{{"bd-1", "--agent"}, {"switch", "--agent", "bd-1"}, {"add", "--agent", "scratch"}} {
		err := execute(t, args, io.Discard, front{})
		if err == nil || !strings.Contains(err.Error(), "--claude") {
			t.Errorf("Execute(%q) = %v; want a refusal naming --claude", args, err)
		}
	}
	sw, _, err := command(stubVersion, wired(), front{}).Find([]string{"switch"})
	if err != nil {
		t.Fatalf("finding switch: %v", err)
	}
	if flag := sw.Flags().Lookup("agent"); flag == nil || !flag.Hidden {
		t.Errorf("--agent is %+v; want a hidden flag", flag)
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
		{"named before the name", []string{"add", "--shell", "scratch"}, options{open: "shell"}},
		{"named after the name", []string{"add", "scratch", "--shell"}, options{open: "shell"}},
		{"handed to claude", []string{"add", "--claude", "scratch"}, options{open: "claude"}},
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
			if !same(got, tt.want) || name != "scratch" {
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
			err := execute(t, tt.args, io.Discard, front{remove: func(f bool, id string) (work.Deletion, error) {
				force, target = f, id
				return work.Deletion{}, nil
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

// refusing is a writer that takes nothing, standing in for a closed pipe.
type refusing struct{}

func (refusing) Write([]byte) (int, error) { return 0, errors.New("refused") }

// What went is printed on the writer cobra handed the command rather than on
// stdout, a line per half, and a refused write fails the removal rather than
// being dropped.
func TestRemovePrints(t *testing.T) {
	both := work.Deletion{Path: "/repo/.worktrees/bd-1", Branch: "bd-1-do-a-thing"}
	tests := []struct {
		name string
		went work.Deletion
		want string
	}{
		{
			"a worktree and its branch",
			both,
			"removed worktree /repo/.worktrees/bd-1\ndeleted branch bd-1-do-a-thing\n",
		},
		{
			"a detached worktree",
			work.Deletion{Path: "/repo/.worktrees/spike"},
			"removed worktree /repo/.worktrees/spike\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			f := front{remove: func(bool, string) (work.Deletion, error) { return tt.went, nil }}
			if err := execute(t, []string{"remove", "scratch"}, &out, f); err != nil {
				t.Fatalf("work remove: %v", err)
			}
			if out.String() != tt.want {
				t.Errorf("work remove printed %q; want %q", out.String(), tt.want)
			}
		})
	}

	f := front{remove: func(bool, string) (work.Deletion, error) { return both, nil }}
	if err := execute(t, []string{"remove", "scratch"}, refusing{}, f); err == nil {
		t.Error("work remove onto a writer that refuses: want the write's error")
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

// edit takes no argument and prints nothing: the terminal goes to the editor,
// so the front end is all there is to observe here.
func TestConfigEdit(t *testing.T) {
	var out strings.Builder
	opened := false
	err := execute(t, []string{"config", "edit"}, &out, front{edit: func() error {
		opened = true
		return nil
	}})
	if err != nil {
		t.Fatalf("Execute(config edit): %v", err)
	}
	if !opened {
		t.Error("Execute(config edit) did not open the settings file")
	}
	if out.String() != "" {
		t.Errorf("Execute(config edit) printed %q; want nothing", out.String())
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
	for _, verb := range []string{"dump", "edit"} {
		if !strings.Contains(out.String(), verb) {
			t.Errorf("Execute(config) printed %q; want the sub-verbs", out.String())
		}
	}
}

// list is a verb of its own, and the only one that prints rather than opening
// something. It prints the branches git reports and asks no listing that would
// name them: the resolved names and their titles are the picker's and the
// completion's.
func TestListPrints(t *testing.T) {
	resolved := func() ([]work.Candidate, error) {
		t.Error("work list asked for the resolved names")
		return nil, nil
	}

	var out strings.Builder
	f := front{
		branches:   func() ([]string, error) { return []string{"bd-1-do-a-thing", "spike"}, nil },
		worktrees:  resolved,
		candidates: resolved,
	}
	if err := execute(t, []string{"list"}, &out, f); err != nil {
		t.Fatalf("work list: %v", err)
	}
	if want := "bd-1-do-a-thing\nspike\n"; out.String() != want {
		t.Errorf("work list printed %q; want %q", out.String(), want)
	}

	// With nothing open there is nothing to print, which is not an error.
	out.Reset()
	if err := execute(t, []string{"list"}, &out, front{branches: func() ([]string, error) { return nil, nil }}); err != nil {
		t.Fatalf("work list with nothing open: %v", err)
	}
	if out.String() != "" {
		t.Errorf("work list with nothing open printed %q; want nothing", out.String())
	}

	// A repository that will not answer is one line on stderr and exit 1.
	silent := front{branches: func() ([]string, error) { return nil, errCancelled }}
	if err := execute(t, []string{"list"}, io.Discard, silent); err == nil {
		t.Error("work list on a silent repository: want an error")
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
		{"--editor is gone", []string{"bd-1", "--editor"}},
		{"--diff is gone", []string{"bd-1", "--diff"}},
		{"--ask is gone", []string{"bd-1", "--ask"}},
		{"claude and a shell at once", []string{"bd-1", "--claude", "--shell"}},
		{"two identifiers", []string{"bd-1", "bd-2"}},
		{"two identifiers on switch", []string{"switch", "bd-1", "bd-2"}},
		{"two actions on switch at once", []string{"switch", "bd-1", "--shell", "--claude"}},
		{"a verb's flag on switch", []string{"switch", "bd-1", "--force"}},
		// Creating is a verb now, and the name it takes is not optional.
		{"--create is gone", []string{"scratch", "--create"}},
		{"add with no name", []string{"add"}},
		{"adding two worktrees at once", []string{"add", "scratch", "other"}},
		// add opens something but has no ticket, so it declines no claim.
		{"declining a claim on add", []string{"add", "scratch", "--no-claim"}},
		{"two actions on add at once", []string{"add", "scratch", "--shell", "--claude"}},
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
		// edit opens the one file of the user's, and carries no flag at all.
		{"config edit with an argument", []string{"config", "edit", "extra"}},
		{"a verb's flag on config edit", []string{"config", "edit", "--shell"}},
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

// A row states what to retype and what kind of thing it is, under the mark the
// resolver that answered for it draws itself with, and the names line up whether
// or not a worktree exists.
func TestLabels(t *testing.T) {
	bead := func(id, title string, open bool) work.Candidate {
		return work.Candidate{Place: worktree.Place{Source: "beads", ID: id, Name: id, Label: title}, Icon: "◆", Open: open}
	}
	pr := func(n, title string, open bool) work.Candidate {
		return work.Candidate{Place: worktree.Place{Source: "github", ID: n, Name: "pr-" + n, Label: title}, Icon: "⇄", Open: open}
	}

	got := labels([]work.Candidate{
		bead("bd-longer", "Other", true),
		bead("bd-1", "Do a thing", false),
		pr("7", "Review this", true),
		// A plain worktree is named by its branch and says nothing more.
		{Place: worktree.Place{Source: "plain", ID: "/elsewhere", Name: "spike"}, Icon: "◇", Open: true},
		bead("bd-untitled-and-longest", "", false), // bd could not say
		pr("9", "", false), // nor could gh
		// A resolver that named no mark leaves the column empty rather than the row
		// short, so the names still line up under one another.
		{Place: worktree.Place{Source: "undrawn", ID: "x", Name: "elsewhere"}},
	})
	want := []string{
		highlight + "⎇ ◆ bd-longer" + reset + "  ·  Other",
		"  ◆ bd-1       ·  Do a thing",
		highlight + "⎇ ⇄ pr-7     " + reset + "  ·  Review this",
		highlight + "⎇ ◇ spike" + reset,
		"  ◆ bd-untitled-and-longest",
		"  ⇄ pr-9",
		"    elsewhere",
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
// over the systems the command line records.
func execute(t *testing.T, args []string, out io.Writer, f front) error {
	t.Helper()
	return executeOn(t, wired(), args, out, f)
}

// executeOn is the same over a wiring of the case's own, standing in for
// whatever it left unnamed: a verb it did not expect to run fails the test
// rather than passing silently.
func executeOn(t *testing.T, sys work.Systems, args []string, out io.Writer, f front) error {
	t.Helper()
	if f.enter == nil {
		f.enter = func(options, string) error { t.Error("entered a worktree"); return nil }
	}
	if f.add == nil {
		f.add = func(options, string) error { t.Error("added a worktree"); return nil }
	}
	if f.remove == nil {
		f.remove = func(bool, string) (work.Deletion, error) {
			t.Error("removed a worktree")
			return work.Deletion{}, nil
		}
	}
	if f.dump == nil {
		f.dump = func(io.Writer) error { t.Error("dumped the configuration"); return nil }
	}
	if f.edit == nil {
		f.edit = func() error { t.Error("opened the settings file"); return nil }
	}
	if f.candidates == nil {
		f.candidates = stub(nil, nil)
	}
	if f.worktrees == nil {
		f.worktrees = stub(nil, nil)
	}
	if f.branches == nil {
		f.branches = func() ([]string, error) { t.Error("listed the branches"); return nil, nil }
	}
	cmd := command(stubVersion, sys, f)
	cmd.SetOut(out)
	cmd.SetErr(io.Discard)
	return run(cmd, args)
}

// Every verb that reaches the core reaches it through the wiring the call
// carries, so two wirings in one process each answer for their own systems.
// Parked in a package-level var, whichever was assigned last would answer for
// both, and which systems a verb reads would depend on what set the var up.
func TestEachCallCarriesItsOwnWiring(t *testing.T) {
	t.Chdir(testenv.InitRepo(t))

	var asked []string
	wire := func(name string) work.Wiring {
		return func(string, string, config.Config) work.Systems {
			asked = append(asked, name)
			return work.Systems{}
		}
	}
	if _, err := (verbs{wire: wire("first")}).listing(); err != nil {
		t.Fatalf("listing: %v", err)
	}
	if _, err := (verbs{wire: wire("second")}).worktreeListing(); err != nil {
		t.Fatalf("worktreeListing: %v", err)
	}
	if _, err := (verbs{wire: wire("third")}).branchListing(); err != nil {
		t.Fatalf("branchListing: %v", err)
	}
	if want := []string{"first", "second", "third"}; !slices.Equal(asked, want) {
		t.Errorf("the wirings asked were %q; want %q", asked, want)
	}
}
