# Configuration

One optional TOML file overrides the compiled-in defaults: `~/.config/work/config.toml`, or `$XDG_CONFIG_HOME/work/config.toml` where that variable names an absolute path. It follows you to every repository on that machine, and no repository carries settings of its own.

A key it does not name falls to `Default` in [internal/config/](../../internal/config/), which also declares the keys and their types. What that resolved to on a given machine is [`work config dump`](cli.md#config); the file itself is what [`work config edit`](cli.md#config) opens, creating it where it is not there yet.

Refused at load, naming the file:

- an unknown key
- a key whose case does not match
- a table under a name a rename replaced, refused with the name it goes by now

Values are validated once the file is read, before anything is created.

## Keys

| Key | Names |
|---|---|
| [`systems`](../../internal/config/systems.go) | the [systems](#systems) that run |
| [`worktree.directory`](../../internal/config/config.go) | a directory inside the main checkout, where a new worktree is created |
| [`branch.ticket`](../../internal/config/config.go) | the branch a ticket's worktree checks out |
| [`branch.pull-request`](../../internal/config/config.go) | the branch a pull request's worktree checks out, and the name that pull request is retyped as |
| [`claude.on-creation`](../../internal/config/verbs.go) | the verbs whose creations [open a session](#opening-on-a-session) |
| [`claude.start-ticket`](../../internal/config/command.go) | the [command](#commands) a ticket's new worktree opens on |
| [`claude.start-pull-request`](../../internal/config/command.go) | the [command](#commands) a pull request's new worktree opens on |
| [`claude.start-session`](../../internal/config/command.go) | the [command](#commands) a worktree opens on where nothing names a ticket or a pull request |

Only creating a worktree reads `worktree.directory`. An existing one is entered [where git reports it](../explanation/worktree-identity.md#the-branch-is-the-identity-not-the-path).

The value is read as a path at load, and where it leads at creation. A directory resolving out of the repository is refused before the worktree is made, whether the value or a symlink standing where it names takes it there.

## Systems

What `work` runs on is worktrees, and no file can take that away: a worktree is listed, entered and removed, `work add` makes a place of a name of your own, and `shell` is there to open on. Everything reached beyond git is a [system](systems.md), and you name the ones you work with:

```toml
systems = ["beads", "claude"]
# claude.* is read whether or not claude is named
```

One that both names places and acts on them, as `beads` does in resolving a ticket and claiming it, is turned on for both by the one name.

Refused at load: a name no system goes by, refused with the names there are.

## Branch patterns

A `[branch]` value is a [Go template](https://pkg.go.dev/text/template) over the values its kind of target has:

| Key | Values |
|---|---|
| `branch.ticket` | `.ID`, the ticket id; `.Slug`, its title lowercased, dash-joined and cut at 40 characters, empty where that leaves nothing |
| `branch.pull-request` | `.Number`, the pull request number |

Refused at load:

- a pattern placing no `.ID` or no `.Number`
- a pattern rendering a branch that opens with a dash

## Opening on a session

`claude.on-creation` names the verbs that hand a worktree they created to [the agent](claude.md). It reaches a worktree once, as that worktree comes into being. A worktree the settings leave `claude` out of is [handed back](cli.md#handoff).

It falls to `add` and `go` where nothing names it.

Refused at load: a word no verb goes by, and a verb no worktree comes into being under. Only `add`, `carry` and `go` create one.

## Commands

A `[claude]` value is the argv of a command run without a shell, one [Go template](https://pkg.go.dev/text/template) per element. An element rendering to nothing is dropped from the argv.

| Value | Rendered by | Is |
|---|---|---|
| `.Name` | every command | what the target is retyped as: the ticket id, `pr-<n>`, or the branch |
| `.Dir` | every command | the worktree, which the process has already changed into |
| `.ID`, `.Title` | `claude.start-ticket` | the ticket id and its title |
| `.Number` | `claude.start-pull-request` | the pull request number |

Refused at load:

- an empty list
- a value the key does not have

A command whose first element renders to nothing is refused at the handoff instead, once the worktree is created and the ticket claimed.

The defaults are in [internal/config/command.go](../../internal/config/command.go), and [what they open](claude.md) is `claude`.
