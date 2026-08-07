# Configuration

Two optional TOML files override the compiled-in defaults.

| Layer | Location | Scope |
|---|---|---|
| the repository | `.work.toml` at the root of the main checkout, checked in | every clone of that one repository |
| the user | `$XDG_CONFIG_HOME/work/config.toml`, `~/.config/work/config.toml` when unset | every repository on that one machine |

They merge key by key, the repository's winning where both set one. Unset keys fall to `Default` in [internal/config/](../../internal/config/), which also declares them and their types.

## Keys

| Key | Default | Names |
|---|---|---|
| `worktree.directory` | `.worktrees` | a directory inside the repository root, where a new worktree is created |
| `branch.ticket` | `{{.ID}}{{with .Slug}}-{{.}}{{end}}` | the branch a ticket's worktree checks out |
| `branch.pull-request` | `pr-{{.Number}}` | the branch a pull request's worktree checks out, and the name that pull request is retyped as |
| `agent.start-ticket` | [below](#commands) | the command a ticket's new worktree opens on |
| `agent.start-pull-request` | [below](#commands) | the command a pull request's new worktree opens on |
| `agent.resume-session` | [below](#commands) | the command that returns to the conversation a worktree carries |

Only creating a worktree reads `worktree.directory`. An existing one is entered [where git reports it](../explanation/worktree-per-ticket.md), so changing the value moves nothing and strands nothing.

`.work.toml` may set `[worktree]` and `[branch]` alone, and naming any other table fails at load: cloning a repository does not decide what runs on the machine that cloned it.

An unknown key is an error, not a skipped line, and key matching is case-sensitive. Values are validated after the merge and before anything is created, so a value the repository replaces is never judged. Either failure names the file it came from.

## Branch patterns

A `[branch]` value is a [Go template](https://pkg.go.dev/text/template) over the values its kind of target has:

| Key | Values |
|---|---|
| `branch.ticket` | `.ID`, the ticket id; `.Slug`, its title lowercased, dash-joined and cut at 40 characters, empty where that leaves nothing |
| `branch.pull-request` | `.Number`, the pull request number |

Refused at load: a pattern placing no `.ID` or no `.Number`, which is what a worktree is found again by, and one rendering a branch that opens with a dash, which `git` reads as a flag.

## Commands

An `[agent]` value is the argv of a command run without a shell, one [Go template](https://pkg.go.dev/text/template) per element. An element rendering to nothing is dropped from the argv.

| Value | Rendered by | Is |
|---|---|---|
| `.Name` | every command | what the target is retyped as: the ticket id, `pr-<n>`, or the branch |
| `.Dir` | every command | the worktree, which the process has already changed into |
| `.Model`, `.Effort` | every command | what `--model` and `--effort` were given, empty where they were not |
| `.ID`, `.Title` | `agent.start-ticket` | the ticket id and its title |
| `.Number` | `agent.start-pull-request` | the pull request number |

`agent.resume-session` names no session: a session is filed by the directory it ran in.

Refused at load: an empty list, and a value the key does not have. A command whose first element renders to nothing is refused at the handoff instead, once the worktree is created and the ticket claimed.

The defaults:

```toml
[agent]
start-ticket = [
  "claude", "--permission-mode", "auto",
  "--name={{.ID}}: {{.Title}}",
  "{{with .Model}}--model={{.}}{{end}}",
  "{{with .Effort}}--effort={{.}}{{end}}",
  "/start {{.ID}}",
]
start-pull-request = [
  "claude", "--name=PR #{{.Number}}",
  "{{with .Model}}--model={{.}}{{end}}",
  "{{with .Effort}}--effort={{.}}{{end}}",
]
resume-session = [
  "claude", "--permission-mode", "auto", "--continue",
  "{{with .Model}}--model={{.}}{{end}}",
  "{{with .Effort}}--effort={{.}}{{end}}",
]
```
