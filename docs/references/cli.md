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
| `work <id> --shell` | create only; the bead is left as it is |
| `work <id> --editor` | vet, create, claim, and open the editor |
| `work` | as for the target picked |

Creating a worktree claims the bead and runs the [command](configuration.md#commands) its kind of target opens on. Only a new worktree needs a [directory](configuration.md#keys) chosen for it.

Every form above re-enters a worktree that already exists. A target's worktree is the one checked out on its [branch](configuration.md#keys), wherever git reports it; entering it hands over [`open.shell`](configuration.md#keys), names the conversations it carries, and prints the line that returns to them.

Vetting refuses a deferred, closed, epic, criteria-less or dependency-blocked bead, and the message says which; the rule is [bead-workflow policy](../explanation/worktree-per-ticket.md). The paths that do not claim do not vet.

## Flags

| Flag | Effect |
|---|---|
| `--model`, `--effort` | placed where the [command](configuration.md#commands) names them; one naming neither drops both silently |
| `--shell` | [create only](#what-each-invocation-does) |
| `--editor` | hands the worktree to [`open.editor`](configuration.md#keys); exclusive with `--shell` |
| `--version` | prints and exits, touching no repository |

An [`open.editor`](configuration.md#commands) that names nothing to run refuses the invocation before anything is created or claimed.

The version reported is what [stamped the binary](mise-tasks.md#version-stamping), or `dev` when nothing did.

## Shell integration

`work init fish | source` in `config.fish` completes the identifier with what the picker offers.

## Handoff

`work` changes into the worktree and replaces itself with the command, so the calling shell keeps its own directory. A failure to enter is one line on stderr and exit 1; a dismissed picker exits 1 silently.

Which command each path runs is `internal/work/handoff.go`.
