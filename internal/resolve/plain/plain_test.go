package plain

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// An identifier is taken as it is: the branch is spelled exactly as the name is,
// so a number is a worktree of that name rather than the pull request another
// resolver would read into it. Only the verb that names this resolver gets here.
func TestIdentifyTakesAnIdentifierAtItsWord(t *testing.T) {
	r := Named(t.TempDir(), t.TempDir())
	for _, name := range []string{"scratch", "7", "one-two", "pr-007"} {
		got, err := r.Identify(name, worktree.Open{})
		if err != nil {
			t.Fatalf("Identify(%q): %v", name, err)
		}
		want := worktree.Place{ID: name, Name: name, Branch: name}
		if got != want {
			t.Errorf("Identify(%q) = %+v; want %+v", name, got, want)
		}
	}
}

// In the chain there is no name to take at its word: an identifier reaching the
// last resolver is one nothing recognised, and making a branch of it would make
// one of every typo and of every identifier whose system is switched off.
func TestIdentifyInTheChainRecognisesNoIdentifier(t *testing.T) {
	r := New(t.TempDir(), t.TempDir())
	for _, name := range []string{"scratch", "7", "bd-42"} {
		if _, err := r.Identify(name, worktree.Open{}); !errors.Is(err, worktree.ErrUnknown) {
			t.Errorf("Identify(%q) = %v; want the chain to move on", name, err)
		}
	}
}

// A worktree is whatever is left, so one nothing else recognises is still one to
// reach: it is named by its branch, and by its directory where it has none.
func TestIdentifyNamesAWorktreeNothingElseClaims(t *testing.T) {
	r := New(t.TempDir(), t.TempDir())
	tests := []struct {
		name string
		open worktree.Open
		want worktree.Place
	}{
		{
			"by its branch",
			worktree.Open{Path: "/wt/loose", Branch: "some-branch"},
			worktree.Place{ID: "/wt/loose", Name: "some-branch", Branch: "some-branch"},
		},
		{
			"detached, by its directory",
			worktree.Open{Path: "/wt/loose"},
			worktree.Place{ID: "/wt/loose", Name: "loose"},
		},
		{
			// The branch a canonicalising resolver would have spelled otherwise is nobody
			// else's, so it lands here under the name it actually carries.
			"a branch another resolver would not own",
			worktree.Open{Path: "/wt/pr", Branch: "pr-007"},
			worktree.Place{ID: "/wt/pr", Name: "pr-007", Branch: "pr-007"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Identify("", tt.open)
			if err != nil {
				t.Fatalf("Identify: %v", err)
			}
			if got != tt.want {
				t.Errorf("Identify = %+v; want %+v", got, tt.want)
			}
		})
	}
}

// A plain worktree has nothing behind it to be named by, so it is only itself:
// asked whether a worktree is a given place's, the directory is the whole answer.
func TestIdentifyConfirmsAPlaceByItsDirectory(t *testing.T) {
	r := New(t.TempDir(), t.TempDir())
	here := t.TempDir()
	open := worktree.Open{Path: here, Branch: "some-branch"}

	if _, err := r.Identify(here, open); err != nil {
		t.Errorf("Identify(its own directory) = %v; want the worktree taken as that place's", err)
	}
	// The same branch elsewhere is another worktree, and this one's name is not an
	// answer either: a plain place is its path.
	for _, id := range []string{t.TempDir(), "some-branch"} {
		if _, err := r.Identify(id, open); !errors.Is(err, worktree.ErrUnknown) {
			t.Errorf("Identify(%q) = %v; want the worktree left to another resolver", id, err)
		}
	}
}

// A name of the user's own is typed, never listed, so this resolver adds no rows
// to the picker.
func TestOfferOffersNothing(t *testing.T) {
	got, err := New(t.TempDir(), t.TempDir()).Offer()
	if err != nil || len(got) != 0 {
		t.Errorf("Offer() = %+v, %v; want nothing offered", got, err)
	}
}

// Preparing asserts the name is free: a branch already holding it is a worktree
// to re-enter, which is the same name without the verb.
func TestPrepareRefusesATakenBranch(t *testing.T) {
	repo := testenv.InitRepo(t)
	testenv.Git(t, repo, "branch", "taken")
	r := Named(repo, repo)

	p, err := r.Prepare(worktree.Place{ID: "taken", Name: "taken", Branch: "taken"})
	if err == nil || !strings.Contains(err.Error(), "work taken") {
		t.Errorf("Prepare(taken) = %+v, %v; want the branch named and work taken pointed at", p, err)
	}

	if _, err := r.Prepare(worktree.Place{ID: "free", Name: "free", Branch: "free"}); err != nil {
		t.Errorf("Prepare(free) = %v; want a name nothing holds prepared", err)
	}
}

// Creating forks a branch of its own from what the checkout work was invoked in
// has at HEAD, spelled exactly as the name is, so a worktree asked for from
// inside another starts from the work under foot rather than from what the main
// checkout happens to have. It still lands where the core says.
func TestCreateForksFromTheCurrentCheckout(t *testing.T) {
	repo := testenv.InitRepo(t)
	main := testenv.Git(t, repo, "rev-parse", "HEAD")
	under := filepath.Join(repo, "trees", "under-foot")
	testenv.Git(t, repo, "worktree", "add", "-b", "under-foot", under)
	testenv.Git(t, under, "commit", "--allow-empty", "-m", "not on main yet")
	ahead := testenv.Git(t, under, "rev-parse", "HEAD")
	r := Named(repo, under)

	place, err := r.Prepare(worktree.Place{ID: "scratch", Name: "scratch", Branch: "scratch"})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	path := filepath.Join(repo, "trees", "scratch")
	if err := r.Create(place, path); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if head := testenv.Git(t, repo, "rev-parse", "scratch"); head != ahead {
		t.Errorf("branch scratch is at %q; want the checkout under foot at %q rather than the main checkout's %q", head, ahead, main)
	}
	list, err := git.Worktrees(repo)
	if err != nil {
		t.Fatalf("Worktrees: %v", err)
	}
	if !slices.ContainsFunc(list, func(w git.Worktree) bool {
		return git.SameDir(w.Path, path) && w.Branch == "scratch"
	}) {
		t.Errorf("the repository has %+v; want the worktree at %q on branch scratch", list, path)
	}
}

// This resolver speaks to git and to no system beyond it, so it has no values to
// supply and never becomes the source of one.
func TestResolverSuppliesNothing(t *testing.T) {
	if _, ok := any(New(t.TempDir(), t.TempDir())).(worktree.Source); ok {
		t.Error("the plain resolver supplies values; it has no system behind it to know any")
	}
}
