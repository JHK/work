package config

import (
	"reflect"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// spelling is the key a field is written as: its toml tag, or its name
// lowercased where it has none, which is what the decoder matches.
func spelling(f reflect.StructField) string {
	if tag, ok := f.Tag.Lookup("toml"); ok {
		return tag
	}
	return strings.ToLower(f.Name)
}

// The settings a dump names are the settings the tables declare, under the
// spelling a settings file loads back. No command reaches [Config.keys].
func TestEveryDeclaredSettingIsDumped(t *testing.T) {
	var want []string
	for table := range reflect.TypeFor[Config]().Fields() {
		// A field that is no table is a key of its own, standing above them all.
		if table.Type.Kind() != reflect.Struct {
			want = append(want, spelling(table))
			continue
		}
		for field := range table.Type.Fields() {
			want = append(want, strings.ToLower(table.Name)+"."+spelling(field))
		}
	}

	var got []string
	for _, k := range Default().keys() {
		got = append(got, k.name)
	}
	testenv.Equal(t, want, got, "the dump and the tables name different settings")
}

// A Config that never reached Load still names a directory, a branch and a
// command. No command reaches a zero Config; only the compiler hands one over.
func TestTheZeroConfigNamesWhatWorkNeeds(t *testing.T) {
	var c Config

	require.Equal(t, defaultDirectory, c.Worktree.Dir(), "the zero Config puts a worktree in the repository root")
	require.Equal(t, "bd-42", c.Branch.Ticket("bd-42", ""), "the zero Config names no ticket branch")
	require.Equal(t, "pr-7", c.Branch.PullRequest("7"), "the zero Config names no pull request branch")

	got, err := c.Claude.Command().Render(worktree.Values{
		worktree.SourceValue:  "github",
		worktree.IDValue:      "7",
		worktree.NameValue:    "pr-7",
		worktree.DirValue:     "/wt",
		worktree.SubjectValue: "PR #7",
	})
	require.NoError(t, err)
	testenv.Equal(t, []string{"claude", "--name=PR #7"}, got, "the zero Config names no command")
}
