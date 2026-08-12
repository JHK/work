package claude

import (
	"errors"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// fixed answers for what a worktree carries without one being written.
type fixed struct {
	has []string
	err error
}

func (f fixed) list(string) ([]string, error) { return f.has, f.err }

// stub is an Opener over the compiled-in commands, answering with these
// conversations.
func stub(has []string, err error) Opener {
	o := New(config.Claude{})
	o.carried = fixed{has: has, err: err}
	return o
}

// configured is the [claude] table a settings body names, read the way work
// reads one. A command's own fields are the package's, so a file is the only way
// to name one.
func configured(t *testing.T, body string) config.Claude {
	t.Helper()
	repo := t.TempDir()
	testenv.Write(t, filepath.Join(repo, config.RepoFile), body)
	cfg, err := config.Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg.Claude
}

// A worktree this run created opens on the first of its keys the values
// in hand can render, from the most that can be said about it to the least.
// Nothing here names a tracker or a forge: which key applies follows from what
// the resolver supplied.
func TestAFreshWorktreeOpensOnWhatTheValuesSay(t *testing.T) {
	tests := []struct {
		name string
		vals worktree.Values
		want []string
	}{
		{
			"a ticket some tracker resolved",
			worktree.Values{"Name": "one", "Dir": "/wt", "ID": "one", "Title": "A ticket"},
			[]string{"claude", "--permission-mode", "auto", "--name=one: A ticket", "/start one"},
		},
		{
			"a pull request some forge resolved",
			worktree.Values{"Name": "pr-7", "Dir": "/wt", "Number": "7"},
			[]string{"claude", "--name=PR #7"},
		},
		{
			"a place whose values say neither",
			worktree.Values{"Name": "scratch", "Dir": "/wt"},
			[]string{"claude", "--permission-mode", "auto", "--name=scratch"},
		},
		{
			// A system this action has never heard of reaches the right session by
			// supplying the right values.
			"a tracker of another name entirely",
			worktree.Values{"Name": "j-9", "Dir": "/wt", "ID": "j-9", "Title": "Someone else's ticket"},
			[]string{"claude", "--permission-mode", "auto", "--name=j-9: Someone else's ticket", "/start j-9"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A worktree just created carries no conversation, so an agent consulted here
			// fails the case rather than answering it.
			o := stub(nil, errors.New("a fresh worktree carries none"))
			tree := worktree.Tree{Place: worktree.Place{Name: tt.vals["Name"]}, Path: "/wt", Created: true}

			got, err := o.Open(tree, tt.vals)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			if !slices.Equal(got.Run, tt.want) {
				t.Errorf("Open runs %q; want %q", got.Run, tt.want)
			}
			if got.Dir != "/wt" {
				t.Errorf("Open hands over in %q; want the worktree", got.Dir)
			}
		})
	}
}

// A worktree already there opens on what it already carries, and no session id
// is ever asked of a person: the one conversation is named outright, and several
// leave the name empty, dropping the element that placed it and reaching the
// agent's own list.
func TestAWorktreeAlreadyThereReturnsToWhatItCarries(t *testing.T) {
	tests := []struct {
		name string
		has  []string
		want []string
	}{
		{"none starts one", nil, []string{"claude", "--permission-mode", "auto", "--name=one"}},
		{"one is returned to", []string{"s1"}, []string{"claude", "--resume", "s1"}},
		{"several reach the list", []string{"s1", "s2"}, []string{"claude", "--resume"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := stub(tt.has, nil)
			tree := worktree.Tree{Place: worktree.Place{Name: "one", ID: "one", Label: "A ticket"}, Path: "/wt"}
			vals := worktree.Values{"Name": "one", "Dir": "/wt", "ID": "one", "Title": "A ticket"}

			got, err := o.Open(tree, vals)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			if !slices.Equal(got.Run, tt.want) {
				t.Errorf("Open runs %q; want %q", got.Run, tt.want)
			}
		})
	}
}

// An agent that cannot say what a worktree carries is not asked to guess.
func TestAnUnreadableTranscriptStoreRefuses(t *testing.T) {
	o := stub(nil, errors.New("no transcript store"))
	tree := worktree.Tree{Place: worktree.Place{Name: "one"}, Path: "/wt"}

	if _, err := o.Open(tree, worktree.Values{"Name": "one", "Dir": "/wt"}); err == nil {
		t.Error("Open with an unreadable store: want an error")
	}
}

// The configured commands are what a worktree is handed to, and the values are
// only what their templates place.
func TestTheConfiguredCommandsAreWhatRuns(t *testing.T) {
	o := New(configured(t, "[claude]\n"+
		"start-ticket = [\"agent\", \"--ticket={{.ID}}\", \"{{.Title}}\"]\n"+
		"start-pull-request = [\"agent\", \"--pr={{.Number}}\"]\n"+
		"start-session = [\"agent\", \"--fresh={{.Name}}\"]\n"+
		"resume-session = [\"agent\", \"--resume\", \"{{.Session}}\"]\n"))
	o.carried = fixed{has: []string{"s1"}}
	fresh := worktree.Tree{Place: worktree.Place{Name: "one"}, Path: "/wt", Created: true}
	vals := worktree.Values{"Name": "one", "Dir": "/wt", "ID": "one", "Title": "A ticket"}

	got, err := o.Open(fresh, vals)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if want := []string{"agent", "--ticket=one", "A ticket"}; !slices.Equal(got.Run, want) {
		t.Errorf("a fresh ticket runs %q; want %q", got.Run, want)
	}

	again := fresh
	again.Created = false
	got, err = o.Open(again, vals)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if want := []string{"agent", "--resume", "s1"}; !slices.Equal(got.Run, want) {
		t.Errorf("a worktree already there runs %q; want %q", got.Run, want)
	}
}

// Rendering is free of consequence: the values are the core's, gathered from every
// source and handed to whichever action opens, so the one name this key places is
// put on top of them rather than into them.
func TestOpenLeavesTheValuesItWasHandedAlone(t *testing.T) {
	o := stub([]string{"s1"}, nil)
	tree := worktree.Tree{Place: worktree.Place{Name: "one"}, Path: "/wt"}
	vals := worktree.Values{"Name": "one", "Dir": "/wt"}

	if _, err := o.Open(tree, vals); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if want := (worktree.Values{"Name": "one", "Dir": "/wt"}); !maps.Equal(vals, want) {
		t.Errorf("Open left the values as %v; want %v, the ones it was handed", vals, want)
	}
}

// A key falls through to the next only where a value nothing supplied says this
// worktree is not the one that key is for. A key that names no command to run
// whatever the worktree is has been misconfigured, and a fresh ticket opening on a
// bare session instead would be the settings quietly overruled.
func TestAMisconfiguredKeyFailsRatherThanFallingThrough(t *testing.T) {
	// Every value the key has is supplied, and the command still names nothing to
	// run: the one element it has renders empty for a ticket carrying no title.
	o := New(configured(t, "[claude]\nstart-ticket = [\"{{.Title}}\"]\n"))
	o.carried = fixed{err: errors.New("a fresh worktree carries none")}
	fresh := worktree.Tree{Place: worktree.Place{Name: "one"}, Path: "/wt", Created: true}
	vals := worktree.Values{"Name": "one", "Dir": "/wt", "ID": "one", "Title": ""}

	got, err := o.Open(fresh, vals)
	if err == nil {
		t.Fatalf("Open runs %q; want the key that will not render refused", got.Run)
	}
	if !strings.Contains(err.Error(), "claude.start-ticket") {
		t.Errorf("Open = %v; want the refusal to name the key behind it", err)
	}
}

// The action is named for what it speaks to, and the [action] keys, the flag and
// the table of commands all spell that name.
func TestNameIsTheActionKeysSpelling(t *testing.T) {
	if Name != "claude" {
		t.Errorf("the action goes by %q; want the name the [action] keys and the flag spell", Name)
	}
	if Name != string(config.ActionClaude) {
		t.Errorf("the action goes by %q; the [action] keys spell it %q", Name, config.ActionClaude)
	}
}
