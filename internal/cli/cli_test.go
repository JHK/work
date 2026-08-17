package cli

import (
	"errors"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/git"
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

// openingVerbs hand a worktree to something; creatingVerbs are the ones among
// them that can bring one into being.
var (
	openingVerbs  = []string{"go", "switch", "add"}
	creatingVerbs = []string{"go", "add"}
)

// The bare form arrives at go as it was typed, whatever stands in the first
// position: a name, a flag, or nothing. Which flag names which action is
// TestGoFlags' to say, the two forms reaching the one command.
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
			err := execute(t, tt.args, io.Discard, front{reach: runs(func(o options, id string) (worktree.Handoff, error) {
				got, target = o, id
				return worktree.Handoff{}, nil
			})})
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

	root := command(stubVersion, sys, front{})
	for _, verb := range openingVerbs {
		cmd, _, err := root.Find([]string{verb})
		if err != nil {
			t.Fatalf("finding %s: %v", verb, err)
		}
		for _, name := range []string{"novel", "spelt-out"} {
			if cmd.Flags().Lookup(name) == nil {
				t.Errorf("%s does not declare --%s", verb, name)
			}
		}
		// The flag that calls an action off is declared by the verbs that can run one.
		if declares, creates := cmd.Flags().Lookup("no-noise") != nil, slices.Contains(creatingVerbs, verb); declares != creates {
			t.Errorf("%s declares --no-noise: %v; want %v", verb, declares, creates)
		}
		// The name an opener spelling its own flag goes by, and the two systems that
		// spelled no flag at all: an action with none runs whenever a worktree does.
		for _, name := range []string{"terse", "quiet", "noisy"} {
			if cmd.Flags().Lookup(name) != nil {
				t.Errorf("%s declares --%s, which nothing spelled", verb, name)
			}
		}
	}

	tests := []struct {
		args []string
		want options
	}{
		{[]string{"go", "--novel", "bd-1"}, options{open: "novel"}},
		{[]string{"go", "--spelt-out", "bd-1"}, options{open: "terse"}},
		{[]string{"go", "--no-noise", "bd-1"}, options{skip: []string{"noisy"}}},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			var got options
			err := executeOn(t, sys, tt.args, io.Discard, front{reach: runs(func(o options, _ string) (worktree.Handoff, error) {
				got = o
				return worktree.Handoff{}, nil
			})})
			if err != nil {
				t.Fatalf("Execute(%q): %v", tt.args, err)
			}
			if !same(got, tt.want) {
				t.Errorf("Execute(%q) = %+v; want %+v", tt.args, got, tt.want)
			}
		})
	}
}

// go is the bare form spelled out, so it takes the same argument and carries the
// same flags, and the verb reaches the names the bare form cannot.
func TestGoFlags(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		want   options
		target string
	}{
		{"an identifier of its own", []string{"go", "bd-1"}, options{}, "bd-1"},
		{"no identifier", []string{"go"}, options{}, ""},
		{"named before the identifier", []string{"go", "--shell", "bd-1"}, options{open: "shell"}, "bd-1"},
		{"named after the identifier", []string{"go", "bd-1", "--shell"}, options{open: "shell"}, "bd-1"},
		{"handed to claude", []string{"go", "--claude", "bd-1"}, options{open: "claude"}, "bd-1"},
		{"declining the claim", []string{"go", "--no-claim", "bd-1"}, options{skip: []string{"beads"}}, "bd-1"},
		// Every flag stands without an identifier too, the picker naming one.
		{"an action for the picker's target", []string{"go", "--claude"}, options{open: "claude"}, ""},
		{"declining the picker's claim", []string{"go", "--no-claim"}, options{skip: []string{"beads"}}, ""},
		// The verbs shadow these names in the bare position; go reaches them.
		{"a worktree named init", []string{"go", "init"}, options{}, "init"},
		{"a worktree named add", []string{"go", "add"}, options{}, "add"},
		{"a worktree named switch", []string{"go", "switch"}, options{}, "switch"},
		{"a worktree named remove", []string{"go", "remove"}, options{}, "remove"},
		{"a worktree named move", []string{"go", "move"}, options{}, "move"},
		{"a worktree named list", []string{"go", "list"}, options{}, "list"},
		{"a worktree named go", []string{"go", "go"}, options{}, "go"},
		{"a worktree named config", []string{"go", "config"}, options{}, "config"},
		{"a worktree named help", []string{"go", "help"}, options{}, "help"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got options
			var target string
			err := execute(t, tt.args, io.Discard, front{reach: runs(func(o options, id string) (worktree.Handoff, error) {
				got, target = o, id
				return worktree.Handoff{}, nil
			})})
			if err != nil {
				t.Fatalf("Execute(%q): %v", tt.args, err)
			}
			if !same(got, tt.want) || target != tt.target {
				t.Errorf("Execute(%q) = %+v, %q; want %+v, %q", tt.args, got, target, tt.want, tt.target)
			}
		})
	}
}

