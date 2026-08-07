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
		{"a flag value is not an identifier", []string{"bd-1", "--model", "opus"}, options{model: "opus"}, "bd-1"},
		{"joined value", []string{"--effort=high", "bd-1"}, options{effort: "high"}, "bd-1"},
		{"flags before the identifier", []string{"--shell", "bd-1"}, options{shell: true}, "bd-1"},
		{"an editor with no identifier", []string{"--editor"}, options{editor: true}, ""},
		{"an editor on a named target", []string{"--editor", "bd-1"}, options{editor: true}, "bd-1"},
		// --no-claim says nothing about which command opens, so it combines with each.
		{"no claim", []string{"--no-claim", "bd-1"}, options{noClaim: true}, "bd-1"},
		{"a shell that does not claim", []string{"--shell", "--no-claim", "bd-1"}, options{shell: true, noClaim: true}, "bd-1"},
		{"an editor that does not claim", []string{"--editor", "--no-claim", "bd-1"}, options{editor: true, noClaim: true}, "bd-1"},
		{"no claim from the picker", []string{"--no-claim"}, options{noClaim: true}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got options
			var target string
			err := run(tt.args, func(o options, id string) error {
				got, target = o, id
				return nil
			})
			if err != nil {
				t.Fatalf("Execute(%q): %v", tt.args, err)
			}
			if got != tt.want || target != tt.target {
				t.Errorf("Execute(%q) = %+v, %q; want %+v, %q", tt.args, got, target, tt.want, tt.target)
			}
		})
	}
}

func TestVersionFlag(t *testing.T) {
	var out strings.Builder
	err := runTo([]string{"--version"}, &out, func(options, string) error {
		t.Error("entered a worktree despite --version")
		return nil
	})
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
		{"a shell and an editor at once", []string{"bd-1", "--shell", "--editor"}},
		{"two identifiers", []string{"bd-1", "bd-2"}},
		{"unknown flag", []string{"bd-1", "--turbo"}},
		{"init without a shell", []string{"init"}},
		{"init with a shell work does not print", []string{"init", "bash"}},
		{"init with two shells", []string{"init", "fish", "zsh"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(tt.args, func(options, string) error {
				t.Error("ran despite invalid flags")
				return nil
			})
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

const stubVersion = "v0.0.0-test"

func run(args []string, f func(options, string) error) error {
	return runTo(args, io.Discard, f)
}

func runTo(args []string, out io.Writer, f func(options, string) error) error {
	cmd := command(stubVersion, f, stub(nil, nil))
	cmd.SetArgs(args)
	cmd.SetOut(out)
	cmd.SetErr(io.Discard)
	return cmd.Execute()
}
