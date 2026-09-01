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
	"github.com/JHK/work-cli/internal/resolve/plain"
	"github.com/JHK/work-cli/internal/work"
)

// Wire names every implementation the settings asked for. A system the list
// leaves out is wired nowhere.
func Wire(repo, checkout string, cfg config.Config) work.Systems {
	return work.Systems{
		Resolvers: resolving(repo, checkout, cfg),
		Actions:   acting(repo, cfg),
		Openers:   opening(cfg),
		Named:     plain.Named(repo, checkout),
	}
}

// resolving is the chain an identifier is put to. The order is the one they are
// asked in: the first to recognise an identifier takes it, and the last takes
// whatever is left.
func resolving(repo, checkout string, cfg config.Config) []work.Resolver {
	var chain []work.Resolver

	// A bare number is a pull request and every other name is a possible ticket id,
	// so the forge is asked ahead of the tracker.
	if cfg.On(config.GithubSystem) {
		chain = append(chain, github.New(repo, cfg.Github))
	}
	if cfg.On(config.BeadsSystem) {
		chain = append(chain, resolvebeads.New(repo, checkout, cfg.Beads))
	}
	// Last, and never off: a worktree nothing recognises is still one to reach.
	return append(chain, plain.New(repo, checkout))
}

// acting is what a worktree coming into being means. The tracker is one system
// on both seams under the one name, so the half [resolving] counts is not
// counted again here.
func acting(repo string, cfg config.Config) []work.Action {
	var run []work.Action

	if cfg.On(config.BeadsSystem) {
		run = append(run, actionbeads.New(repo))
	}
	if cfg.On(config.MiseSystem) {
		run = append(run, mise.Trust{})
	}
	return run
}

func opening(cfg config.Config) []work.Opener {
	var on []work.Opener

	if cfg.On(config.ClaudeSystem) {
		on = append(on, claude.New(cfg.Claude))
	}
	// Never off either: a worktree always has something to open on.
	return append(on, shell.Opener{})
}
