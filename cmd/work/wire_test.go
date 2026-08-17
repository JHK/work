package main

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	actionbeads "github.com/JHK/work-cli/internal/action/beads"
	"github.com/JHK/work-cli/internal/config"
	resolvebeads "github.com/JHK/work-cli/internal/resolve/beads"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// The flags the command line records are spelled by the systems themselves:
// internal/cli holds no list of them, so a system dropped or renamed here takes
// its flag with it and nothing else has to be told.
func TestTheWiringSpellsTheFlagsTheCommandLineRecords(t *testing.T) {
	// Asked the way the front end asks it to name the flags, before there is a
	// repository: a constructor reaching for one here would be reaching on every
	// invocation, work init and work --help included.
	systems := wire("", "", config.Shipped())

	// An opener is named by its flag, so every one of them spells one and says what
	// it does; an action is declined by its flag, and one that spells none runs
	// whenever a worktree comes into being.
	opens := map[string]bool{}
	for _, op := range systems.Openers {
		f, ok := op.(worktree.Flagged)
		if !ok {
			t.Errorf("the %s action spells no flag; nothing on the command line names it", op.Name())
			continue
		}
		name, usage := f.Flag()
		if usage == "" {
			t.Errorf("--%s says nothing in --help", name)
		}
		opens[name] = true
	}
	for _, want := range []string{"claude", "shell"} {
		if !opens[want] {
			t.Errorf("no action spells --%s; the flag the command line records is gone", want)
		}
	}

	declines := map[string]bool{}
	for _, a := range systems.Actions {
		if f, ok := a.(worktree.Flagged); ok {
			name, _ := f.Flag()
			declines[name] = true
		}
	}
	if !declines["no-claim"] {
		t.Error("no action spells --no-claim; the flag the command line records is gone")
	}
}

// The picker draws each row by the mark the resolver that answered for it names,
// so a resolver naming none is a row that reaches the screen blank, and two
// naming the same one are two kinds of place a reader cannot tell apart.
func TestEveryResolverWiredDrawsItsOwnRows(t *testing.T) {
	repo := t.TempDir()
	systems := wire(repo, repo, config.Shipped())

	marks := map[string]string{}
	for _, r := range systems.Resolvers {
		d, ok := r.(worktree.Drawn)
		if !ok || d.Icon() == "" {
			t.Errorf("%s marks its rows with nothing; the picker draws them blank", r.Name())
			continue
		}
		if by, taken := marks[d.Icon()]; taken {
			t.Errorf("%s and %s both mark their rows %q", by, r.Name(), d.Icon())
		}
		// The picker lines the names up behind one mark's width, so a mark of two
		// shifts every row its resolver answered for.
		if n := utf8.RuneCountInString(d.Icon()); n != 1 {
			t.Errorf("%s marks its rows with %d runes; want the one column the picker pads for", r.Name(), n)
		}
		marks[d.Icon()] = r.Name()
	}
}

// One resolver list serves two orders, and no type states either.
//
// For an identifier the first resolver to answer takes it, so a resolver answering
// for any identifier makes every resolver after it unreachable for identifiers. For an
// open worktree the last to answer takes whatever is left, so a resolver answering for
// any worktree has to be last or the ones behind it are never asked.
//
// plain answers for any worktree, so it is wired last. Nothing answers for any
// identifier: a name no system knows is work add's to invent.
func TestTheResolverOrderHoldsBothWays(t *testing.T) {
	repo := t.TempDir()
	// An order between systems can only be read off a wiring that has them all.
	systems := wire(repo, repo, config.Shipped())
	if len(systems.Resolvers) == 0 {
		t.Fatal("nothing is wired")
	}

	// An identifier no pattern discriminates on, and a worktree on a branch none can
	// name: what answers for either is answering for anything.
	const anything = "an-identifier-no-pattern-matches"
	somewhere := worktree.Open{
		Path:   filepath.Join(repo, "elsewhere"),
		Branch: "a-branch-no-pattern-names",
	}

	var ids, trees []string
	for _, r := range systems.Resolvers {
		if _, err := r.Identify(anything, worktree.Open{}); err == nil {
			ids = append(ids, r.Name())
		}
		if _, err := r.Identify("", somewhere); err == nil {
			trees = append(trees, r.Name())
		}
	}
	if len(ids) > 0 {
		t.Errorf("%s answers for an identifier nothing discriminates on; a name no system knows is work add's, and every resolver after %s would never be asked",
			ids[0], ids[0])
	}
	if len(trees) == 0 {
		t.Fatal("no resolver answers for a worktree nothing recognises; the listing would refuse a worktree that is there")
	}

	last := systems.Resolvers[len(systems.Resolvers)-1].Name()
	if trees[0] != last {
		t.Errorf("%s answers for any worktree but %s is wired last; every resolver after %s is never asked about one",
			trees[0], last, trees[0])
	}
}

