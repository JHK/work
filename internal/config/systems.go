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

// resolved are the systems that switched on, in the compiled-in order and each
// named once, which is what a loaded Config holds.
func (c Config) resolved() []string {
	return slices.DeleteFunc(SystemNames(), func(name string) bool { return !c.On(name) })
}

// validateSystems refuses a name no system goes by.
func (c Config) validateSystems() error {
	names := SystemNames()
	for _, name := range c.Systems {
		if !slices.Contains(names, name) {
			return fmt.Errorf("%q is no system work has; they are %s", name, strings.Join(names, ", "))
		}
	}
	return nil
}
