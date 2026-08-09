package config

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Agent is what a worktree is handed over to. Each key falls back to
// defaultAgent on its own, so a config pointing work at another agent sets every
// one of them: the agent that starts a conversation is the one that returns to
// it.
type Agent struct {
	StartTicketCommand      Command `toml:"start-ticket"`
	StartPullRequestCommand Command `toml:"start-pull-request"`
	StartSessionCommand     Command `toml:"start-session"`
	ResumeSessionCommand    Command `toml:"resume-session"`
}

const (
	startTicketKey      = "agent.start-ticket"
	startPullRequestKey = "agent.start-pull-request"
	startSessionKey     = "agent.start-session"
	resumeSessionKey    = "agent.resume-session"
)

// Launch is everything a command may be rendered with. Which of these a key
// actually has is its launchValues; a template naming any other is refused.
type Launch struct {
	Name    string // what the target is retyped as, or the file config edit opens
	Dir     string // the worktree the process has already changed into, or that file
	ID      string // the ticket id
	Title   string // the ticket title
	Number  string // the pull request number
	Session string // the one conversation the worktree carries, empty where it carries several
	Shell   string // $SHELL, or /bin/sh
	Editor  string // $VISUAL, else $EDITOR
	Base    string // the commit the worktree's branch forked from
}

// StartTicket is the command a fresh ticket worktree opens on.
func (a Agent) StartTicket(l Launch) ([]string, error) {
	return a.startTicket().render(l)
}

// StartPullRequest is the command a fresh pull request worktree opens on.
func (a Agent) StartPullRequest(l Launch) ([]string, error) {
	return a.startPullRequest().render(l)
}

// StartSession is the command a worktree that carries no conversation opens on.
func (a Agent) StartSession(l Launch) ([]string, error) {
	return a.startSession().render(l)
}

// ResumeSession is the command that returns to the conversation a worktree
// carries. An empty .Session drops the element that placed it, so the one
// conversation is returned to outright and several reach the agent's own list.
func (a Agent) ResumeSession(l Launch) ([]string, error) {
	return a.resumeSession().render(l)
}

// An unset command is the compiled-in one, as it is below, so a Config that
// never reached Load still names something to run.
func (a Agent) startTicket() Command {
	return a.StartTicketCommand.or(defaultAgent.StartTicketCommand)
}

func (a Agent) startPullRequest() Command {
	return a.StartPullRequestCommand.or(defaultAgent.StartPullRequestCommand)
}

func (a Agent) startSession() Command {
	return a.StartSessionCommand.or(defaultAgent.StartSessionCommand)
}

func (a Agent) resumeSession() Command {
	return a.ResumeSessionCommand.or(defaultAgent.ResumeSessionCommand)
}

// validate judges each command against the values its key renders with, and
// names the key work cannot use the value of.
func (a *Agent) validate() (string, error) {
	if err := a.StartTicketCommand.bind(startTicketValues); err != nil {
		return startTicketKey, err
	}
	if err := a.StartPullRequestCommand.bind(startPullRequestValues); err != nil {
		return startPullRequestKey, err
	}
	if err := a.StartSessionCommand.bind(startSessionValues); err != nil {
		return startSessionKey, err
	}
	if err := a.ResumeSessionCommand.bind(resumeSessionValues); err != nil {
		return resumeSessionKey, err
	}
	return "", nil
}

// defaultAgent is the session work opened before any of this was a setting,
// less the review prompt a pull request used to get, which is the reviewer's to
// choose.
var defaultAgent = Agent{
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
// for what any of them reaches.
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

// Shell is the command an existing worktree is entered with, and the one
// --shell hands over to.
func (o Open) Shell(l Launch) ([]string, error) {
	return o.shell().render(l)
}

// Editor is the command --editor hands the worktree to.
func (o Open) Editor(l Launch) ([]string, error) {
	return o.editor().render(l)
}

// Diff is the command --diff hands the worktree to.
func (o Open) Diff(l Launch) ([]string, error) {
	return o.diff().render(l)
}

// An unset command is the compiled-in one, as [Agent]'s are.
func (o Open) shell() Command  { return o.ShellCommand.or(defaultOpen.ShellCommand) }
func (o Open) editor() Command { return o.EditorCommand.or(defaultOpen.EditorCommand) }
func (o Open) diff() Command   { return o.DiffCommand.or(defaultOpen.DiffCommand) }

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
	// Not the three-dot form, which would leave uncommitted work out.
	DiffCommand: mustCommand(diffValues, "git", "diff", "{{.Base}}"),
}

// Command is a whole command line: one [text/template] per argv element,
// rendered with the values its key has. Which values those are is settled by
// bind, once the key the command was read from says.
type Command struct {
	parts  []tmpl
	values launchValues
}

// launchValues are the value names one key's command may place, and the key it
// is named by.
type launchValues struct {
	key   string
	names []string
}

// common are the values every command has, whatever it is for; a key's own
// follow them.
var common = []string{"Name", "Dir"}

var (
	startTicketValues      = launchValues{startTicketKey, slices.Concat(common, []string{"ID", "Title"})}
	startPullRequestValues = launchValues{startPullRequestKey, slices.Concat(common, []string{"Number"})}
	startSessionValues     = launchValues{startSessionKey, common}
	resumeSessionValues    = launchValues{resumeSessionKey, slices.Concat(common, []string{"Session"})}
	shellValues            = launchValues{shellKey, slices.Concat(common, []string{"Shell"})}
	editorValues           = launchValues{editorKey, slices.Concat(common, []string{"Editor"})}
	diffValues             = launchValues{diffKey, slices.Concat(common, []string{"Base"})}
)

// data is what one render is given: the values the key has and no others, so a
// template naming another fails rather than quietly rendering nothing.
func (v launchValues) data(l Launch) map[string]any {
	data := map[string]any{
		"Name": l.Name, "Dir": l.Dir,
		"ID": l.ID, "Title": l.Title, "Number": l.Number, "Session": l.Session,
		"Shell": l.Shell, "Editor": l.Editor, "Base": l.Base,
	}
	// Dropping rather than picking: a name here that Launch has no field for leaves
	// the map short, which bind refuses, instead of rendering as <no value>.
	maps.DeleteFunc(data, func(name string, _ any) bool {
		return !slices.Contains(v.names, name)
	})
	return data
}

// list is how the values read in a refusal.
func (v launchValues) list() string {
	out := make([]string, len(v.names))
	for i, name := range v.names {
		out[i] = "{{." + name + "}}"
	}
	return strings.Join(out, ", ")
}

// filledLaunch stands for a target carrying every value, so that binding renders
// the arms an empty one would skip.
var filledLaunch = Launch{
	Name: "x", Dir: "x", ID: "x", Title: "x", Number: "x",
	Session: "x", Shell: "x", Editor: "x", Base: "x",
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
func mustCommand(v launchValues, texts ...string) Command {
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
func (c *Command) bind(v launchValues) error {
	if len(c.parts) == 0 {
		return errors.New("names no command to run")
	}
	for _, probe := range []Launch{{}, filledLaunch} {
		data := v.data(probe)
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

// render builds the argv, dropping every element that renders to nothing, so an
// optional flag is one element rather than a pair. bind settled which values are
// named here, and refused every command that could fail to render for them.
func (c Command) render(l Launch) ([]string, error) {
	data := c.values.data(l)
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
