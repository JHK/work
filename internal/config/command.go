package config

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/JHK/work-cli/internal/worktree"
)

// Claude is the agent's table: whether the action runs at all, and the commands
// it runs. Each command falls back to defaultClaude on its own, so a config
// pointing the action at another binary sets every one of them: what starts a
// conversation is what returns to it. An unset command is the compiled-in one,
// so a Config that never reached Load still names something to run; whether the
// action runs at all is [Shipped]'s to say, this table holding only the commands
// it would run.
type Claude struct {
	Enabled                 bool
	StartTicketCommand      Command `toml:"start-ticket"`
	StartPullRequestCommand Command `toml:"start-pull-request"`
	StartSessionCommand     Command `toml:"start-session"`
	ResumeSessionCommand    Command `toml:"resume-session"`
}

const (
	startTicketKey      = "claude.start-ticket"
	startPullRequestKey = "claude.start-pull-request"
	startSessionKey     = "claude.start-session"
	resumeSessionKey    = "claude.resume-session"
)

// StartTicket is the command a fresh ticket worktree opens on.
func (c Claude) StartTicket() Command {
	return c.StartTicketCommand.or(defaultClaude.StartTicketCommand)
}

// StartPullRequest is the command a fresh pull request worktree opens on.
func (c Claude) StartPullRequest() Command {
	return c.StartPullRequestCommand.or(defaultClaude.StartPullRequestCommand)
}

// StartSession is the command a worktree that carries no conversation opens on.
func (c Claude) StartSession() Command {
	return c.StartSessionCommand.or(defaultClaude.StartSessionCommand)
}

// ResumeSession is the command that returns to the conversation a worktree
// carries. An empty Session drops the element that placed it, so the one
// conversation is returned to outright and several reach claude's own list.
func (c Claude) ResumeSession() Command {
	return c.ResumeSessionCommand.or(defaultClaude.ResumeSessionCommand)
}

// validate judges each command against the values its key renders with, and
// names the key work cannot use the value of.
func (c *Claude) validate() (string, error) {
	if err := c.StartTicketCommand.bind(startTicketValues); err != nil {
		return startTicketKey, err
	}
	if err := c.StartPullRequestCommand.bind(startPullRequestValues); err != nil {
		return startPullRequestKey, err
	}
	if err := c.StartSessionCommand.bind(startSessionValues); err != nil {
		return startSessionKey, err
	}
	if err := c.ResumeSessionCommand.bind(resumeSessionValues); err != nil {
		return resumeSessionKey, err
	}
	return "", nil
}

// defaultClaude is the session work opened before any of this was a setting,
// less the review prompt a pull request used to get, which is the reviewer's to
// choose.
var defaultClaude = Claude{
	StartTicketCommand: mustCommand(startTicketValues,
		"claude", "--permission-mode", "auto",
		"--name={{.ID}}: {{.Title}}",
		"/start {{.ID}}",
	),
	StartPullRequestCommand: mustCommand(startPullRequestValues, "claude", "--name=PR #{{.Number}}"),
	// Named after the worktree, so a conversation started here is that worktree in
	// every later list.
	StartSessionCommand: mustCommand(startSessionValues,
		"claude", "--permission-mode", "auto", "--name={{.Name}}",
	),
	// No --permission-mode: claude ignores it alongside --resume.
	ResumeSessionCommand: mustCommand(resumeSessionValues,
		"claude", "--resume", "{{.Session}}",
	),
}

// Open is what a worktree is handed over to when no session is started. The
// keys owe each other nothing, so the table is named for the verb rather than
// for what any of them reaches. An unset command is the compiled-in one, as
// [Claude]'s are.
type Open struct {
	ShellCommand  Command `toml:"shell"`
	EditorCommand Command `toml:"editor"`
	DiffCommand   Command `toml:"diff"`
}

const (
	shellKey  = "open.shell"
	editorKey = "open.editor"
	diffKey   = "open.diff"
)

// Shell is the command an existing worktree is entered with, and the one --shell
// hands over to.
func (o Open) Shell() Command { return o.ShellCommand.or(defaultOpen.ShellCommand) }

