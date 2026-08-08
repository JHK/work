# Command line

`work --help` lists the commands and the flags.

## Commands

| Command | Does |
|---|---|
| `work [<identifier>]` | [enters](#invocations) the worktree an [identifier](#identifiers) names, creating it if there is none |
| [`work remove [<name>]`](#remove) | removes a worktree and deletes the branch it had checked out |
| [`work init fish`](#shell-integration) | prints the shell integration |

A command wins the first position, `help` included, so no worktree named for one is reachable.

## Identifiers

| Argument | Resolves to |
|---|---|
| `feature/x` | the worktree already open under that name, ahead of everything below |
| `bd-42` | that bead, for any name matching `[A-Za-z0-9][A-Za-z0-9._-]*` |
| `7`, `007` | pull request 7 |
| [`pr-7`](configuration.md) | pull request 7, so a worktree name can be retyped |
| `https://host/owner/repo/pull/7`, with any trailing path | pull request 7 |
| omitted | [the picker](#the-picker) |

A pull request number is read against the current repository, whatever host the URL names.

Refused: leading dashes, path separators, `.` and `..`.

## The picker

`work` with no argument opens an fzf list of:

- every worktree git reports, less the main checkout
- the open pull requests of `origin`
- the ready beads without a pull request

Under [`remove`](#remove) the list is the worktrees alone, there being nothing else to remove.

A worktree is read off its branch. One whose branch names neither a bead nor a pull request is offered under that branch.

`bd` titles the bead rows, `gh pr list` the pull request rows, drafts included. gh is pinned to origin's URL. A [missing tool](tools.md) costs its own rows and its own titles.

## The screen

The second question, put where `--ask` or an [`action`](configuration.md#actions) key naming `ask` reached it: a second fzf list, of the actions that apply, in this order.

| Row | Hands the worktree to |
|---|---|
| `agent` | [`--agent`](#flags) |
| `shell` | [`--shell`](#flags) |
| `editor` | [`--editor`](#flags), and is left off where [`open.editor`](configuration.md#commands) names nothing to run |
| `diff` | [`--diff`](#flags) |

`ask` is never a row: the screen is what it names.

It is drawn after the vetting and before anything is created, so a refused ticket is refused without a question, and a dismissed list exits 1 with nothing created and nothing claimed.

## Invocations

| Target | On one with no worktree |
|---|---|
| a bead | vet, create, claim, and launch |
| a pull request | create, and launch |
| a [`--create`](#flags) name | create on a branch of that name, and launch |

Which [command](configuration.md#commands) a worktree opens on says nothing about its ticket: creating one for a bead vets it and claims it whichever command that is, and `--no-claim`, which combines with any of them, declines the claim alone. A target the picker handed over goes the same way as one named. Only a new worktree needs a [directory](configuration.md#keys) chosen for it.

Every target above re-enters a worktree that already exists, `--create` excepted: it asserts the name is free, and refuses one a branch already holds. A target's worktree is the one checked out on its [branch](configuration.md#keys), wherever git reports it, less the branches [a longer ticket name owns](../explanation/worktree-per-ticket.md); entering it hands over what [`action.enter`](configuration.md#actions) names, [the screen](#the-screen) by default, or what the flag named. The launch above is likewise what [`action.create`](configuration.md#actions) names by default.

Vetting refuses a deferred, closed, epic, criteria-less or dependency-blocked bead, and the message says which; the rule is [bead-workflow policy](../explanation/worktree-per-ticket.md). No flag reaches past it.

## Flags

| Flag | Effect |
|---|---|
| `--create` | a worktree of the name given, on a new branch spelled the same way and forked from the main checkout's `HEAD`; nothing is asked of `bd` |
| `--agent` | hands the worktree to its agent: a newly created one to [`agent.start-ticket`, `agent.start-pull-request` or `agent.start-session`](configuration.md#keys) by what it was created for, an existing one to what [the conversations it carries](#conversations) name |
| `--shell` | hands the worktree to [`open.shell`](configuration.md#keys), a newly created one included |
| `--editor` | hands the worktree to [`open.editor`](configuration.md#keys) |
| `--diff` | hands the worktree to [`open.diff`](configuration.md#keys), its work against the point its branch forked from, committed and uncommitted alike |
| `--ask` | offers [the screen](#the-screen) and hands the worktree to the action picked there; needs fzf |
| `--no-claim` | [creates the worktree without claiming](#invocations) the bead |
| `--version` | prints and exits, touching no repository |

`--agent`, `--shell`, `--editor`, `--diff` and `--ask` exclude one another, and each wins over [`action.create` and `action.enter`](configuration.md#actions) for that invocation. `--create` excludes `--no-claim`, a worktree with no ticket behind it having no claim to decline.

An [`open.editor`](configuration.md#commands) that names nothing to run refuses the invocation before anything is created or claimed. [`open.diff`](configuration.md#commands) is rendered once the worktree exists, so a diff that will not render leaves the worktree made and the ticket claimed.

The version reported is what [stamped the binary](mise-tasks.md#version-stamping), or `dev` when nothing did.

### Conversations

`--agent` counts the conversations the agent has recorded for the worktree's directory and hands over accordingly:

| Conversations | Handed to |
|---|---|
| none | [`agent.start-session`](configuration.md#keys) |
| one | [`agent.resume-session`](configuration.md#keys) naming it, so nothing is asked |
| several | `agent.resume-session` naming none, which is the agent's own list |

The count is [`claude`'s transcript store](agent.md), read by `internal/sessions/`. A transcript `claude -p` wrote is not a conversation to return to and does not count.

## remove

`work remove <name>` removes the worktree git reports for that [identifier](#identifiers) and deletes the branch it had checked out, leaving the ticket alone and asking no tracker. With no name it opens [the picker](#the-picker).

`--force`, the command's only flag, takes a worktree with modified or untracked files and a branch not merged into the main checkout, which are refused without it. Both are weighed before anything is removed. The worktree the shell is standing in is refused either way.

## Shell integration

`work init fish | source` in `config.fish` completes the identifier with what the picker offers, and [`remove`](#remove)'s name with the worktrees alone.

## Handoff

`work` changes into the worktree and replaces itself with the command, so the calling shell keeps its own directory. A failure to enter is one line on stderr and exit 1; either list dismissed exits 1 silently.

[`remove`](#remove) hands over to nothing: it names on stdout what it removed and exits.

Which command each path runs is `internal/work/handoff.go`.
