package config

import (
	"fmt"
	"slices"
	"strings"
)

// ClaudeOpener and ShellOpener are the two things a worktree opens on, under the
// names the openers go by.
const (
	ClaudeOpener = ClaudeSystem
	ShellOpener  = "shell"
)

// creating are the verbs that can bring a worktree into being, which is what
// claude.on-creation may name, and entering the rest of the command line.
// Nothing in the compiler holds these and the command tree together.
var (
	creating = []string{"add", "carry", "go"}
	entering = []string{"config", "init", "list", "move", "remove", "switch"}
)

// Creating are the verbs a worktree can come into being under.
func Creating() []string { return creating }

// Verbs are every verb the command line takes.
func Verbs() []string { return slices.Concat(creating, entering) }

// defaultOnCreation is what opens a session where the key names nothing: the two
// verbs a person reaches for to start work. It stays out of defaultClaude, whose
// fields [Default] hands to the decoder to write over.
var defaultOnCreation = []string{"add", "go"}

// OnCreation are the verbs whose creations open a session. An unset key is the
// compiled-in default, so a Config that never reached Load still names them.
func (c Claude) OnCreation() []string {
	if c.OnCreationVerbs == nil {
		return defaultOnCreation
	}
	return c.OnCreationVerbs
}

// OpensOnCreation reports whether a worktree created under that verb opens on a
// session, which the settings leaving the agent out says of no verb.
func (c Config) OpensOnCreation(verb string) bool {
	return c.On(ClaudeSystem) && slices.Contains(c.Claude.OnCreation(), verb)
}

// validate refuses a verb no worktree comes into being under, and a word no verb
// goes by at all.
func (c Claude) validateOnCreation() error {
	for _, verb := range c.OnCreationVerbs {
		if slices.Contains(creating, verb) {
			continue
		}
		if slices.Contains(entering, verb) {
			return fmt.Errorf("%q creates no worktree; the verbs that do are %s", verb, strings.Join(creating, ", "))
		}
		return fmt.Errorf("%q is not a verb; the verbs that create a worktree are %s", verb, strings.Join(creating, ", "))
	}
	return nil
}
