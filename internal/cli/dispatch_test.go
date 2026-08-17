package cli

import (
	"slices"
	"testing"
)

// The first word decides: a verb, a root flag and cobra's completion request
// answer for themselves, and everything else is the bare form, which is go.
func TestDispatch(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"nothing at all", nil, []string{"go"}},
		{"an identifier", []string{"bd-1"}, []string{"go", "bd-1"}},
		{"a flag in the bare position", []string{"--shell", "bd-1"}, []string{"go", "--shell", "bd-1"}},
		// Only the root's own two flags stop here; the rest are go's, to act on or
		// to refuse, whichever verb declares them.
		{"an unknown flag", []string{"--turbo"}, []string{"go", "--turbo"}},
		{"another verb's flag", []string{"--force", "scratch"}, []string{"go", "--force", "scratch"}},

		{"go", []string{"go", "bd-1"}, []string{"go", "bd-1"}},
		{"switch", []string{"switch", "bd-1"}, []string{"switch", "bd-1"}},
		{"add", []string{"add", "scratch"}, []string{"add", "scratch"}},
		{"remove", []string{"remove", "--force"}, []string{"remove", "--force"}},
		{"move", []string{"move", "scratch", "settled"}, []string{"move", "scratch", "settled"}},
		{"list", []string{"list"}, []string{"list"}},
		{"init", []string{"init", "fish"}, []string{"init", "fish"}},
		{"help", []string{"help", "add"}, []string{"help", "add"}},

		{"the version", []string{"--version"}, []string{"--version"}},
		{"the version shorthand", []string{"-v"}, []string{"-v"}},
		{"the help flag", []string{"--help"}, []string{"--help"}},
		{"the help shorthand", []string{"-h"}, []string{"-h"}},
		{"a spelled-out value", []string{"--version=true"}, []string{"--version=true"}},

		// Every tab press is one of these, so neither may be dispatched.
		{"the completion request", []string{"__complete", "sw"}, []string{"__complete", "sw"}},
		{"the completion request without descriptions", []string{"__completeNoDesc", "sw"}, []string{"__completeNoDesc", "sw"}},
	}
	root := command(stubVersion, wired(), front{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dispatch(root, tt.args); !slices.Equal(got, tt.want) {
				t.Errorf("dispatch(%q) = %q; want %q", tt.args, got, tt.want)
			}
		})
	}
}
