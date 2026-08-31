package cli

import (
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// A verb and a root flag answer for themselves, and everything else is the bare
// form, which is go. These are the words that reach nothing else.
func TestTheFirstWordDecides(t *testing.T) {
	// The row index is fzf's answer, so a run that reached the picker took the row.
	s := repository(t, testenv.Stub{Name: "fzf", Says: "0\tscratch\n"})
	path := s.opened("scratch")

	tests := []struct {
		name string
		args []string
		want result
	}{
		{"nothing at all", nil, result{Answered: path, Asked: []string{putUp}}},
		// A flag the root does not declare holds no position either: the words reach
		// go, which refuses the flag it does not carry.
		{"a flag in the bare position", []string{"--force", "scratch"}, result{Code: 1}},
		// The root passes --log-level down to every verb, so it says how the bare form
		// runs rather than standing for a verb of its own.
		{"the level spelled out", []string{"--log-level=info", "scratch"}, result{Answered: path}},
		// The word a handed-down flag takes its value from is no position either, so
		// the one behind it is still the first.
		{"the level and the word it takes", []string{"--log-level", "info", "scratch"}, result{Answered: path}},
		// The bare form with nothing behind the level: the picker stands in, and info
		// is read as the level rather than as a name.
		{"the level and nothing else", []string{"--log-level", "info"}, result{Answered: path, Asked: []string{putUp}}},
		// A flag spelling its own value takes no word after it: --version is the
		// root's whichever way it is written.
		{"a spelled-out value", []string{"--version=true"}, result{Out: versionLine}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := s.run(tt.args...)

			r.came(t, tt.want, saidApart)
		})
	}
}

// The help names the verbs rather than running one of them, wherever it is asked
// for.
func TestTheHelpIsTheRootsOwn(t *testing.T) {
	s := repository(t)

	for _, args := range [][]string{{"--help"}, {"-h"}, {"help"}} {
		r := s.run(args...)

		r.came(t, result{}, besides("Out"))
		for _, verb := range []string{"go", "switch", "add", "carry", "remove", "move", "list", "init", "config"} {
			require.Contains(t, r.Out, verb, "work %v named no %s", args, verb)
		}
	}
}

func TestCompletionIsAWorktreeNameLikeAnyOther(t *testing.T) {
	s := repository(t)
	path := s.opened("completion")

	r := s.run("completion")

	r.came(t, result{Answered: path})
}
