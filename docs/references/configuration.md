# Configuration

One optional TOML file overrides the compiled-in defaults: `~/.config/work/config.toml`, or `$XDG_CONFIG_HOME/work/config.toml` where that variable names an absolute path. It follows you to every repository on that machine, and no repository carries settings of its own.

A key it does not name falls to `Default` in [internal/config/](../../internal/config/), which also declares the keys and their types. What that resolved to on a given machine is [`work config dump`](cli.md#config); the file itself is what [`work config edit`](cli.md#config) opens, creating it where it is not there yet.

Refused at load, naming the file:

- an unknown key
- a key whose case does not match
- a table or an [action](#actions) under a name a rename replaced, refused with the name it goes by now

Values are validated once the file is read, before anything is created.

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

Only creating a worktree reads `worktree.directory`. An existing one is entered [where git reports it](../explanation/worktree-identity.md#the-branch-is-the-identity-not-the-path).

The value is read as a path at load, and where it leads at creation. A directory resolving out of the repository is refused before the worktree is made, whether the value or a symlink standing where it names takes it there.

## Systems

What `work` runs on is worktrees, and no file can take that away: a worktree is listed, entered and removed, `work add` makes a place of a name of your own, and `shell` is there to open on. Everything reached beyond git is a [system](systems.md), and you turn on the ones you work with:

```toml
[beads]
enabled = true

[claude]
enabled = true
# claude.* is read whether or not this is
```

One name carries a system wherever it appears: the table its own keys sit in, its rows, and its flags. One that both names places and acts on them, as `beads` does in resolving a ticket and claiming it, is turned on for both by the one key.

A system left out is wired nowhere: it spells no [flag](cli.md#open-on-flags) and offers no rows, and a file naming its [action](#actions) is refused at load.

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

An `[action]` value is one of the actions that are on here, which is what an [open-on flag](cli.md#open-on-flags) names. Both keys fall to `shell` where nothing names one, and a flag naming an action wins over both for that invocation.

| Value | On where | Hands the worktree to |
|---|---|---|
| `claude` | `claude.enabled = true` | what [`--claude`](cli.md#open-on-flags) hands it to |
| `shell` | always | nothing: the worktree is [handed back](cli.md#handoff) |

Refused at load: a name no action goes by, an action of a [system](#systems) that is off among them. An action that only runs when a worktree comes into being, such as `beads` or `mise`, is not an `[action]` value: it is [declined by a flag](cli.md#open-on-flags).

## Commands

A `[claude]` value is the argv of a command run without a shell, one [Go template](https://pkg.go.dev/text/template) per element. An element rendering to nothing is dropped from the argv.

| Value | Rendered by | Is |
|---|---|---|
| `.Name` | every command | what the target is retyped as: the ticket id, `pr-<n>`, or the branch |
| `.Dir` | every command | the worktree, which the process has already changed into |
| `.ID`, `.Title` | `claude.start-ticket` | the ticket id and its title |
| `.Number` | `claude.start-pull-request` | the pull request number |
| `.Session` | `claude.resume-session` | the conversation the worktree carries, empty where it carries several |

An empty `.Session` drops the element that placed it, so `claude.resume-session` reaches the one [conversation](claude.md#the-contract) outright and the agent's own list where there are several.

Refused at load:

- an empty list
- a value the key does not have

A command whose first element renders to nothing is refused at the handoff instead, once the worktree is created and the ticket claimed.

The defaults are in [internal/config/command.go](../../internal/config/command.go), and rest on [`claude`'s own behaviour](claude.md).
