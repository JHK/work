package config

import (
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/JHK/work-cli/internal/worktree"
)

// Claude is the agent's table: which verbs open a session on what they create,
// and the command it runs, read whether or not the systems list names the agent.
type Claude struct {
	OnCreationVerbs []string `toml:"on-creation"`
	CommandLine     Command  `toml:"command"`
}

const (
	onCreationKey = "claude.on-creation"
	commandKey    = "claude.command"
)

// Command is the command a fresh worktree opens on.
func (c Claude) Command() Command { return c.CommandLine.or(defaultClaude.CommandLine) }

// validate names the key work cannot use the value of.
func (c *Claude) validate() (string, error) {
	if err := c.validateOnCreation(); err != nil {
		return onCreationKey, err
	}
	if err := c.CommandLine.validate(); err != nil {
		return commandKey, err
	}
	return "", nil
}

var defaultClaude = Claude{
	CommandLine: mustCommand(
		`{{if .Subject}}claude{{end}}`,
		`{{if eq .Source "beads"}}--permission-mode=auto{{end}}`,
		`--name={{.Subject}}`,
		`{{if eq .Source "beads"}}/start {{.ID}}{{end}}`,
	),
}

// Command is a whole command line: one [text/template] per argv element,
// rendered with [commandValues].
type Command struct {
	parts []tmpl
}

// commandValues are always supplied, empty where nothing behind the worktree
// has one.
var commandValues = []string{"Source", "ID", "Title", "Name", "Dir", "Subject"}

// A name outside [commandValues] fails the render rather than rendering empty.
func knownValues(vals worktree.Values) map[string]any {
	d := make(map[string]any, len(commandValues))
	for _, name := range commandValues {
		d[name] = vals[name]
	}
	return d
}

var valueList = "{{." + strings.Join(commandValues, "}}, {{.") + "}}"

// mark is the shortest any real value is, so a command rendering with it renders
// with anything.
const mark = "x"

// Crossing the sources in is what reaches an arm naming one, which would
// otherwise render for the first time at the handoff.
func probes() []map[string]any {
	sources := sourceNames()
	out := make([]map[string]any, 0, 2*len(sources))
	for _, value := range []string{"", mark} {
		for _, source := range sources {
			d := everyValue(value)
			d["Source"] = source
			out = append(out, d)
		}
	}
	return out
}

func everyValue(value string) map[string]any {
	d := make(map[string]any, len(commandValues))
	for _, name := range commandValues {
		d[name] = value
	}
	return d
}

// UnmarshalTOML reads a command out of a settings file, where it is written as
// an array of strings, one per argv element. validate judges the values later.
func (c *Command) UnmarshalTOML(v any) error {
	list, ok := v.([]any)
	if !ok {
		return errors.New("is not a list of command line arguments")
	}
	texts := make([]string, len(list))
	for i, e := range list {
		text, ok := e.(string)
		if !ok {
			return errors.New("is not a list of command line arguments")
		}
		texts[i] = text
	}
	q, err := parseCommand(texts)
	*c = q
	return err
}

// commandFuncs are the filters a command element may pipe a value through, named
// as sprig names them.
var commandFuncs = template.FuncMap{"squote": shellQuote}

// shellQuote is the value as one word of a POSIX shell, for an element that is
// itself a shell script. Single quotes hold every other character, and a single
// quote closes them, escapes and reopens.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func parseCommand(texts []string) (Command, error) {
	// Non-nil even when empty, so that a key set to no command reads as set.
	parts := make([]tmpl, 0, len(texts))
	for _, text := range texts {
		t, err := parseTmpl("command", text, commandFuncs)
		if err != nil {
			return Command{}, err
		}
		parts = append(parts, t)
	}
	return Command{parts: parts}, nil
}

// mustCommand panics: a compiled-in default cannot be at fault.
func mustCommand(texts ...string) Command {
	c, err := parseCommand(texts)
	if err == nil {
		err = c.validate()
	}
	if err != nil {
		panic(fmt.Sprintf("config: default command %q %v", texts, err))
	}
	return c
}

// validate renders every value both set and unset, so a template that cannot
// render at all is refused at load rather than at the handoff.
func (c Command) validate() error {
	if len(c.parts) == 0 {
		return errors.New("names no command to run")
	}
	for _, d := range probes() {
		for _, p := range c.parts {
			if _, err := p.execute(d); err != nil {
				return fmt.Errorf("%w; the values here are %s", err, valueList)
			}
		}
	}
	return nil
}

// or is the command itself, or def where no file named one.
func (c Command) or(def Command) Command {
	if c.parts == nil {
		return def
	}
	return c
}

// Render drops every element that renders to nothing. A first element rendering
// to nothing leaves no command, and the empty argv is the worktree itself.
func (c Command) Render(vals worktree.Values) ([]string, error) {
	d := knownValues(vals)
	argv := make([]string, 0, len(c.parts))
	for i, p := range c.parts {
		s, err := p.execute(d)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", commandKey, err)
		}
		if s == "" {
			if i == 0 {
				return nil, nil
			}
			continue
		}
		argv = append(argv, s)
	}
	return argv, nil
}
