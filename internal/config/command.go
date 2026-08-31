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
	if err := c.Command().validate(); err != nil {
		return commandKey, err
	}
	return "", nil
}

const defaultCommand = `{{if .Subject}}
claude
{{if eq .Source "beads"}}--permission-mode=auto{{end}}
--name={{.Subject}}
{{if eq .Source "beads"}}/start {{.ID}}{{end}}
{{end}}
`

var defaultClaude = Claude{CommandLine: mustCommand(defaultCommand)}

// Command is a whole command line: one [text/template] rendered over
// [worktree.ValueNames], then read a line at a time.
type Command struct{ tmpl }

var valueNames = worktree.ValueNames()

// A name outside [worktree.ValueNames] fails the render rather than rendering
// empty. Every one of them is supplied, empty where nothing behind the worktree
// has one.
func knownValues(vals worktree.Values) map[string]any {
	d := make(map[string]any, len(valueNames))
	for _, name := range valueNames {
		d[name] = vals[name]
	}
	return d
}

var valueList = "{{." + strings.Join(valueNames, "}}, {{.") + "}}"

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
			d[worktree.SourceValue] = source
			out = append(out, d)
		}
	}
	return out
}

func everyValue(value string) map[string]any {
	d := make(map[string]any, len(valueNames))
	for _, name := range valueNames {
		d[name] = value
	}
	return d
}

// UnmarshalTOML reads the block a settings file writes the command as. validate
// judges the value later.
func (c *Command) UnmarshalTOML(v any) error {
	text, ok := v.(string)
	if !ok {
		return errors.New("is not text")
	}
	parsed, err := parseCommand(text)
	*c = parsed
	return err
}

// commandFuncs are the filters a command may pipe a value through, named as
// sprig names them.
var commandFuncs = template.FuncMap{"squote": shellQuote}

// shellQuote is the value as one word of a POSIX shell. Single quotes hold every
// other character, and a single quote closes them, escapes and reopens.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func parseCommand(text string) (Command, error) {
	t, err := parseTmpl("command", text, commandFuncs)
	return Command{tmpl: t}, err
}

func mustCommand(text string) Command {
	c, err := parseCommand(text)
	if err == nil {
		err = c.validate()
	}
	if err != nil {
		panic(fmt.Sprintf("config: default command %q %v", text, err))
	}
	return c
}

// validate renders the block over every value both set and unset, so one that
// cannot render, or that names no command for any of them, is refused at load.
func (c Command) validate() error {
	namesCommand := false
	for _, d := range probes() {
		out, err := c.execute(d)
		if err != nil {
			return fmt.Errorf("%w; the values here are %s", err, valueList)
		}
		namesCommand = namesCommand || len(arguments(out)) > 0
	}
	if !namesCommand {
		return errors.New("names no command to run")
	}
	return nil
}

// or is the command itself, or def where no file named one.
func (c Command) or(def Command) Command {
	if c.t == nil {
		return def
	}
	return c
}

// Render is the block rendered over a worktree's values and cut into argv, empty
// where the block rendered nothing.
func (c Command) Render(vals worktree.Values) ([]string, error) {
	out, err := c.execute(knownValues(vals))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", commandKey, err)
	}
	return arguments(out), nil
}

// arguments are the non-blank lines of a rendered block, one each, trimmed.
func arguments(rendered string) []string {
	var argv []string
	for line := range strings.SplitSeq(rendered, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			argv = append(argv, line)
		}
	}
	return argv
}
