# Worktree switcher core

*Epic: work-cli-cce*

`work` knows worktrees and nothing else. A ticket, a pull request, an agent, an editor and mise all sit outside the core, behind interfaces it calls.

`internal/work` imports `beads`, `forge` and `mise` outright today, and `Provision` switches on the target's kind to fetch a pull request's ref, delegate to `bd`, or ask git. Both grow with every tracker, forge and tool `work` learns, and the growth lands in the file that should be the most stable in the repository.

## The seams

Two, on either side of the worktree existing.

A **resolver** turns an identifier into a worktree. It offers the picker its candidates, says whether an identifier is its own, names the branch the worktree checks out and creates it. The bead's slug and the pull request's ref and fetch are here, and for a plain name the absence of both.

An **action** is handed a worktree that exists, and whether it was just created. The agent handoff, the shell, the editor and the diff are actions. So is trusting the worktree's mise config, which does its work on creation and nothing on entry. `ask` is the action holding the others, opening on the one chosen.

One system can fill both roles. A bead resolves an identifier, and claiming it is what the worktree coming into being means, so beads is an action too. `--no-claim` turns off that action rather than qualifying the resolver, and closing the bead when the work finishes is the same action at the other end. An action reaching a tracker this way is told what resolved the worktree it was handed.

Behind a seam an implementation is free to be specific, and is named for what it speaks to. The agent handoff opens Claude Code, so the action is `claude` rather than `agent`; the resolver reading pull requests speaks to GitHub, so it is `github` rather than `forge`. Flags and configuration values follow the names.

The core owns the sequence and nothing more: ask the resolvers what this identifier is, create the worktree if it is not there, hand it to an action. A capability surviving only as a special case there is shaped wrong rather than special.

Each seam names the error types it returns, because the core acts on differences it cannot otherwise see. An identifier a resolver does not recognise moves to the next resolver; one it recognised but could not reach `bd` for stops the run. Actions divide the same way, between a tool that is absent and one that ran and failed.

## Package boundaries

Every implementation is a package of its own under the seam it implements, and a system filling both roles takes the same name on both sides over the one client it already has.

```
internal/worktree                 the types both sides speak
internal/work                     the core, declaring Resolver and Action
internal/resolve/{beads,github,plain}
internal/action/{beads,claude,shell,editor,diff,mise}
cmd/work                          the only place implementations are named
```

The types sit in a leaf package so an implementation can speak the core's vocabulary without importing the core, and the interfaces are declared where they are consumed. A client a seam package sits on top of, `internal/beads` and `internal/mise` among them, stays where it is.

The separation is checkable: `internal/work` imports `internal/worktree`, `internal/git` and the standard library, and nothing else. Git is the exception because worktrees are git's.

## Configuration

Nearly every key in [the configuration](../references/configuration.md) belongs to one implementation already, `branch.ticket` to the beads resolver and `agent.*` to the claude action, and the seams are what give them an owner to be grouped by.

Both choices that come with it are the user's: which implementations run at all, so a repository with no tracker is never offered tickets and a shipped action can be kept off the screen, and whatever a single implementation needs to be told.

A key exists because a user has a choice to make and not because the code grew an interface, so the structure never shows through on its own. `[action]` is a worked example of the cost: `action.create` and `action.enter` name an action, so that table cannot also be the place an action's own keys live.

## Flags

An implementation may need a flag as well as a key. `--no-claim` is the beads action's, and every action is named by one. A verb owns its flags, which is [R2](../rules/command-grammar.md), and the core presents in `--help` a set it does not hold by hand.

## Out of scope

- Runtime loading, discovery, registration and implementations from outside the repository. The set is compiled in.
- Preserving the behaviour the current code has, or the commands, flags, keys and names carrying it.
- The verbs and their flags as [the command line](../references/cli.md) records them, which the seams hang off rather than change.
- What the picker offers.