// Editor is the command --editor hands the worktree to.
func (o Open) Editor() Command { return o.EditorCommand.or(defaultOpen.EditorCommand) }

// Diff is the command --diff hands the worktree to.
func (o Open) Diff() Command { return o.DiffCommand.or(defaultOpen.DiffCommand) }

func (o *Open) validate() (string, error) {
	if err := o.ShellCommand.bind(shellValues); err != nil {
		return shellKey, err
	}
	if err := o.EditorCommand.bind(editorValues); err != nil {
		return editorKey, err
	}
	if err := o.DiffCommand.bind(diffValues); err != nil {
		return diffKey, err
	}
	return "", nil
}

// defaultOpen places what the environment named for the two that read it from
// there, and git for the diff. Whatever the editor makes of the terminal it is
// handed is its own business, so a terminal and a GUI editor are invoked alike.
var defaultOpen = Open{
	ShellCommand:  mustCommand(shellValues, "{{.Shell}}"),
	EditorCommand: mustCommand(editorValues, "{{.Editor}}", "{{.Dir}}"),
	// --merge-base rather than the three-dot form: given one commit, git diffs the
	// merge-base against the working tree, where three dots would diff it against
	// HEAD and leave uncommitted work out.
	DiffCommand: mustCommand(diffValues, "git", "diff", "--merge-base", "{{.Base}}"),
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

// common are the values every command has, whatever it is for; a key's own
// follow them.
var common = []string{"Name", "Dir"}

var (
	startTicketValues      = keyValues{startTicketKey, slices.Concat(common, []string{"ID", "Title"})}
	startPullRequestValues = keyValues{startPullRequestKey, slices.Concat(common, []string{"Number"})}
	startSessionValues     = keyValues{startSessionKey, common}
	resumeSessionValues    = keyValues{resumeSessionKey, slices.Concat(common, []string{"Session"})}
	shellValues            = keyValues{shellKey, slices.Concat(common, []string{"Shell"})}
	editorValues           = keyValues{editorKey, slices.Concat(common, []string{"Editor"})}
	diffValues             = keyValues{diffKey, slices.Concat(common, []string{"Base"})}
)

// ErrUnsupplied is a value the key places that nothing in the wiring supplied,
// which is the one refusal a caller choosing between keys may pass over: what it
// says about the key is that this worktree is not the one that key is for.
// Everything else a render refuses with is a key that cannot name a command
// whatever the worktree is.
var ErrUnsupplied = errors.New("nothing here supplies")

// data is what one render is given: the values the key has and no others, so a
// template naming another fails rather than quietly rendering nothing. A name the
// key has that nothing supplied is refused here rather than left to the template, so
// that a value no system in the wiring knows how to supply reads as that rather than
// as a missing map key.
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

// mark is a value standing in for one that has not been supplied: the shortest any
// real value is, so a command rendering with it renders with anything.
const mark = "x"

// fill is every value the key has, set to the one mark. Binding renders both arms a
// key makes, an empty one and a filled one, and [Command.Applies] fills the names no
// source has supplied yet.
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
// an array of strings, one per argv element. Which values it may place follows
// from the key it was read from, so bind judges that later.
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

// Applies reports whether the values in hand leave a command to run, standing in
// for every value no source has supplied yet. It is how a tool the machine does not
// have is told apart from one whose values are still coming: an unset editor is
// supplied as nothing and refused here, while a merge-base that only exists once
// the worktree does is not supplied at all and stood in for.
func (c Command) Applies(vals worktree.Values) error {
	probe := c.values.fill(mark)
	// What a source actually supplied wins over the stand-in, so a value supplied
	// empty is judged as the empty thing it is.
	maps.Copy(probe, vals)
	_, err := c.Render(probe)
	return err
}

// Render builds the argv, dropping every element that renders to nothing, so an
// optional flag is one element rather than a pair. bind settled which values are
// named here, and refused every command that could fail to render for them; what
// is left to fail is a value nothing supplied.
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
