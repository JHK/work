# Command line

One command, an optional identifier and flags, and `init` for the shell integration. `work --help` lists the forms and the flags.

## Identifiers

| Argument | Resolves to |
|---|---|
| `feature/x` | the worktree already open under that name, ahead of everything below |
| `bd-42` | that bead, for any name matching `[A-Za-z0-9][A-Za-z0-9._-]*` |
| `7`, `007` | pull request 7 |
| [`pr-7`](configuration.md) | pull request 7, so a worktree name can be retyped |
| `https://host/owner/repo/pull/7`, with any trailing path | pull request 7 |
| omitted | [the picker](#what-the-picker-offers) |

A pull request number is read against the current repository, whatever host the URL names.

Refused: leading dashes, path separators, `.` and `..`.

`init` and `help` are commands, so neither reaches the table.

## What the picker offers

`work` with no argument opens an fzf list of:

- every worktree git reports, less the main checkout
- the open pull requests of `origin`
- the ready beads without a pull request

A worktree is read off its branch. One whose branch names neither a bead nor a pull request is offered under that branch.

`bd` titles the bead rows, `gh pr list` the pull request rows, drafts included. gh is pinned to origin's URL. A [missing tool](tools.md) costs its own rows and its own titles.

## What each invocation does

| Invocation | On a target with no worktree |
|---|---|
| `work <id>` | vet, create, claim, and launch |
| `work <pr>` | create, and launch |
| `work <name> --create` | create on a branch of that name, and launch |
| `work <id> --shell` | vet, create, claim, and open a shell |
| `work <id> --editor` | vet, create, claim, and open the editor |
| `work <id> --diff` | vet, create, claim, and open the diff |
| `work <id> --no-claim` | vet, create, and launch |
| `work` | as for the target picked |

Which [command](configuration.md#commands) a worktree opens on says nothing about its ticket: creating one for a bead vets it and claims it whichever command that is, and `--no-claim`, which combines with any of them, declines the claim alone. Only a new worktree needs a [directory](configuration.md#keys) chosen for it.

Every form above re-enters a worktree that already exists, `--create` excepted: it asserts the name is free, and refuses one a branch already holds. A target's worktree is the one checked out on its [branch](configuration.md#keys), wherever git reports it, less the branches [a longer ticket name owns](../explanation/worktree-per-ticket.md); entering it hands over what [`action.enter`](configuration.md#actions) names, [`open.shell`](configuration.md#keys) by default, or what the flag named. The launch above is likewise what [`action.create`](configuration.md#actions) names by default.

Vetting refuses a deferred, closed, epic, criteria-less or dependency-blocked bead, and the message says which; the rule is [bead-workflow policy](../explanation/worktree-per-ticket.md). No flag reaches past it.

## Flags

| Flag | Effect |
|---|---|
| `--create` | a worktree of the name given, on a new branch spelled the same way and forked from the main checkout's `HEAD`; nothing is asked of `bd` |
| `--agent` | hands the worktree to its agent: a newly created one to [`agent.start-ticket`, `agent.start-pull-request` or `agent.start-session`](configuration.md#keys) by what it was created for, an existing one to what [the conversations it carries](#what-a-worktree-carries) name |
| `--shell` | hands the worktree to [`open.shell`](configuration.md#keys), a newly created one included |
| `--editor` | hands the worktree to [`open.editor`](configuration.md#keys) |
| `--diff` | hands the worktree to [`open.diff`](configuration.md#keys), its work against the point its branch forked from, committed and uncommitted alike |
| `--no-claim` | [creates the worktree without claiming](#what-each-invocation-does) the bead |
| `--version` | prints and exits, touching no repository |

`--agent`, `--shell`, `--editor` and `--diff` exclude one another, and each wins over [`action.create` and `action.enter`](configuration.md#actions) for that invocation. `--create` excludes `--no-claim`, a worktree with no ticket behind it having no claim to decline.

### What a worktree carries

`--agent` counts the conversations the agent has recorded for the worktree's directory and hands over accordingly:

| Conversations | Handed to |
|---|---|
| none | [`agent.start-session`](configuration.md#keys) |
| one | [`agent.resume-session`](configuration.md#keys) naming it, so nothing is asked |
| several | `agent.resume-session` naming none, which is the agent's own list |

The count is `claude`'s transcript store, read by `internal/sessions/`. A transcript `claude -p` wrote is not a conversation to return to and does not count.

An [`open.editor`](configuration.md#commands) that names nothing to run refuses the invocation before anything is created or claimed. [`open.diff`](configuration.md#commands) is rendered once the worktree exists, so a diff that will not render leaves the worktree made and the ticket claimed.

The version reported is what [stamped the binary](mise-tasks.md#version-stamping), or `dev` when nothing did.

## Shell integration

`work init fish | source` in `config.fish` completes the identifier with what the picker offers.

## Handoff

`work` changes into the worktree and replaces itself with the command, so the calling shell keeps its own directory. A failure to enter is one line on stderr and exit 1; a dismissed picker exits 1 silently.

Which command each path runs is `internal/work/handoff.go`.