// switch takes the identifier in the argument position and carries the open-on
// flags wherever those sit. Its argument is optional, the picker standing in.
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
		{"an action for the picker's target", []string{"switch", "--claude"}, options{open: "claude"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got options
			var target string
			err := execute(t, tt.args, io.Discard, front{enter: runs(func(o options, id string) (worktree.Handoff, error) {
				got, target = o, id
				return worktree.Handoff{}, nil
			})})
			if err != nil {
				t.Fatalf("Execute(%q): %v", tt.args, err)
			}
			if !same(got, tt.want) || target != tt.target {
				t.Errorf("Execute(%q) = %+v, %q; want %+v, %q", tt.args, got, target, tt.want, tt.target)
			}
		})
	}
}

// Every flag the bare form takes is declared once, on the verb it reaches: the
// open-on set on each opening verb, the flags that call an action off on the
// verbs that can run one.
func TestTheOpenOnFlagsAreEachVerbs(t *testing.T) {
	root := command(stubVersion, wired(), front{})
	for _, verb := range openingVerbs {
		cmd, _, err := root.Find([]string{verb})
		if err != nil {
			t.Fatalf("finding %s: %v", verb, err)
		}
		for _, name := range []string{"claude", "shell", "no-claim"} {
			if root.Flags().Lookup(name) != nil {
				t.Errorf("--%s is still declared on the root", name)
			}
		}
		for _, name := range []string{"claude", "shell"} {
			if cmd.Flags().Lookup(name) == nil {
				t.Errorf("%s does not declare --%s", verb, name)
			}
		}
		if declares, creates := cmd.Flags().Lookup("no-claim") != nil, slices.Contains(creatingVerbs, verb); declares != creates {
			t.Errorf("%s declares --no-claim: %v; want %v", verb, declares, creates)
		}
	}
}

// A flag an action used to answer to is refused with the one it answers to now,
// wherever the set is carried, and is offered by nothing.
func TestTheRenamedFlagIsRefusedByItsNewName(t *testing.T) {
	for _, args := range [][]string{{"bd-1", "--agent"}, {"go", "--agent", "bd-1"}, {"switch", "--agent", "bd-1"}, {"add", "--agent", "scratch"}} {
		err := execute(t, args, io.Discard, front{})
		if err == nil || !strings.Contains(err.Error(), "--claude") {
			t.Errorf("Execute(%q) = %v; want a refusal naming --claude", args, err)
		}
	}
	root := command(stubVersion, wired(), front{})
	for _, verb := range openingVerbs {
		cmd, _, err := root.Find([]string{verb})
		if err != nil {
			t.Fatalf("finding %s: %v", verb, err)
		}
		if flag := cmd.Flags().Lookup("agent"); flag == nil || !flag.Hidden {
			t.Errorf("%s's --agent is %+v; want a hidden flag", verb, flag)
		}
	}
}

