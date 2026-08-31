package config

import (
	"fmt"
	"slices"
	"strings"
)

// These are the names a settings file switches a system on under, and also the
// names the implementations answer with. Nothing in the compiler holds the two
// spellings together.
const (
	GithubSystem = "github"
	BeadsSystem  = "beads"
	MiseSystem   = "mise"
	ClaudeSystem = "claude"
)

// plainSystem is the resolver that answers for whatever the others leave. It
// runs whatever the settings say, so no list switches it on and no file names it.
const plainSystem = "plain"

// sourceNames are the resolvers a place can be sourced to, which is not
// [SystemNames]: an action is never one, and plain is in no settings list.
func sourceNames() []string {
	return []string{GithubSystem, BeadsSystem, plainSystem}
}

// systemsKey is the one list that switches systems on.
const systemsKey = "systems"

// SystemNames are the systems a settings file can name, in the order a dump
// prints them.
func SystemNames() []string {
	return []string{GithubSystem, BeadsSystem, MiseSystem, ClaudeSystem}
}

// On reports whether the settings named that system. One name carries a system
// wherever it appears, so the tracker named once is on at both the seams it
// fills.
func (c Config) On(name string) bool { return slices.Contains(c.Systems, name) }

// switchedOn are the systems that switched on, in the compiled-in order and each
// named once, which is what a loaded Config holds.
func (c Config) switchedOn() []string {
	return slices.DeleteFunc(SystemNames(), func(name string) bool { return !c.On(name) })
}

func (c Config) validateSystems() error {
	names := SystemNames()
	for _, name := range c.Systems {
		if !slices.Contains(names, name) {
			return fmt.Errorf("%q is no system work has; they are %s", name, strings.Join(names, ", "))
		}
	}
	return nil
}
