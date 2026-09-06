// Package wiring is the only place work's implementations are named: it turns a
// repository's settings into the systems the core asks at each seam.
package wiring

import (
	actionbeads "github.com/JHK/work-cli/internal/action/beads"
	"github.com/JHK/work-cli/internal/action/claude"
	"github.com/JHK/work-cli/internal/action/mise"
	"github.com/JHK/work-cli/internal/action/shell"
	"github.com/JHK/work-cli/internal/config"
	resolvebeads "github.com/JHK/work-cli/internal/resolve/beads"
	"github.com/JHK/work-cli/internal/resolve/github"
	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// Wire names every implementation the settings asked for. A system the list
// leaves out is wired nowhere.
func Wire(repo worktree.Repo, checkout worktree.Path, cfg config.Config) work.Systems {
	return work.Systems{
		Resolvers: resolving(repo, checkout, cfg),
		Actions:   acting(repo, cfg),
		Handback:  shell.Handback{},
	}
}

// resolving is the settings' systems, in the order they are asked.
func resolving(repo worktree.Repo, checkout worktree.Path, cfg config.Config) []work.Resolver {
	var chain []work.Resolver

	// A bare number is a pull request and every other name is a possible ticket id,
	// so the forge is asked ahead of the tracker.
	if cfg.On(config.GithubSystem) {
		chain = append(chain, github.New(repo, cfg.Github))
	}
	if cfg.On(config.BeadsSystem) {
		chain = append(chain, resolvebeads.New(repo, checkout, cfg.Beads))
	}
	return chain
}

// acting is what a worktree that exists is handed to, at both of an action's
// moments. The tracker is one system on both seams, so [resolving] counts it, not this.
func acting(repo worktree.Repo, cfg config.Config) []work.Action {
	var run []work.Action

	if cfg.On(config.BeadsSystem) {
		run = append(run, actionbeads.New(repo))
	}
	if cfg.On(config.MiseSystem) {
		run = append(run, mise.Trust{})
	}
	if cfg.On(config.ClaudeSystem) {
		run = append(run, claude.New(cfg.Claude))
	}
	return run
}
