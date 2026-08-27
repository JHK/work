# Command line

## Commands

| Command | Description |
|---|---|
| [`work go [<identifier>]`](#go) | reaches the worktree an [identifier](#identifiers) names, creating it if there is none |
| [`work switch [<identifier>]`](#switch) | enters the worktree an identifier already has |
| [`work add [<identifier>]`](#add) | creates the worktree an identifier has none of, and opens it |
| [`work remove [<name>]`](#remove) | removes a worktree and deletes the branch it had checked out |
| [`work move [<name>] [<destination>]`](#move) | moves a worktree and renames the branch it has checked out with it |
| [`work list`](#list) | prints the worktrees open |
| [`work init <shell>`](#init) | prints the shell integration |
| [`work config dump`](#config) | prints the effective [configuration](configuration.md) and where each key came from |
| [`work config edit`](#config) | opens the [settings file](configuration.md) in `$VISUAL`, else `$EDITOR` |
| `work [<identifier>]` | dispatched to [`go`](#go) |

`--help`, `--log-level` and `--version` are the root's own:

| Flag | Description |
|---|---|
| `-h`, `--help` | prints a command's help, and is the one shorthand of the three |
| `--log-level <level>` | says what `work` reached for, at `warn`, `info` or `debug` |
| `--version` | prints [the version](mise-tasks.md#version-stamping), in or out of a repository |

### go

`work go <identifier>` enters the worktree the [identifier](#identifiers) names, creating it where there is none, forked from the `HEAD` of the directory the shell is standing in.

With no identifier, [the picker](#the-picker) stands in for one.

A [system](systems.md) can add to either form.

### switch

`work switch <identifier>` enters the worktree the [identifier](#identifiers) already has. One with no worktree open is refused.

With no identifier, [the picker](#the-picker) stands in for one.

### add

`work add <identifier>` creates the worktree the [identifier](#identifiers) has none of, forked from the `HEAD` of the directory the shell is standing in, and hands it to what [`action.create`](configuration.md#actions) names or an [open-on flag](#open-on-flags) named.

The working state moves with it: the invoking checkout is left carrying none of it and the new worktree carries the changes, what was staged still staged. Untracked files travel; ignored files stay put. Changes that do not apply cleanly are kept in the stash, whatever part of them the new worktree already carries.

The identifier resolves as anywhere else, and a name no [system](systems.md) answers for becomes a branch spelled exactly as it is. An identifier that already has a worktree is refused, and so are a name of your own whose branch is already there and a directory already sitting where the worktree would go.

With no identifier, [the picker](#the-picker) stands in for one.

### remove

`work remove <name>` removes the worktree git reports for that [identifier](#identifiers) and deletes the branch it had checked out. An identifier with no worktree open is refused.

With no name, [the picker](#the-picker) stands in for one.

Only a clean worktree can be removed, an unclean one with `--force`. The main checkout cannot be removed, and neither can the worktree the shell is standing in.

### move

`work move <name> <destination>` moves the worktree git reports for that [identifier](#identifiers) and renames the branch it has checked out to the destination's last element. Both land or neither does. An identifier with no worktree open is refused, and so are a destination already there and a branch name already taken, each before anything moves.

A destination spelled as a bare name lands the worktree beside where it sits; one carrying a path separator is a path, absolute or read from the directory work was invoked in. The last element is held to the [naming rule](#identifiers) either way.

With no destination, [the prompt](#the-prompt) asks for one. With no name either, [the picker](#the-picker) stands in for the name first.

The main checkout cannot be moved, and neither can the worktree the shell is standing in.

### list

`work list` prints the worktrees git reports, one per line on stdout: the branch each has checked out, or its directory where it is detached.

### init

`work init <shell>` prints the [function](#the-function) and the [completion](#completion), both installed by the one line that sources it:

- bash: `source <(work init bash)` in `.bashrc`
- fish: `work init fish | source` in `config.fish`
- zsh: `source <(work init zsh)` in `.zshrc`

### config

`work config dump` prints what the [settings](configuration.md) resolved to, as TOML work loads back, each key under a comment naming where it came from: the compiled-in default, or the file's path. [Patterns and commands](configuration.md#branch-patterns) print as they are written, rendering one needing a target.

A configuration work would refuse to load is refused here too, with nothing printed.

`work config edit` opens [the file](configuration.md) in `$VISUAL`, else `$EDITOR`. The file and the directory it sits in are created where neither is there yet, so an editor that creates neither still opens.

Where neither variable names an editor, the invocation is refused before anything is created. The settings are the same wherever the shell stands, so both verbs run anywhere.

## Arguments

### Identifiers

| Argument | Resolves to |
|---|---|
| `feature/x` | the worktree already open under that name, ahead of everything below |
| `bd-42` | that bead, for an id [the tracker](systems.md#beads) lists |
| `7`, `007` | pull request 7 |
| [`pr-7`](configuration.md) | pull request 7, so a worktree name can be retyped |
| `https://host/owner/repo/pull/7`, with any trailing path | pull request 7 |

A pull request number is read against the current repository, whatever host the URL names.

The first row is `work`'s own; every other is one a [system](systems.md) adds. A name nothing answers for is refused; [`add`](#add) is the verb that makes a worktree of a name of your own.

Refused, here and wherever else a worktree is named: leading dashes, path separators, `.` and `..`.

The worktree an identifier reaches is the one checked out on its [branch](configuration.md#keys), [wherever git reports it](../explanation/worktree-identity.md#the-branch-is-the-identity-not-the-path).

A ticket's branch is recognised, never read apart: the [pattern](configuration.md#branch-patterns) is matched against the whole branch with an id filled in and the rest wildcarded, and of the ids the tracker listed the longest owner takes it. Where it lists none, nothing is longer and the pattern alone settles it.

Where several worktrees match one ticket, the shortest branch takes it, ties settled on the name, so it is never git's listing order.

### The picker

An fzf list standing in for the argument a verb was not given. It and [the prompt](#the-prompt) are the only things that need `fzf`: name the arguments instead, and every verb runs without it.

What a verb puts up is what it can act on:

| Verb | Offers | Leaves out |
|---|---|---|
| [`go`](#go) | every worktree open, and the tickets and pull requests with none | the one you are standing in |
| [`switch`](#switch) | the worktrees open | the one you are standing in |
| [`add`](#add) | the tickets and pull requests with no worktree yet | nothing |
| [`remove`](#remove) | the worktrees open | the one you are standing in, and the main checkout |
| [`move`](#move) | the worktrees open | the one you are standing in, and the main checkout |

A listing left with no rows is one line on stderr naming what there is nothing of.

A worktree row is one git reports with a working tree to reach, read off its branch, and offered under that branch where no [system](systems.md) names it.

Beside the worktrees, a system can add rows of its own and the titles on them.

### The prompt

One question with the answer already typed into it, standing in for the destination [`move`](#move) was not given. What it carries is the name the worktree goes by now. An answer left empty, and an interruption, are the invocation cancelled.

### Completion

What [`init`](#init)'s integration offers at each position:

| Position | Offers |
|---|---|
| the first word | the commands |
| [`config`](#config)'s sub-verb | `dump` and `edit` |
| [`go`](#go), [`switch`](#switch), [`add`](#add), [`remove`](#remove) and [`move`](#move)'s identifier or name | what [that verb's picker](#the-picker) offers |
| [`init`](#init)'s shell | the shells it prints |

In bash the completion needs the `bash-completion` package sourced, cobra's script calling into it. In zsh it needs `compinit` already run.

## Opening

### Open-on flags

What a worktree is handed to, for one invocation, ahead of what [`action.create` and `action.enter`](configuration.md#actions) name. [`go`](#go), [`switch`](#switch) and [`add`](#add) carry the set, and naming two of it in one invocation is refused. A flag an action used to answer to is refused with the one it answers to now.

[`go`](#go) and [`add`](#add) carry a second set beside it: one flag per action that spells one, calling that action off for the invocation; [`switch`](#switch) carries none of them and refuses them.

Both sets are spelled from the [settings](configuration.md#systems) as read, before any repository is opened, so a [system](systems.md) that is off has no flag at all and `--help` lists what this machine switched on.

| Flag | Hands the worktree to |
|---|---|
| `--claude` | the [`claude` command](configuration.md#keys) the moment names: its launcher where the worktree was just created, else the one [its conversations](claude.md#the-contract) name |
| `--shell` | nothing: the worktree is [handed back](#handoff), one just created included |

## Handoff

`work` changes into the worktree and execs the command it opens on, so the calling shell keeps its own directory. [`config edit`](#config) hands over the same way, into the directory its file sits in.

The `shell` action hands the worktree back instead: the path goes into the file [the function](#the-function) named, or onto stdout where nothing named one. A terminal reading that path is told on stderr, in one line naming [`work init`](#init), that the integration is not sourced; the invocation still exits 0.

A dismissed list exits 1 silently.

[`remove`](#remove), [`move`](#move), [`list`](#list) and [`config dump`](#config) hand over to nothing: what went, what moved and what its branch became, what is open, and the configuration are printed on stdout.

Which command each path runs is the actions under `internal/action/`, `config edit` naming its own in `internal/config/`; replacing the process with it is `internal/worktree/`, and the function and the file it names are `internal/shim/`.

### The function

`work` in a shell that sourced [`init`](#init) is a shell function calling the binary. It names a temporary file in `WORK_CD_FILE` and changes into the path the binary wrote there.
