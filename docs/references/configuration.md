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
| [`claude.command`](../../internal/config/command.go) | the [command](#commands) a new worktree opens on |

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
- a pattern naming `squote`, which is a command's alone

## Opening on a session

`claude.on-creation` names the verbs that hand a worktree they created to [the agent](systems.md#claude). It reaches a worktree once, as that worktree comes into being. A worktree the settings leave `claude` out of is [handed back](cli.md#handoff), and so is one the command renders nothing for.

It falls to `add` and `go` where nothing names it.

Refused at load: a word no verb goes by, and a verb no worktree comes into being under. Only `add`, `carry` and `go` create one.

## Commands

`claude.command` is a command run without a shell, written as one [Go template](https://pkg.go.dev/text/template) over [the values a worktree carries](systems.md#values), in a TOML multiline literal string. It is rendered whole, then read a line at a time: each non-blank line is one argument, trimmed.

A line that is itself a shell script may pipe a value through `squote`, the one filter there is: the value as one word of that shell, in single quotes.

Refused at load:

- a value that is not text
- a block no value renders a command from
- a value the command may not place
- a name no filter goes by

A block that renders to nothing leaves no command to run: the worktree is [handed back](cli.md#handoff), once it is created and the ticket claimed.

The default is in [internal/config/command.go](../../internal/config/command.go), and [what it opens](systems.md#claude) is `claude`.
