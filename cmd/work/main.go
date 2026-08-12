// Command work turns a ticket, a pull request, or an open worktree into a git
// worktree, and hands the terminal to what that worktree opens on.
package main

import (
	"os"

	actionbeads "github.com/JHK/work-cli/internal/action/beads"
	"github.com/JHK/work-cli/internal/action/claude"
	"github.com/JHK/work-cli/internal/action/mise"
	"github.com/JHK/work-cli/internal/action/open"
	"github.com/JHK/work-cli/internal/cli"
	"github.com/JHK/work-cli/internal/config"
	resolvebeads "github.com/JHK/work-cli/internal/resolve/beads"
	"github.com/JHK/work-cli/internal/resolve/github"
	"github.com/JHK/work-cli/internal/resolve/plain"
	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
)

// version is stamped by the mise build tasks; a plain go build keeps the default.
var version = "dev"

func main() {
	os.Exit(cli.Execute(version, wire))
}

// wire is the only place work's implementations are named. A system the settings
// did not ask for is wired to nothing and kept only so that what it would have
// answered for is refused as a system switched off.
//
// The order is the one they are asked in: the first resolver to recognise an
// identifier takes it, and the last takes whatever is left.
func wire(repo, checkout string, cfg config.Config) work.Systems {
	var (
		resolvers []work.Resolver
		actions   []work.Action
		openers   []work.Opener
		off       []worktree.System
	)

	// A bare number is a pull request and every other name is a possible ticket id,
	// so the forge is asked ahead of the tracker.
	if pulls := github.New(repo, cfg.Branch); cfg.Github.Enabled {
		resolvers = append(resolvers, pulls)
	} else {
		off = append(off, pulls)
	}
	// One system on both seams under the one name: it resolves the tickets and
	// claims the one a worktree was made for.
	if tickets := resolvebeads.New(repo, checkout, cfg.Branch); cfg.Beads.Enabled {
		resolvers = append(resolvers, tickets)
		actions = append(actions, actionbeads.New(repo))
	} else {
		off = append(off, tickets)
	}
	// Last, and never off: a worktree nothing recognises is still one to reach.
	resolvers = append(resolvers, plain.New(repo, checkout))

	if trust := (mise.Trust{}); cfg.Mise.Enabled {
		actions = append(actions, trust)
	} else {
		off = append(off, trust)
	}

	if agent := claude.New(cfg.Claude); cfg.Claude.Enabled {
		openers = append(openers, agent)
	} else {
		off = append(off, agent)
	}
	// Never off either: a worktree always has something to open on.
	openers = append(openers, open.Shell(cfg.Open))

	return work.Systems{
		Resolvers: resolvers,
		Actions:   actions,
		Openers:   openers,
		// What any worktree can be described by, whichever resolver answered for it.
		Sources: []worktree.Source{
			open.Values{},
		},
		Named:    plain.Named(repo, checkout),
		Disabled: off,
	}
}
