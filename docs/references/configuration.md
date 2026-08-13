# Configuration

Two optional TOML files override the compiled-in defaults.

| Layer | Location | Scope |
|---|---|---|
| the repository | `.work.toml` at the root of the main checkout, checked in | every clone of that one repository |
| the user | `~/.config/work/config.toml`, `$XDG_CONFIG_HOME/work/config.toml` where that variable names an absolute path | every repository on that one machine |

They merge key by key, the repository's winning where both set one. Unset keys fall to `Default` in [internal/config/](../../internal/config/), which also declares them and their types. What the merge produced on a given machine, in a given repository, is [`work config dump`](cli.md#config); the user's file is what [`work config edit`](cli.md#config) opens, creating it where it is not there yet.

Refused at load, each naming the file it came from:

- an unknown key
- a key whose case does not match
- a table or an [action](#actions) under a name a rename replaced, refused with the name it goes by now

Values are validated after the merge and before anything is created.

## Keys

| Key | Names |
|---|---|
| [`<system>.enabled`](../../internal/config/config.go) | whether that [system](#systems) runs at all |
| [`worktree.directory`](../../internal/config/config.go) | a directory inside the main checkout, where a new worktree is created |
| [`branch.ticket`](../../internal/config/config.go) | the branch a ticket's worktree checks out |
| [`branch.pull-request`](../../internal/config/config.go) | the branch a pull request's worktree checks out, and the name that pull request is retyped as |
| [`action.create`](../../internal/config/action.go) | the [action](#actions) a newly created worktree opens on |
| [`action.enter`](../../internal/config/action.go) | the [action](#actions) an existing worktree opens on |
| [`claude.start-ticket`](../../internal/config/command.go) | the [command](#commands) a ticket's new worktree opens on |
| [`claude.start-pull-request`](../../internal/config/command.go) | the [command](#commands) a pull request's new worktree opens on |
| [`claude.start-session`](../../internal/config/command.go) | the [command](#commands) a worktree opens on with no ticket and no conversation to name another |
| [`claude.resume-session`](../../internal/config/command.go) | the [command](#commands) that returns to the conversation a worktree carries |
| [`open.shell`](../../internal/config/command.go) | the [command](#commands) the `shell` action hands the worktree to, `--shell` included |

Only creating a worktree reads `worktree.directory`. An existing one is entered [where git reports it](../explanation/worktree-identity.md#the-branch-is-the-identity-not-the-path).

## Systems

What `work` runs on is worktrees, and no file can take that away: a worktree is listed, entered and removed, `work add` makes a place of a name of your own, and `shell` is there to open on. Everything reached beyond git is a [system](systems.md), and a repository turns on the ones it works with:

```toml
[beads]
enabled = true

[claude]
enabled = true
# claude.* is read whether or not this is
```

One name carries a system wherever it appears: the table its own keys sit in, its rows, its flags, and the key a refusal names. One that both names places and acts on them, as `beads` does in resolving a ticket and claiming it, is turned on for both by the one key.

## Branch patterns

A `[branch]` value is a [Go template](https://pkg.go.dev/text/template) over the values its kind of target has:

| Key | Values |
|---|---|
| `branch.ticket` | `.ID`, the ticket id; `.Slug`, its title lowercased, dash-joined and cut at 40 characters, empty where that leaves nothing |
| `branch.pull-request` | `.Number`, the pull request number |

Refused at load:

- a pattern placing no `.ID` or no `.Number`
- a pattern rendering a branch that opens with a dash

## Actions

An `[action]` value is one of the actions [a flag](cli.md#open-on-flags) names, anything else being refused at load, and a flag naming one wins over both keys for that invocation:

| Value | Hands the worktree to |
|---|---|
| `claude` | what [`--claude`](cli.md#open-on-flags) hands it to |
| `shell` | `open.shell` |

## Commands

A `[claude]` or `[open]` value is the argv of a command run without a shell, one [Go template](https://pkg.go.dev/text/template) per element. An element rendering to nothing is dropped from the argv.

| Value | Rendered by | Is |
|---|---|---|
| `.Name` | every command | what the target is retyped as: the ticket id, `pr-<n>`, or the branch |
| `.Dir` | every command | the worktree, which the process has already changed into |
| `.ID`, `.Title` | `claude.start-ticket` | the ticket id and its title |
| `.Number` | `claude.start-pull-request` | the pull request number |
| `.Session` | `claude.resume-session` | the conversation the worktree carries, empty where it carries several |
| `.Shell` | `open.shell` | `$SHELL`, else `/bin/sh` |

An empty `.Session` drops the element that placed it, so `claude.resume-session` reaches the one [conversation](claude.md#the-contract) outright and the agent's own list where there are several. No id is ever asked of a person.

Refused at load:

- an empty list
- a value the key does not have

A command whose first element renders to nothing is refused at the handoff instead, once the worktree is created and the ticket claimed.

The defaults are in [internal/config/command.go](../../internal/config/command.go), and rest on [`claude`'s own behaviour](claude.md).