// add takes the identifier in the argument position and carries the open-on
// flags wherever those sit. Its argument is optional, the picker standing in.
func TestAddFlags(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		want   options
		target string
	}{
		{"a name of its own", []string{"add", "scratch"}, options{}, "scratch"},
		{"no identifier", []string{"add"}, options{}, ""},
		{"named before the name", []string{"add", "--shell", "scratch"}, options{open: "shell"}, "scratch"},
		{"named after the name", []string{"add", "scratch", "--shell"}, options{open: "shell"}, "scratch"},
		{"handed to claude", []string{"add", "--claude", "scratch"}, options{open: "claude"}, "scratch"},
		// A ticket add creates is claimed, so the flag that calls the claim off stands
		// here too.
		{"declining the claim", []string{"add", "--no-claim", "bd-1"}, options{skip: []string{"beads"}}, "bd-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got options
			var target string
			err := execute(t, tt.args, io.Discard, front{add: runs(func(o options, id string) (worktree.Handoff, error) {
				got, target = o, id
				return worktree.Handoff{}, nil
			})})
			if err != nil {
				t.Fatalf("Execute(%q): %v", tt.args, err)
			}
			if !same(got, tt.want) || target != tt.target {
				t.Errorf("Execute(%q) = %+v, %q; want %+v, %q", tt.args, got, target, tt.want, tt.target)
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
			err := execute(t, tt.args, io.Discard, front{remove: runs(func(f bool, id string) (work.Deletion, error) {
				force, target = f, id
				return work.Deletion{}, nil
			})})
			if err != nil {
				t.Fatalf("Execute(%q): %v", tt.args, err)
			}
			if force != tt.force || target != tt.target {
				t.Errorf("Execute(%q) = %v, %q; want %v, %q", tt.args, force, target, tt.force, tt.target)
			}
		})
	}
}

// move is a verb of its own, taking the name and the destination in the two
// argument positions. Either may be left out, the picker standing in for the
// first and the prompt for the second.
func TestMoveFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		target, dest string
	}{
		{"a named worktree and where it goes", []string{"move", "scratch", "settled"}, "scratch", "settled"},
		{"a destination that is a path", []string{"move", "scratch", "../beside"}, "scratch", "../beside"},
		{"no destination", []string{"move", "scratch"}, "scratch", ""},
		{"neither", []string{"move"}, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target, dest string
			err := execute(t, tt.args, io.Discard, front{move: runs(func(id, to string) (work.Move, error) {
				target, dest = id, to
				return work.Move{}, nil
			})})
			if err != nil {
				t.Fatalf("Execute(%q): %v", tt.args, err)
			}
			if target != tt.target || dest != tt.dest {
				t.Errorf("Execute(%q) = %q, %q; want %q, %q", tt.args, target, dest, tt.target, tt.dest)
			}
		})
	}
}

// go names add for a name nothing answered for; the verbs that only act on a
// worktree already open do not.
func TestOnlyGoNamesAddForAnIdentifierNothingAnswersFor(t *testing.T) {
	repo := testenv.InitRepo(t)
	t.Chdir(repo)
	f := standing(last{})

	if _, err := f.reach.run(options{}, "typo"); err == nil || !strings.Contains(err.Error(), "work add typo") {
		t.Errorf("reach(typo) = %v; want it refused, naming the verb that makes a worktree of it", err)
	}

	acting := map[string]error{}
	_, acting["remove"] = f.remove.run(false, "typo")
	_, acting["move"] = f.move.run("typo", "other")
	_, acting["switch"] = f.enter.run(options{}, "typo")
	for verb, err := range acting {
		if err == nil || !strings.Contains(err.Error(), `nothing answers for "typo"`) {
			t.Errorf("%s(typo) = %v; want it refused as a name nothing answers for", verb, err)
		}
		if strings.Contains(err.Error(), "work add") {
			t.Errorf("%s(typo) = %v; want no invitation to create what it cannot act on", verb, err)
		}
	}
}

