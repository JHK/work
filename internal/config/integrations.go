package config

import (
	"fmt"
	"slices"

	"github.com/JHK/work-cli/internal/worktree"
)

// These are the names a settings file switches an integration on under, and
// also the names the implementations answer with. Nothing in the compiler holds
// the two spellings together.
const (
	GithubIntegration worktree.IntegrationName = "github"
	BeadsIntegration  worktree.IntegrationName = "beads"
	MiseIntegration   worktree.IntegrationName = "mise"
	ClaudeIntegration worktree.IntegrationName = "claude"
)

// KnownSources are the resolvers a place can be sourced to, which is not
// [KnownIntegrations]: an action is never one, and git is in no settings list.
func KnownSources() []worktree.IntegrationName {
	return []worktree.IntegrationName{GithubIntegration, BeadsIntegration, worktree.GitSource}
}

// integrationsKey is the one list that switches integrations on.
const integrationsKey = "integrations"

// KnownIntegrations are the integrations a settings file can name, in the order
// a dump prints them.
func KnownIntegrations() []worktree.IntegrationName {
	return []worktree.IntegrationName{GithubIntegration, BeadsIntegration, MiseIntegration, ClaudeIntegration}
}

// On reports whether the settings named that integration. One name carries an
// integration wherever it appears, so the tracker named once is on at both the
// seams it fills.
func (c Config) On(name worktree.IntegrationName) bool { return slices.Contains(c.Integrations, name) }

// switchedOn are the integrations that switched on, in the compiled-in order
// and each named once, which is what a loaded Config holds.
func (c Config) switchedOn() []worktree.IntegrationName {
	return slices.DeleteFunc(KnownIntegrations(), func(name worktree.IntegrationName) bool { return !c.On(name) })
}

func (c Config) validateIntegrations() error {
	known := KnownIntegrations()
	for _, name := range c.Integrations {
		if !slices.Contains(known, name) {
			return fmt.Errorf("%q is no integration work has; they are %s", name, joined(known, ", "))
		}
	}
	return nil
}
