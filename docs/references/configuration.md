# Configuration

One optional TOML file overrides the compiled-in defaults: `~/.config/work/config.toml`, or `$XDG_CONFIG_HOME/work/config.toml` where that variable names an absolute path. It follows you to every repository on that machine, and no repository carries settings of its own.

A key it does not name falls to the compiled-in default. What the settings resolved to on a given machine is [`work config dump`](cli.md#config); the file itself is what [`work config edit`](cli.md#config) opens, creating it where it is not there yet.

`work` reads the keys below, each spelled exactly as the table spells it. Values are validated once the file is read, before anything is created.

## Keys

| Key | Names |
|---|---|
| `integrations` | the [integrations](#integrations) that run |
| `worktree.directory` | the [directory](#worktree-directory) a new worktree is created in |
| `beads.branch` | the branch a ticket's worktree checks out |
| `claude.command` | the [command](#commands) [the agent](integrations.md#claude) runs |

Only creating a worktree reads `worktree.directory`. An existing one is entered [where git reports it](../explanation/worktree-identity.md#the-branch-is-the-identity-not-the-path).

## Worktree directory

`worktree.directory` is a [Go template](https://pkg.go.dev/text/template) over `.Repo`, the main checkout's path. The compiled-in default is `{{.Repo}}/.worktrees`.

## Integrations

What `work` runs on is worktrees, and no file can take that away: a worktree is listed, entered and removed, and `work add` makes a place of a name of your own. Everything reached beyond git is an [integration](integrations.md), and you name the ones you work with:

```toml
integrations = ["beads", "claude"]
# claude.* is read whether or not claude is named
```

One that both names places and acts on them, as `beads` does in resolving a ticket and claiming it, is turned on for both by the one name.

## Branch patterns

`beads.branch` is a [Go template](https://pkg.go.dev/text/template) over `.ID`, the ticket id, and `.Slug`, its title lowercased, dash-joined and cut at 40 characters, empty where that leaves nothing.

A pattern places `.ID`, which is how a ticket's worktree is found again. A ticket without a slug renders its own branch, which may not open with a dash.

## Commands

`claude.command` is a command run without a shell, written as one [Go template](https://pkg.go.dev/text/template) over [the values a worktree carries](integrations.md#values), in a TOML multiline literal string. It is rendered whole, then read a line at a time: each non-blank line is one argument, trimmed.

A line that is itself a shell script may pipe a value through `squote`, the one filter there is: the value as one word of that shell, in single quotes.

A command renders at least one argument for some worktree.

The default opens `claude` on the worktree's subject, and renders nothing for a worktree that has none. [`work config dump`](cli.md#config) prints it as written.
