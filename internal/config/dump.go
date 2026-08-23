package config

import (
	"cmp"
	"fmt"
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
		name, leaf, _ := strings.Cut(k.name, ".")
		if name != table {
			if table != "" {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "[%s]\n", name)
			table = name
		}
		fmt.Fprintf(&b, "# %s\n%s = %s\n", cmp.Or(from[k.name], compiledIn), leaf, k.value)
	}
	return b.String(), nil
}

// key is one setting of a dump: the whole dotted name, which is both how a
// settings file spells it and how its source is keyed, and its value as TOML.
type key struct {
	name  string
	value string
}

// keys is every setting work reads, in the order the reference documents them,
// each holding the value work resolved rather than what a file wrote.
func (c Config) keys() []key {
	return []key{
		{dirKey, value(c.Worktree.Dir())},
		{ticketKey, value(c.Branch.ticket().tmpl.text)},
		{pullRequestKey, value(c.Branch.pullRequest().tmpl.text)},
		{createKey, value(string(c.Action.Create()))},
		{enterKey, value(string(c.Action.Enter()))},
		{SystemKey(githubSystem), value(c.Github.Enabled)},
		{SystemKey(beadsSystem), value(c.Beads.Enabled)},
		{SystemKey(miseSystem), value(c.Mise.Enabled)},
		{SystemKey(claudeSystem), value(c.Claude.Enabled)},
		{startTicketKey, value(argv(c.Claude.StartTicket()))},
		{startPullRequestKey, value(argv(c.Claude.StartPullRequest()))},
		{startSessionKey, value(argv(c.Claude.StartSession()))},
		{resumeSessionKey, value(argv(c.Claude.ResumeSession()))},
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
