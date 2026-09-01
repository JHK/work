package config

import (
	"fmt"
	"iter"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
)

// Dump renders the configuration as TOML. What it prints is what work would load
// back.
func (c Config) Dump() string {
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
		fmt.Fprintf(&b, "%s = %s\n", leaf, spelled(k.value))
	}
	return b.String()
}

// key is one setting: the whole name, as a settings file spells it, and the
// value work resolved.
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
		{githubBranchKey, c.Github.pattern().tmpl.text},
		{beadsBranchKey, c.Beads.pattern().tmpl.text},
		{onCreationKey, c.Claude.OnCreation()},
		{commandKey, block(c.Claude.Command().text)},
	}
}

// block is a value a settings file writes as a multiline literal string.
type block string

// spelled is one setting as a settings file spells it, quoted by the package that
// reads it back.
func spelled(v any) string {
	if b, ok := v.(block); ok {
		if s := string(b); fitsLiteral(s) {
			// TOML drops the newline after the opening quotes, so the text reads back as written.
			return "'''\n" + s + "'''"
		}
		v = string(b)
	}
	out, err := toml.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("config: dumping %v: %v", v, err))
	}
	return string(out)
}

// TOML escapes nothing inside a literal string, so it holds neither the quotes
// that close it nor a control character other than a tab.
func fitsLiteral(s string) bool {
	return !strings.Contains(s, "'''") &&
		!strings.ContainsFunc(s, func(r rune) bool { return r != '\t' && r != '\n' && unicode.IsControl(r) })
}

// Settings is every key work read and the value it resolved to, in the order
// the reference documents them. [Config.Dump] is the same settings as a file
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