// A worktree goes by its branch where nothing else names it, and a branch may be
// spelled with separators in it. The question carries the directory instead, so
// that the answer left as it stands is a bare name and not a path.
func TestMoveAsksWithTheDirectoryNotTheBranch(t *testing.T) {
	repo := testenv.InitRepo(t)
	dir := filepath.Join(repo, config.Default().Worktree.Dir())
	testenv.Git(t, repo, "worktree", "add", "-b", "feature/login", filepath.Join(dir, "login"))
	t.Chdir(repo)
	ran := testenv.Stubs(t, testenv.Stub{Name: "fzf", Says: "login\n", Exits: 1})

	if _, err := standing(last{}).move.run("feature/login", ""); err == nil {
		t.Error("move onto the directory it already sits in: want it refused")
	}
	if asked := strings.Join(ran(), "\n"); !strings.Contains(asked, "--query login") {
		t.Errorf("the destination was asked for as %q; want the directory in it", asked)
	}
}

// A move given no destination asks for one with the current name already in it,
// so the answer is the name edited rather than retyped.
func TestMoveAsksForTheDestination(t *testing.T) {
	repo := testenv.InitRepo(t)
	dir := filepath.Join(repo, config.Default().Worktree.Dir())
	testenv.Git(t, repo, "worktree", "add", "-b", "scratch", filepath.Join(dir, "scratch"))
	t.Chdir(repo)
	ran := testenv.Stubs(t, testenv.Stub{Name: "fzf", Says: "settled\n", Exits: 1})

	m, err := standing(last{}).move.run("scratch", "")
	if err != nil {
		t.Fatalf("move: %v", err)
	}

	if asked := strings.Join(ran(), "\n"); !strings.Contains(asked, "--query scratch") {
		t.Errorf("the destination was asked for as %q; want the current name in it", asked)
	}
	if want := filepath.Join(dir, "settled"); m.To != want || m.Now != "settled" {
		t.Errorf("move = %+v; want it at %q on branch settled", m, want)
	}
}

// What a candidate is refused over is known from the candidate alone, so the
// refusal lands where a destination was given and where one was not, and the
// question is never put where it is thrown away.
func TestMoveRefusesBeforeAsking(t *testing.T) {
	tests := []struct {
		name, target, want string
		stoodIn            bool
	}{
		{"the worktree stood in", "scratch", "standing in", true},
		{"no worktree open", "nowhere", "no worktree", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testenv.InitRepo(t)
			path := filepath.Join(repo, config.Default().Worktree.Dir(), "scratch")
			testenv.Git(t, repo, "worktree", "add", "-b", "scratch", path)
			from := repo
			if tt.stoodIn {
				from = path
			}
			t.Chdir(from)
			ran := testenv.Stubs(t, testenv.Stub{Name: "fzf", Says: "settled\n", Exits: 1})

			f := standing(anything{})
			// A destination given and one left for the prompt alike: neither is what the
			// refusal waits on.
			for _, dest := range []string{"", "settled"} {
				if _, err := f.move.run(tt.target, dest); err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Errorf("move(%q, %q) = %v; want a refusal naming %q", tt.target, dest, err, tt.want)
				}
			}
			if asked := ran(); len(asked) > 0 {
				t.Errorf("the destination was asked for as %q; want the refusal ahead of the question", asked)
			}
		})
	}
}

