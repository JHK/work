package config

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JHK/work-cli/internal/worktree"
)

// These are the names a settings file switches an integration on under, and
// also the names the implementations answer with. Nothing in the compiler holds
// the two spellings together.
const (
	GithubIntegration = "github"
	BeadsIntegration  = "beads"
	MiseIntegration   = "mise"
	ClaudeIntegration = "claude"
)

// sourceNames are the resolvers a place can be sourced to, which is not
// [IntegrationNames]: an action is never one, and git is in no settings list.
func sourceNames() []string {
	return []string{GithubIntegration, BeadsIntegration, string(worktree.GitSource)}
}

// integrationsKey is the one list that switches integrations on.
const integrationsKey = "integrations"

// IntegrationNames are the integrations a settings file can name, in the order
// a dump prints them.
func IntegrationNames() []string {
	return []string{GithubIntegration, BeadsIntegration, MiseIntegration, ClaudeIntegration}
}

// On reports whether the settings named that integration. One name carries an
// integration wherever it appears, so the tracker named once is on at both the
// seams it fills.
func (c Config) On(name string) bool { return slices.Contains(c.Integrations, name) }

// switchedOn are the integrations that switched on, in the compiled-in order
// and each named once, which is what a loaded Config holds.
func (c Config) switchedOn() []string {
	return slices.DeleteFunc(IntegrationNames(), func(name string) bool { return !c.On(name) })
}

func (c Config) validateIntegrations() error {
	names := IntegrationNames()
	for _, name := range c.Integrations {
		if !slices.Contains(names, name) {
			return fmt.Errorf("%q is no integration work has; they are %s", name, strings.Join(names, ", "))
		}
	}
	return nil
}
