// Package wiring is the only place work's implementations are named: it turns a
// repository's settings into the integrations the core asks at each seam.
package wiring

import (
	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/integration/beads"
	"github.com/JHK/work-cli/internal/integration/claude"
	"github.com/JHK/work-cli/internal/integration/github"
	"github.com/JHK/work-cli/internal/integration/mise"
	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// Wire names every implementation the settings asked for. An integration the list
// leaves out is wired nowhere.
func Wire(repo worktree.Repo, checkout worktree.Path, cfg config.Config) work.Integrations {
	return work.Integrations{
		Resolvers: resolving(repo, checkout, cfg),
		Actions:   acting(repo, cfg),
	}
}

// resolving is the settings' integrations, in the order they are asked.
func resolving(repo worktree.Repo, checkout worktree.Path, cfg config.Config) []work.Resolver {
	var chain []work.Resolver

	// A bare number is a pull request and every other name is a possible ticket id,
	// so the forge is asked ahead of the tracker.
	if cfg.On(config.GithubIntegration) {
		chain = append(chain, github.New(repo))
	}
	if cfg.On(config.BeadsIntegration) {
		chain = append(chain, beads.NewResolver(repo, checkout, cfg.Beads))
	}
	return chain
}

// acting is what a worktree that exists is handed to, at both of an action's
// moments. The tracker is one integration on both seams, so [resolving] counts
// it, not this.
func acting(repo worktree.Repo, cfg config.Config) []work.Action {
	var run []work.Action

	if cfg.On(config.BeadsIntegration) {
		run = append(run, beads.NewClaim(repo))
	}
	if cfg.On(config.MiseIntegration) {
		run = append(run, mise.Trust{})
	}
	if cfg.On(config.ClaudeIntegration) {
		run = append(run, claude.New(cfg.Claude))
	}
	return run
}
