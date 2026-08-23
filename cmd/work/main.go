// Command work turns a ticket, a pull request, or an open worktree into a git
// worktree, and hands it to what that worktree opens on.
package main

import (
	"os"

	actionbeads "github.com/JHK/work-cli/internal/action/beads"
	"github.com/JHK/work-cli/internal/action/claude"
	"github.com/JHK/work-cli/internal/action/mise"
	"github.com/JHK/work-cli/internal/action/shell"
	"github.com/JHK/work-cli/internal/cli"
	"github.com/JHK/work-cli/internal/config"
	resolvebeads "github.com/JHK/work-cli/internal/resolve/beads"
	"github.com/JHK/work-cli/internal/resolve/github"
	"github.com/JHK/work-cli/internal/resolve/plain"
	"github.com/JHK/work-cli/internal/work"
)

// version is stamped by the mise build tasks; a plain go build keeps the default.
var version = "dev"

func main() {
	os.Exit(cli.Execute(version, wire))
}

// wire is the only place work's implementations are named.
//
// The order is the one they are asked in: the first resolver to recognise an
// identifier takes it, and the last takes whatever is left.
func wire(repo, checkout string, cfg config.Config) work.Systems {
	var (
		resolvers []work.Resolver
		actions   []work.Action
		openers   []work.Opener
	)

	// A bare number is a pull request and every other name is a possible ticket id,
	// so the forge is asked ahead of the tracker.
	if cfg.Github.Enabled {
		resolvers = append(resolvers, github.New(repo, cfg.Branch))
	}
	// One system on both seams under the one name: it resolves the tickets and
	// claims the one a worktree was made for.
	if cfg.Beads.Enabled {
		resolvers = append(resolvers, resolvebeads.New(repo, checkout, cfg.Branch))
		actions = append(actions, actionbeads.New(repo))
	}
	// Last, and never off: a worktree nothing recognises is still one to reach.
	resolvers = append(resolvers, plain.New(repo, checkout))

	if cfg.Mise.Enabled {
		actions = append(actions, mise.Trust{})
	}

	if cfg.Claude.Enabled {
		openers = append(openers, claude.New(cfg.Claude))
	}
	// Never off either: a worktree always has something to open on.
	openers = append(openers, shell.Action{})

	return work.Systems{
		Resolvers: resolvers,
		Actions:   actions,
		Openers:   openers,
		Named:     plain.Named(repo, checkout),
	}
}
