# Configuration

Two optional TOML files override the compiled-in defaults.

| Layer | Location | Scope |
|---|---|---|
| the repository | `.work.toml` at the root of the main checkout, checked in | every clone of that one repository |
| the user | `$XDG_CONFIG_HOME/work/config.toml`, `~/.config/work/config.toml` when unset | every repository on that one machine |

They merge key by key, the repository's winning where both set one. Unset keys fall to `Default` in [internal/config/config.go](../../internal/config/config.go), which also declares them and their types.

## Keys

| Key | Default | Names |
|---|---|---|
| `worktree.directory` | `.worktrees` | a directory inside the repository root, where a new worktree is created |
| `branch.ticket` | `{{.ID}}{{with .Slug}}-{{.}}{{end}}` | the branch a ticket's worktree checks out |
| `branch.pull-request` | `pr-{{.Number}}` | the branch a pull request's worktree checks out, and the name that pull request is retyped as |

Only creating a worktree reads `worktree.directory`. An existing one is entered [where git reports it](../explanation/worktree-per-ticket.md), so changing the value moves nothing and strands nothing.

An unknown key is an error, not a skipped line, and key matching is case-sensitive. Values are validated after the merge and before anything is created, so a value the repository replaces is never judged. Either failure names the file it came from.

## Branch patterns

A `[branch]` value is a [text/template](https://pkg.go.dev/text/template) over the values its kind of target has:

| Key | Values |
|---|---|
| `branch.ticket` | `.ID`, the ticket id; `.Slug`, its title lowercased, dash-joined and cut at 40 characters, empty where that leaves nothing |
| `branch.pull-request` | `.Number`, the pull request number |

Refused at load: a pattern placing no `.ID` or no `.Number`, which is what a worktree is found again by, and one rendering a branch that opens with a dash, which `git` reads as a flag.
