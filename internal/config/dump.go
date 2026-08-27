package config

import (
	"cmp"
	"fmt"
	"iter"
	"strings"

	"github.com/BurntSushi/toml"
)

// compiledIn is where a key no file set came from.
const compiledIn = "the compiled-in default"

// Dump renders the configuration as TOML, each key under a comment naming where
// it came from. What it prints is what work would load back.
func Dump() (string, error) {
	c, from, err := read()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	table := ""
	for _, k := range c.keys() {
		// A key with no table of its own stands above the first header, where TOML
		// reads it as the top-level key it is.
		name, leaf, under := strings.Cut(k.name, ".")
		if !under {
			name, leaf = "", name
		}
		if name != table {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			if name != "" {
				fmt.Fprintf(&b, "[%s]\n", name)
			}
			table = name
		}
		fmt.Fprintf(&b, "# %s\n%s = %s\n", cmp.Or(from[k.name], compiledIn), leaf, value(k.value))
	}
	return b.String(), nil
}

// key is one setting: the whole name, which is both how a settings file spells
// it and how its source is keyed, and the value work resolved.
type key struct {
	name  string
	value any
}

// keys is every setting work reads, in the order the reference documents them,
// each holding the value work resolved rather than what a file wrote. A key
// sitting in no table comes first, which is where TOML reads one.
func (c Config) keys() []key {
	return []key{
		{systemsKey, c.Systems},
		{dirKey, c.Worktree.Dir()},
		{ticketKey, c.Branch.ticket().tmpl.text},
		{pullRequestKey, c.Branch.pullRequest().tmpl.text},
		{onCreationKey, c.Claude.OnCreation()},
		{startTicketKey, argv(c.Claude.StartTicket())},
		{startPullRequestKey, argv(c.Claude.StartPullRequest())},
		{startSessionKey, argv(c.Claude.StartSession())},
	}
}

// argv is a command as it was written, element by element, rather than as it
// renders.
func argv(c Command) []string {
	out := make([]string, len(c.parts))
	for i, p := range c.parts {
		out[i] = p.text
	}
	return out
}

// value is one setting as a settings file spells it, quoted by the package that
// reads it back.
func value(v any) string {
	out, err := toml.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("config: dumping %v: %v", v, err))
	}
	return string(out)
}

// Settings is every key work read and the value it resolved to, in the order
// the reference documents them. [Dump] is the same settings as a file
// spells them.
func (c Config) Settings() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for _, k := range c.keys() {
			if !yield(k.name, k.value) {
				return
			}
		}
	}
}