// The settings the command line spells its flags from leave no system out. A
// system this wiring knows and those settings do not is one whose flag no
// repository can reach, wherever it is enabled, and the flag set is settled
// before any repository is read.
func TestTheShippedSettingsLeaveNoSystemOut(t *testing.T) {
	repo := t.TempDir()

	var off []string
	for _, s := range wire(repo, repo, config.Shipped()).Disabled {
		off = append(off, s.Name())
	}
	if len(off) > 0 {
		t.Errorf("config.Shipped leaves %q switched off; the command line spells no flag of theirs", off)
	}
}

// What a repository that says nothing gets is the core: a worktree to adopt, a
// name to make a place of, and something to open on. None of the three has a
// key, so no settings file can take them away either.
func TestTheWorktreesAreWhatARepositoryGetsWithNoSettings(t *testing.T) {
	repo := t.TempDir()
	systems := wire(repo, repo, load(t, repo))

	if _, err := systems.Resolvers[len(systems.Resolvers)-1].Identify("", worktree.Open{
		Path: filepath.Join(repo, "trees", "loose"), Branch: "loose",
	}); err != nil {
		t.Errorf("a worktree nothing recognises is left unadopted: %v", err)
	}
	if systems.Named == nil {
		t.Error("no resolver makes a place of a name; work add would have nothing to ask")
	}
	if len(systems.Openers) == 0 {
		t.Error("nothing is wired for a worktree to open on")
	}
}

// Switching a system on in a settings file is what wires it, on every seam it
// fills: beads resolves the tickets and claims them, and the one table answers
// for both halves.
//
// It goes through a file rather than through the fields behind one, because the
// table is named for the system and nothing in the compiler holds those two
// spellings together: internal/config spells the table, each implementation
// spells the name it goes by, and a refusal composes the key from the name. A
// table config does not know is refused at load.
func TestSwitchingASystemOnWiresIt(t *testing.T) {
	for _, name := range config.SystemNames() {
		t.Run(name, func(t *testing.T) {
			repo := t.TempDir()
			systems := wire(repo, repo, with(t, repo, name))

			if on := wired(systems); !slices.Contains(on, name) {
				t.Errorf("[%s] enabled = true wired %q; want the system its table names among them", name, on)
			}
		})
	}
}

// The action and the resolver behind the one tracker go by one name, which is
// what lets a place resolved by the one be recognised by the other, and what
// --no-claim spells. It is asked here because R4 leaves neither half naming the
// other, and the wiring is what names both.
func TestBothHalvesGoByOneName(t *testing.T) {
	if actionbeads.Name != resolvebeads.Name {
		t.Errorf("the action goes by %q and the resolver by %q; want one name on both seams", actionbeads.Name, resolvebeads.Name)
	}
}

// A repository that asks for none of them leaves every one among the switched
// off, which is where a refusal reads the key that puts a system back. One
// missing from that set is a system whose identifier is refused as a name work
// has never heard of.
func TestAnUnconfiguredRepositoryCountsEverySystemAmongTheSwitchedOff(t *testing.T) {
	repo := t.TempDir()

	var off []string
	for _, s := range wire(repo, repo, load(t, repo)).Disabled {
		off = append(off, s.Name())
	}
	slices.Sort(off)
	if want := slices.Sorted(slices.Values(config.SystemNames())); !slices.Equal(off, want) {
		t.Errorf("a repository that configured nothing counts %q among the switched off; want %q", off, want)
	}
}

// with is the settings a repository that asks for that system loads, read back
// from a file it wrote: the surface a user has, where a table config does not
// know is refused rather than quietly doing nothing.
func with(t *testing.T, repo, name string) config.Config {
	t.Helper()
	testenv.Write(t, filepath.Join(repo, config.RepoFile), "["+name+"]\nenabled = true\n")
	return load(t, repo)
}

// load is the settings that repository loads. A directory no file was written to
// is a repository that configured nothing, read through Load rather than taken
// from config.Default so that what a user gets by writing nothing is judged.
func load(t *testing.T, repo string) config.Config {
	t.Helper()
	cfg, err := config.Load(repo)
	if err != nil {
		t.Fatalf("the settings in %s: %v", repo, err)
	}
	return cfg
}

