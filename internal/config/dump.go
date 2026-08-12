package config

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

// compiledIn is the layer a key no file set came from.
const compiledIn = "the compiled-in default"

// Dump renders the merged configuration as TOML, each key under a comment
// naming the layer that set it, so what it prints is also what work would load
// back in the repository it was read for.
func Dump(repo string) (string, error) {
	c, from, err := merge(repo)
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
		fmt.Fprintf(&b, "# %s\n%s = %s\n", layer(from[k.name], repo), leaf, k.value)
	}
	return b.String(), nil
}

// key is one setting of a dump: the whole dotted name, which is both how a
// settings file spells it and how the layers are keyed, and its value as TOML.
type key struct {
	name  string
	value string
}

// keys is every setting work reads, in the order the reference documents them,
// each holding the value the merge settled on rather than what a file wrote.
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
		{shellKey, value(argv(c.Open.Shell()))},
	}
}

// layer names where a value came from. The repository's file is named as it is
// written rather than by its path, being always at the root of the checkout the
// dump was taken in.
func layer(path, repo string) string {
	switch path {
	case "":
		return compiledIn
	case repoSettings(repo):
		return RepoFile
	}
	return path
}

// argv is a command as it was written, element by element, rather than as it
// renders, a rendering needing a target.
func argv(c Command) []string {
	out := make([]string, len(c.parts))
	for i, p := range c.parts {
		out[i] = p.text
	}
	return out
}

// value is one setting as a settings file spells it. The quoting is the package
// that reads it back doing it, so printing and reading cannot drift apart.
func value(v any) string {
	out, err := toml.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("config: dumping %v: %v", v, err))
	}
	return string(out)
}
