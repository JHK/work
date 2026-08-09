# Command line

## Commands

| Command | Does |
|---|---|
| [`work switch [<identifier>]`](#switch) | enters the worktree an [identifier](#identifiers) names, creating it if there is none |
| [`work add <name>`](#add) | creates a worktree on a new branch of that name and opens it |
| [`work remove [<name>]`](#remove) | removes a worktree and deletes the branch it had checked out |
| [`work list`](#list) | prints the worktrees open |
| [`work init fish`](#init) | prints the shell integration |
| [`work config dump`](#config) | prints the effective [configuration](configuration.md) and the layer behind each key |
| [`work config edit`](#config) | opens your own [settings file](configuration.md) in `open.editor` |
| `work help [<command>]` | prints a command's help, the root's with none |
| `work [<identifier>]` | dispatched to [`switch`](#switch) |

`--help` and `--version` are the root's own, shorthands included; `--version` prints [the version](mise-tasks.md#version-stamping), in or out of a repository.

### switch

`work switch <identifier>` enters the worktree the [identifier](#identifiers) names, creating it where there is none, a pull request's on the head fetched for it.

With no identifier, [the picker](#the-picker) offers every worktree open, and the open pull requests of `origin` and ready beads that have none.

Creating one for a ticket [vets and claims it](tickets.md); `--no-claim` declines the claim alone.

### add

`work add <name>` creates a worktree on a new branch spelled exactly as the name is, forked from the main checkout's `HEAD` and asking no tracker, and hands it to what [`action.create`](configuration.md#actions) names or an [open-on flag](#open-on-flags) named. A branch already holding the name is refused.

### remove

`work remove <name>` removes the worktree git reports for that [identifier](#identifiers) and deletes the branch it had checked out, leaving the ticket alone and asking no tracker. An identifier with no worktree open is refused.

With no name, [the picker](#the-picker) offers the worktrees alone.

`--force` takes a worktree with modified or untracked files and a branch not merged into the main checkout, which are refused without it. Both are weighed before anything is removed. The worktree the shell is standing in is refused either way.

### list

`work list` prints the worktrees git reports, the main checkout excepted, one per line on stdout: the name the same worktree is retyped as, then the title bd gives the bead behind it, lined up in a column. A [missing bd](tools.md) costs the titles alone. gh is not asked, so a pull request's worktree prints as its name alone, as does one bd cannot name.

It takes no argument; a name to filter on is [`switch`](#switch)'s. With nothing open it prints nothing.

### init

`work init fish | source` in `config.fish` installs the [completion](#completion). Printing it touches no repository.

### config

`work config dump` prints what the [layers](configuration.md) resolved to here, as TOML work loads back, each key under a comment naming the layer that set it: the compiled-in default, the user file's path, or `.work.toml`. [Patterns and commands](configuration.md#branch-patterns) print as they are written, rendering one needing a target.

A configuration work would refuse to load is refused here too, with nothing printed.

`work config edit` hands [the user's file](configuration.md) to [`open.editor`](configuration.md#commands), which is given the path as `{{.Dir}}`. The file and the directory it sits in are created where neither is there yet, so an editor that creates neither still opens; a file already there is opened as it is. The repository's `.work.toml` is not this file.

Nothing is created where the editor names no command to run, where git names no repository, or where the configuration is one work would refuse to load.

## Arguments

### Identifiers

| Argument | Resolves to |
|---|---|
| `feature/x` | the worktree already open under that name, ahead of everything below |
| `bd-42` | that bead, for any name matching `[A-Za-z0-9][A-Za-z0-9._-]*` |
| `7`, `007` | pull request 7 |
| [`pr-7`](configuration.md) | pull request 7, so a worktree name can be retyped |
| `https://host/owner/repo/pull/7`, with any trailing path | pull request 7 |

A pull request number is read against the current repository, whatever host the URL names.

Refused, here and wherever else a worktree is named: leading dashes, path separators, `.` and `..`.

The worktree an identifier reaches is the one checked out on its [branch](configuration.md#keys), [wherever git reports it](../explanation/worktree-identity.md#the-branch-is-the-identity-not-the-path).

A ticket's branch is recognised, never read apart: the [pattern](configuration.md#branch-patterns) is matched against the whole branch with an id filled in and the rest wildcarded, and of the ids `bd` listed the longest owner takes it. Where `bd` names none, nothing is longer and the pattern alone settles it.

Where several worktrees match one ticket, the shortest branch takes it, ties settled on the name, so it is never git's listing order.

### The picker

An fzf list standing in for the argument a verb was not given.

A worktree row is one git reports other than the main checkout, read off its branch. One whose branch names neither a bead nor a pull request is offered under that branch.

`bd` titles the bead rows, `gh pr list` the pull request rows, drafts included. gh is pinned to origin's URL. A [missing tool](tools.md) costs its own rows and its own titles.

### Completion

What [`init`](#init)'s integration offers at each position:

| Position | Offers |
|---|---|
| the first word | the commands |
| [`config`](#config)'s sub-verb | `dump` and `edit` |
| [`switch`](#switch)'s identifier | what [the picker](#the-picker) offers |
| [`remove`](#remove)'s name | the worktrees |
| [`add`](#add)'s name | nothing |

A word past the position named completes nothing, and so does every word after [`list`](#list), which takes none. A repository that will not answer completes nothing, and no error reaches the shell.

## Opening

### Open-on flags

What a worktree is handed to, for one invocation, ahead of what [`action.create` and `action.enter`](configuration.md#actions) name. [`switch`](#switch) and [`add`](#add) carry the set, and naming two of it in one invocation is refused.

| Flag | Hands the worktree to |
|---|---|
| `--agent` | the [`agent` command](configuration.md#keys) the moment names: its launcher where the worktree was just created, else the one [its conversations](claude.md#the-contract) name |
| `--shell` | [`open.shell`](configuration.md#keys), one just created included |
| `--editor` | [`open.editor`](configuration.md#keys) |
| `--diff` | [`open.diff`](configuration.md#keys), its work against the point its branch forked from, committed and uncommitted alike |
| `--ask` | [the screen](#the-screen), and on to the action picked there; needs fzf |

### The screen

The second question, put where [`--ask`](#open-on-flags) or an [`action`](configuration.md#actions) key naming `ask` reached it: an fzf list of the actions that apply, in that table's order. `ask` is never a row, and `editor` is left off where [`open.editor`](configuration.md#commands) names nothing to run.

It is drawn after the vetting and before anything is created, so a refused ticket is refused without a question, and a dismissed list exits 1 with nothing created and nothing claimed.

## Handoff

`work` changes into the worktree and replaces itself with the command, so the calling shell keeps its own directory. [`config edit`](#config) hands over the same way, into the directory its file sits in. A failure to enter is one line on stderr and exit 1; a dismissed list exits 1 silently.

[`remove`](#remove), [`list`](#list) and [`config dump`](#config) hand over to nothing: what went, what is open, and the configuration are printed on stdout.

Which command each path runs is `internal/work/handoff.go`.