// A repository that configured nothing runs on its worktrees: every verb goes
// through, creating one included, and a machine with bd, gh, mise and claude on
// it hears from none of them. Refusing an identifier is the one moment work does
// put a question to a system that is off, to say which key would have answered
// for it, and that question reaches no tool either: a system asked anything would
// be running on a machine that never asked for it.
func TestNoToolIsAskedAnythingWhereNoSystemWasAskedFor(t *testing.T) {
	repo := testenv.InitRepo(t)
	// A stand-in for each tool, failing where it is reached at all, so that a
	// question put to one is a line in the log rather than whatever the machine
	// running the tests happens to have installed.
	ran := testenv.Stubs(t,
		testenv.Stub{Name: "bd", Exits: 1},
		testenv.Stub{Name: "gh", Exits: 1},
		testenv.Stub{Name: "mise", Exits: 1},
		testenv.Stub{Name: "claude", Exits: 1},
	)

	cfg := load(t, repo)
	e := work.Env{Repo: repo, Config: cfg, Systems: wire(repo, repo, cfg)}
	// The picker's rows, and switch with a name of its own.
	if _, _, err := e.Candidates(); err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	c, err := e.Add("scratch")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	// No key was configured, so this is action.create alone deciding what a fresh
	// worktree opens on, and it may not name a system this repository never asked for.
	h, err := e.Enter(c, work.Options{})
	if err != nil {
		t.Fatalf("Enter: %v", err)
	}
	// What a repository that configured nothing opens on is the worktree itself, so
	// the wiring answers a front end with a directory rather than a command.
	if !h.Directory() {
		t.Errorf("a worktree no key spoke for opens on %q; want the worktree itself", h.Run)
	}
	if _, err := e.Worktrees(); err != nil {
		t.Fatalf("Worktrees: %v", err)
	}
	again, err := e.Resolve("scratch")
	if err != nil {
		t.Fatalf(`Resolve("scratch"): %v`, err)
	}
	if _, err := e.Enter(again, work.Options{}); err != nil {
		t.Fatalf("Enter an existing worktree: %v", err)
	}
	if _, err := e.Delete(again, false); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// An identifier a switched-off system claims as its own, which is one the forge
	// reads off the spelling: the refusal names the key that puts the forge back.
	if _, err := e.Resolve("https://github.com/o/r/pull/7"); err == nil || !strings.Contains(err.Error(), config.SystemKey("github")) {
		t.Errorf(`Resolve(a pull request URL) = %v; want the key that would have answered for it`, err)
	}
	// A name no system claims. The tracker would have taken it, having taken
	// whatever the chain left, but a name is a ticket id only if bd says so, and
	// asking bd is what the repository switched off.
	for _, id := range []string{"bd-42", "nothing-of-anyones"} {
		_, err := e.Resolve(id)
		if err == nil {
			t.Errorf("Resolve(%q) = a place; want a name nothing here answers for refused", id)
			continue
		}
		if strings.Contains(err.Error(), "is off") {
			t.Errorf("Resolve(%q) = %v; want no system named for a spelling none of them claims", id, err)
		}
	}

	if asked := ran(); len(asked) > 0 {
		t.Errorf("work ran %s; want a repository that asked for none of them asking nothing of any",
			strings.Join(asked, ", "))
	}
}

// Listing the worktrees for a reader asks git and nothing else, on a repository
// that wired every system: which worktrees are open is git's own answer, so a
// tracker that cannot be reached costs a listing that never needed one.
func TestListingTheWorktreesAsksNoToolWhereverEverySystemIsOn(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Git(t, repo, "worktree", "add", "-b", "bd-1-a-slug", filepath.Join(repo, "trees", "bd-1"))
	ran := testenv.Stubs(t,
		testenv.Stub{Name: "bd", Exits: 1},
		testenv.Stub{Name: "gh", Exits: 1},
	)

	cfg := config.Shipped()
	e := work.Env{Repo: repo, Config: cfg, Systems: wire(repo, repo, cfg)}
	got, err := e.Branches()
	if err != nil {
		t.Fatalf("Branches: %v", err)
	}
	if want := []string{"main", "bd-1-a-slug"}; !slices.Equal(got, want) {
		t.Errorf("Branches() = %q; want %q", got, want)
	}
	if asked := ran(); len(asked) > 0 {
		t.Errorf("listing the worktrees ran %s; want git asked alone", strings.Join(asked, ", "))
	}
}

// wired are the systems behind the seams, under the names they go by.
func wired(s work.Systems) []string {
	var names []string
	for _, r := range s.Resolvers {
		names = append(names, r.Name())
	}
	for _, a := range s.Actions {
		names = append(names, a.Name())
	}
	for _, o := range s.Openers {
		names = append(names, o.Name())
	}
	return names
}
