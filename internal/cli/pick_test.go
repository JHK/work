package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
)

// A listing left with no rows is refused in one line naming what there is
// nothing of, rather than an empty screen to dismiss: fzf is never reached.
func TestAnEmptyListingIsRefusedRatherThanPutUp(t *testing.T) {
	// The main checkout alone, standing in it: switch leaves out the worktree stood
	// in, remove and move leave out that and the main checkout, and add leaves out
	// what is already open, so every one of these four is empty. go still has the
	// main checkout to offer, so it is not asked.
	t.Chdir(testenv.InitRepo(t))
	ran := testenv.Stubs(t, testenv.Stub{Name: "fzf", Says: "0\trow\n"})
	screens := pickers(opened(t, last{}))

	for verb, want := range map[string]string{
		"switch": "no worktree to switch to",
		"remove": "no worktree to remove",
		"move":   "no worktree to move",
		"add":    "nothing left to add",
	} {
		t.Run(verb, func(t *testing.T) {
			if err := screens[verb](); err == nil || err.Error() != want {
				t.Errorf("work %s = %v; want the one line %q", verb, err, want)
			}
		})
	}
	if asked := ran(); len(asked) > 0 {
		t.Errorf("the picker was put up as %q; want nothing on offer to be nothing to put up", asked)
	}
}

// The prompt hands fzf the answer already typed and takes back what was typed
// over it, which is the query rather than a row: fzf exits 1 on a query matching
// none of the nothing on offer, and that is every answer there is.
func TestAskTakesTheQueryBack(t *testing.T) {
	ran := testenv.Stubs(t, testenv.Stub{Name: "fzf", Says: "settled\n", Exits: 1})

	got, err := ask("scratch")
	if err != nil || got != "settled" {
		t.Fatalf("ask = %q, %v; want settled", got, err)
	}
	asked := strings.Join(ran(), "\n")
	for _, want := range []string{"--print-query", "--query scratch"} {
		if !strings.Contains(asked, want) {
			t.Errorf("fzf was run as %q; want %s", asked, want)
		}
	}
}

// An answer nobody gave is nothing to act on: an interruption says nothing and
// exits 130, and a query emptied out says nothing either.
func TestAskWithoutAnAnswer(t *testing.T) {
	tests := []struct {
		name string
		stub testenv.Stub
	}{
		{"interrupted", testenv.Stub{Name: "fzf", Exits: 130}},
		{"emptied out", testenv.Stub{Name: "fzf", Says: "\n", Exits: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testenv.Stubs(t, tt.stub)
			if got, err := ask("scratch"); !errors.Is(err, errCancelled) {
				t.Errorf("ask = %q, %v; want it cancelled", got, err)
			}
		})
	}
}

// Any other failure, a missing binary above all, is one the user has to be told
// about rather than a destination of its own.
func TestAskWhenFzfFails(t *testing.T) {
	testenv.Stubs(t, testenv.Stub{Name: "fzf", Says: "settled\n", Exits: 2})

	if got, err := ask("scratch"); err == nil || errors.Is(err, errCancelled) || !strings.Contains(err.Error(), "fzf") {
		t.Errorf("ask = %q, %v; want a failure naming fzf", got, err)
	}
}