// What moved is printed on the writer cobra handed the command, a line per half,
// and a branch that took no new name is a half that did not happen.
func TestMovePrints(t *testing.T) {
	moved := work.Move{
		From: "/repo/.worktrees/bd-1", To: "/repo/.worktrees/settled",
		Was: "bd-1-do-a-thing", Now: "settled",
	}
	tests := []struct {
		name string
		went work.Move
		want string
	}{
		{
			"a worktree and its branch",
			moved,
			"moved worktree /repo/.worktrees/bd-1 to /repo/.worktrees/settled\nrenamed branch bd-1-do-a-thing to settled\n",
		},
		{
			"a detached worktree",
			work.Move{From: "/repo/.worktrees/spike", To: "/elsewhere/spike"},
			"moved worktree /repo/.worktrees/spike to /elsewhere/spike\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			f := front{move: runs(func(string, string) (work.Move, error) { return tt.went, nil })}
			if err := execute(t, []string{"move", "scratch", "settled"}, &out, f); err != nil {
				t.Fatalf("work move: %v", err)
			}
			if out.String() != tt.want {
				t.Errorf("work move printed %q; want %q", out.String(), tt.want)
			}
		})
	}

	f := front{move: runs(func(string, string) (work.Move, error) { return moved, nil })}
	if err := execute(t, []string{"move", "scratch", "settled"}, refusing{}, f); err == nil {
		t.Error("work move onto a writer that refuses: want the write's error")
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
			f := front{remove: runs(func(bool, string) (work.Deletion, error) { return tt.went, nil })}
			if err := execute(t, []string{"remove", "scratch"}, &out, f); err != nil {
				t.Fatalf("work remove: %v", err)
			}
			if out.String() != tt.want {
				t.Errorf("work remove printed %q; want %q", out.String(), tt.want)
			}
		})
	}

	f := front{remove: runs(func(bool, string) (work.Deletion, error) { return both, nil })}
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
	err := execute(t, []string{"config", "edit"}, &out, front{edit: func(io.Writer) error {
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
	f := front{branches: func() ([]string, error) { return []string{"bd-1-do-a-thing", "spike"}, nil }}
	for _, list := range lists(&f) {
		*list = resolved
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
		{"two identifiers on go", []string{"go", "bd-1", "bd-2"}},
		{"two actions on go at once", []string{"go", "bd-1", "--shell", "--claude"}},
		{"a verb's flag on go", []string{"go", "bd-1", "--force"}},
		{"two identifiers on switch", []string{"switch", "bd-1", "bd-2"}},
		{"two actions on switch at once", []string{"switch", "bd-1", "--shell", "--claude"}},
		{"a verb's flag on switch", []string{"switch", "bd-1", "--force"}},
		// switch only enters, so no action of its own ever runs to be called off.
		{"declining a claim on switch", []string{"switch", "bd-1", "--no-claim"}},
		// Creating is a verb now.
		{"--create is gone", []string{"scratch", "--create"}},
		{"adding two worktrees at once", []string{"add", "scratch", "other"}},
		{"two actions on add at once", []string{"add", "scratch", "--shell", "--claude"}},
		// Removing is a verb now, and --force went with it.
		{"--delete is gone", []string{"scratch", "--delete"}},
		{"--force is gone from the root", []string{"scratch", "--force"}},
		// remove declares --force and nothing else, so a root flag is unknown to it.
		{"a root flag on remove", []string{"remove", "scratch", "--shell"}},
		{"removing two worktrees at once", []string{"remove", "scratch", "other"}},
		// move takes the two positions and no flag at all.
		{"a name, a destination and a third word", []string{"move", "scratch", "settled", "over"}},
		{"a root flag on move", []string{"move", "scratch", "--shell"}},
		{"another verb's flag on move", []string{"move", "scratch", "--force"}},
		// Listing takes no argument: a name to filter on is go's.
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
		{"init with a shell work does not print", []string{"init", "tcsh"}},
		{"init with two shells", []string{"init", "bash", "fish"}},
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

// runs is a verb doing what a case wants of it and offering nothing:
// [executeOn] fills in a listing left unnamed.
func runs[T any](run T) offering[T] { return offering[T]{run: run} }

// lists is what each verb in a front completes with, under the verb it belongs to.
func lists(f *front) map[string]*func() ([]work.Candidate, error) {
	return map[string]*func() ([]work.Candidate, error){
		"go": &f.reach.list, "switch": &f.enter.list, "add": &f.add.list,
		"remove": &f.remove.list, "move": &f.move.list,
	}
}

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
	if f.reach.run == nil {
		f.reach.run = func(options, string) (worktree.Handoff, error) {
			t.Error("reached a place")
			return worktree.Handoff{}, nil
		}
	}
	if f.enter.run == nil {
		f.enter.run = func(options, string) (worktree.Handoff, error) {
			t.Error("entered a worktree")
			return worktree.Handoff{}, nil
		}
	}
	if f.add.run == nil {
		f.add.run = func(options, string) (worktree.Handoff, error) {
			t.Error("added a worktree")
			return worktree.Handoff{}, nil
		}
	}
	if f.remove.run == nil {
		f.remove.run = func(bool, string) (work.Deletion, error) {
			t.Error("removed a worktree")
			return work.Deletion{}, nil
		}
	}
	if f.move.run == nil {
		f.move.run = func(string, string) (work.Move, error) {
			t.Error("moved a worktree")
			return work.Move{}, nil
		}
	}
	if f.dump == nil {
		f.dump = func(io.Writer) error { t.Error("dumped the configuration"); return nil }
	}
	if f.edit == nil {
		f.edit = func(io.Writer) error { t.Error("opened the settings file"); return nil }
	}
	for _, list := range lists(&f) {
		if *list == nil {
			*list = stub(nil, nil)
		}
	}
	if f.branches == nil {
		f.branches = func() ([]string, error) { t.Error("listed the branches"); return nil, nil }
	}
	cmd := command(stubVersion, sys, f)
	cmd.SetOut(out)
	cmd.SetErr(io.Discard)
	return run(cmd, args)
}

// A system that would not answer is said on the way past, so a listing short of
// its rows does not read as a repository with none. Each refusal is one line,
// naming the command that was put and what came back.
func TestEveryRefusalIsSaidOnce(t *testing.T) {
	var said strings.Builder
	report(&said, []error{
		errors.New("bd ready --limit 0 --json: the database is not there"),
		errors.New("gh pr list: gh is not on PATH"),
	})

	want := "work: bd ready --limit 0 --json: the database is not there\n" +
		"work: gh pr list: gh is not on PATH\n"
	if said.String() != want {
		t.Errorf("work said %q; want %q", said.String(), want)
	}
}

// wiring is a repository's systems as a case names them, read without settings.
func wiring(chain ...work.Resolver) work.Wiring {
	return func(string, string, config.Config) work.Systems {
		return work.Systems{Resolvers: chain}
	}
}

// standing is the front the shell stands in, under resolvers of the case's own:
// each verb wired to its listing the way [Execute] wires it.
func standing(chain ...work.Resolver) front { return fronting(verbs{wire: wiring(chain...)}) }

// pickers is each verb's screen put up with no argument given, under the verb it
// belongs to.
func pickers(f front) map[string]func() error {
	return map[string]func() error{
		"go":     func() error { _, err := f.reach.run(options{}, ""); return err },
		"switch": func() error { _, err := f.enter.run(options{}, ""); return err },
		"add":    func() error { _, err := f.add.run(options{}, ""); return err },
		"remove": func() error { _, err := f.remove.run(false, ""); return err },
		"move":   func() error { _, err := f.move.run("", ""); return err },
	}
}

// last is the resolver at the end of a chain: it answers for whatever worktree
// git reports, the main checkout included, and for no identifier.
type last struct{}

func (last) Name() string { return "last" }

func (last) Identify(id string, o worktree.Open) (worktree.Place, error) {
	// Nothing behind it to be named by, so the directory settles an identifier.
	if o.None() || (id != "" && id != o.Path) {
		return worktree.Place{}, worktree.ErrUnknown
	}
	return worktree.Place{ID: o.Path, Name: o.Name(), Branch: o.Branch}, nil
}

func (last) Offer() ([]worktree.Place, error)                 { return nil, nil }
func (last) Prepare(p worktree.Place) (worktree.Place, error) { return p, nil }
func (last) Create(worktree.Place, string) error              { return nil }

// anything is last with an identifier of its own: a name no worktree is open for
// still resolves, which is a candidate to refuse rather than one nothing answers
// for.
type anything struct{ last }

func (anything) Name() string { return "anything" }

func (a anything) Identify(id string, o worktree.Open) (worktree.Place, error) {
	if o.None() {
		return worktree.Place{ID: id, Name: id, Branch: id}, nil
	}
	return a.last.Identify(id, o)
}

// Every verb that reaches the core reaches it through the wiring the call
// carries, so two wirings in one process each answer for their own systems.
// Parked in a package-level var, whichever was assigned last would answer for
// both, and which systems a verb reads would depend on what set the var up.
func TestEachCallCarriesItsOwnWiring(t *testing.T) {
	t.Chdir(testenv.InitRepo(t))

	var asked []string
	calls := []struct {
		name string
		ask  func(front) error
	}{
		{"go", func(f front) error { _, err := f.reach.list(); return err }},
		{"switch", func(f front) error { _, err := f.enter.list(); return err }},
		{"add", func(f front) error { _, err := f.add.list(); return err }},
		{"remove", func(f front) error { _, err := f.remove.list(); return err }},
		{"move", func(f front) error { _, err := f.move.list(); return err }},
		{"branches", func(f front) error { _, err := f.branches(); return err }},
	}
	var want []string
	for _, c := range calls {
		wire := func(repo, checkout string, cfg config.Config) work.Systems {
			asked = append(asked, c.name)
			return wiring(last{})(repo, checkout, cfg)
		}
		if err := c.ask(fronting(verbs{wire: wire})); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		want = append(want, c.name)
	}
	if !slices.Equal(asked, want) {
		t.Errorf("the wirings asked were %q; want %q", asked, want)
	}
}

// making is a resolver whose creation is git's own: what a verb leaves the
// checkout carrying is read against a worktree that really exists.
type making struct {
	anything
	repo string
}

func (m making) Create(p worktree.Place, path string) error {
	return git.NewWorktree(m.repo, path, p.Branch)
}

// carrying is a repository the shell stands in with work in hand, and the front
// over it: git makes the worktrees, and the shell is what one opens on.
func carrying(t *testing.T) (string, front) {
	t.Helper()
	repo := testenv.InitRepo(t)
	testenv.Write(t, filepath.Join(repo, ".gitignore"), config.Default().Worktree.Dir()+"/\n")
	testenv.Git(t, repo, "add", ".gitignore")
	testenv.Git(t, repo, "commit", "-m", "ignore the worktrees")
	testenv.Write(t, filepath.Join(repo, "wip"), "in hand")
	t.Chdir(repo)

	sys := work.Systems{
		Resolvers: []work.Resolver{making{repo: repo}},
		Openers:   []work.Opener{opener{string(config.ActionShell)}},
	}
	return repo, fronting(verbs{wire: func(string, string, config.Config) work.Systems { return sys }})
}

// Parking is the verb's and not the creation's: work add carries the checkout's
// work into the worktree it makes, and work go creating one leaves it in hand.
func TestOnlyAddParksWhatTheCheckoutWasCarrying(t *testing.T) {
	repo, f := carrying(t)
	if _, err := f.add.run(options{}, "parked"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if left := testenv.Git(t, repo, "status", "--porcelain"); left != "" {
		t.Errorf("add left the checkout carrying %q; want it clean", left)
	}
	made := filepath.Join(repo, config.Default().Worktree.Dir(), "parked")
	if got := testenv.Git(t, made, "status", "--porcelain"); got != "?? wip" {
		t.Errorf("the worktree add made carries %q; want the work that was in hand", got)
	}

	repo, f = carrying(t)
	if _, err := f.reach.run(options{}, "reached"); err != nil {
		t.Fatalf("go: %v", err)
	}
	if got := testenv.Git(t, repo, "status", "--porcelain"); got != "?? wip" {
		t.Errorf("go left the checkout carrying %q; want the work still in hand", got)
	}
}
