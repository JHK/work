package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/JHK/work-cli/internal/worktree"
)

// Claude is the agent's table: which verbs open a session on what they create,
// and the commands it runs, read whether or not the systems list names the
// agent. Each command falls back to defaultClaude on its own, so an unset one is
// the compiled-in command and a config pointing the action at another binary
// sets every one of them.
type Claude struct {
	OnCreationVerbs         []string `toml:"on-creation"`
	StartTicketCommand      Command  `toml:"start-ticket"`
	StartPullRequestCommand Command  `toml:"start-pull-request"`
}

const (
	onCreationKey       = "claude.on-creation"
	startTicketKey      = "claude.start-ticket"
	startPullRequestKey = "claude.start-pull-request"
)

// StartTicket is the command a fresh ticket worktree opens on.
func (c Claude) StartTicket() Command {
	return c.StartTicketCommand.or(defaultClaude.StartTicketCommand)
}

// StartPullRequest is the command a fresh pull request worktree opens on.
func (c Claude) StartPullRequest() Command {
	return c.StartPullRequestCommand.or(defaultClaude.StartPullRequestCommand)
}

// validate judges each command against the values its key renders with, and
// names the key work cannot use the value of.
func (c *Claude) validate() (string, error) {
	if err := c.validateOnCreation(); err != nil {
		return onCreationKey, err
	}
	if err := c.StartTicketCommand.bind(startTicketValues); err != nil {
		return startTicketKey, err
	}
	if err := c.StartPullRequestCommand.bind(startPullRequestValues); err != nil {
		return startPullRequestKey, err
	}
	return "", nil
}

var defaultClaude = Claude{
	StartTicketCommand: mustCommand(startTicketValues,
		"claude", "--permission-mode", "auto",
		"--name={{.ID}}: {{.Title}}",
		"/start {{.ID}}",
	),
	StartPullRequestCommand: mustCommand(startPullRequestValues, "claude", "--name=PR #{{.Number}}"),
}

// Command is a whole command line: one [text/template] per argv element,
// rendered with the values its key has. Which values those are is settled by
// bind, once the key the command was read from says.
type Command struct {
	parts  []tmpl
	values keyValues
}

// keyValues are the value names one key's command may place, and the key it
// is named by.
type keyValues struct {
	key   string
	names []string
}

// common are the values every command has; a key's own follow them.
var common = []string{"Name", "Dir"}

var (
	startTicketValues      = keyValues{startTicketKey, slices.Concat(common, []string{"ID", "Title"})}
	startPullRequestValues = keyValues{startPullRequestKey, slices.Concat(common, []string{"Number"})}
)

// ErrUnsupplied is a value the key places that nothing in the wiring supplied. It
// is the one refusal a caller choosing between keys may pass over: it says this
// worktree is not the one that key is for.
var ErrUnsupplied = errors.New("nothing here supplies")

// data is what one render is given: the values the key has and no others, so a
// template naming another fails rather than quietly rendering nothing. A name the
// key has that nothing supplied is refused here as [ErrUnsupplied].
func (v keyValues) data(vals worktree.Values) (map[string]any, error) {
	data := make(map[string]any, len(v.names))
	for _, name := range v.names {
		value, supplied := vals[name]
		if !supplied {
			return nil, fmt.Errorf("%s: %w {{.%s}}", v.key, ErrUnsupplied, name)
		}
		data[name] = value
	}
	return data, nil
}

// mark is the filled arm of a binding probe: the shortest any real value is, so a
// command rendering with it renders with anything.
const mark = "x"

// fill is every value the key has, set alike: binding probes an empty arm and a
// filled one.
func (v keyValues) fill(value string) worktree.Values {
	vals := make(worktree.Values, len(v.names))
	for _, name := range v.names {
		vals[name] = value
	}
	return vals
}

// list is how the values read in a refusal.
func (v keyValues) list() string {
	out := make([]string, len(v.names))
	for i, name := range v.names {
		out[i] = "{{." + name + "}}"
	}
	return strings.Join(out, ", ")
}

// UnmarshalTOML reads a command out of a settings file, where it is written as
// an array of strings, one per argv element. bind judges the values later.
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

func parseCommand(texts []string) (Command, error) {
	// Non-nil even when empty, so that a key set to no command reads as set.
	parts := make([]tmpl, 0, len(texts))
	for _, text := range texts {
		t, err := parseTmpl("command", text)
		if err != nil {
			return Command{}, err
		}
		parts = append(parts, t)
	}
	return Command{parts: parts}, nil
}

// mustCommand binds a compiled-in default, which cannot be at fault.
func mustCommand(v keyValues, texts ...string) Command {
	c, err := parseCommand(texts)
	if err == nil {
		err = c.bind(v)
	}
	if err != nil {
		panic(fmt.Sprintf("config: default command %q %v", texts, err))
	}
	return c
}

// bind ties the command to the values its key renders with, or reports why it
// cannot name a command. Every value is rendered both set and unset, so a
// template that cannot render at all is refused at load rather than at the
// handoff.
func (c *Command) bind(v keyValues) error {
	if len(c.parts) == 0 {
		return errors.New("names no command to run")
	}
	for _, probe := range []worktree.Values{v.fill(""), v.fill(mark)} {
		data, err := v.data(probe)
		if err != nil {
			return err
		}
		for _, p := range c.parts {
			if _, err := p.execute(data); err != nil {
				return fmt.Errorf("%w; the values here are %s", err, v.list())
			}
		}
	}
	c.values = v
	return nil
}

// or is the command itself, or def where no file named one.
func (c Command) or(def Command) Command {
	if c.parts == nil {
		return def
	}
	return c
}

// Render builds the argv, dropping every element that renders to nothing, so an
// optional flag is one element rather than a pair.
func (c Command) Render(vals worktree.Values) ([]string, error) {
	data, err := c.values.data(vals)
	if err != nil {
		return nil, err
	}
	argv := make([]string, 0, len(c.parts))
	for i, p := range c.parts {
		s, err := p.execute(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.values.key, err)
		}
		// A first element rendering to nothing leaves no command at all; a later one
		// is the optional flag it was written as.
		if s == "" {
			if i == 0 {
				return nil, fmt.Errorf("%s: %q named nothing, leaving no command to run", c.values.key, p.text)
			}
			continue
		}
		argv = append(argv, s)
	}
	return argv, nil
}
