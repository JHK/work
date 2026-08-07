# Configuration

*Epic: work-cli-8lo*

## Intent

The choices `work` makes on a person's behalf are defaults, not facts about the binary. A repository states where its worktrees go and how its branches are named; a person states what a worktree opens on.

This settles the keys, the file each belongs in, and how a value reaches the code, so the children wiring them do not each answer that.

## The surface

`[worktree]` and `[branch]` hold what a worktree is made of; `[agent]` and `[open]` hold commands.

```toml
[worktree]
directory = ".worktrees"

[branch]
ticket = "{{.ID}}{{with .Slug}}-{{.}}{{end}}"
pull-request = "pr-{{.Number}}"

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

[open]
shell = ["{{.Shell}}"]
editor = ["{{.Editor}}", "{{.Dir}}"]
```

These render what the constants produce, less two departures. A pull request opens on a bare named session, dropping the auto permission mode and the `/code-review` prompt `SessionLaunch` sets today, because the reviewer chooses what to run. Resuming carries no session id, for [the reason below](#returning-to-a-conversation).

*Ticket* is the [vocabulary](../explanation/worktree-per-ticket.md) the keys are named in; *bead* is the `bd` instance of it, and stays inside the code. `worktree.directory` is a path inside the repository, and only a new worktree needs one, since an existing one is entered where git reports it. `bd worktree create` adds a ticket worktree's path to the repository's `.gitignore`; a pull request's is made with `git worktree add`, which does not, so a configured directory has to be ignored by the repository itself.

`[agent]` is named for what its commands reach rather than for what each does, because they have to agree: the agent that starts a conversation is the one that returns to it, so they move together when the agent changes. `[open]`'s two owe each other nothing, so that table is named for the verb.

### Where the keys are read from

Two files, because the halves of the surface belong to different people: the repository's is checked in and everyone cloning gets it, the user's follows them across repositories.

| Layer | Location |
|---|---|
| the repository | `.work.toml` at the repository root, checked in |
| the user | `$XDG_CONFIG_HOME/work/config.toml`, `~/.config/work/config.toml` when unset |
| defaults | compiled in |

Highest first, merged key by key.

Either file may set any key, the repository's winning where both name one, so nothing in the loader depends on which table a key sits in. What a value states is still checked wherever it came from: `worktree.directory` has to resolve inside the repository, and a rendered branch may not open with a dash, which `git` and `bd` would read as a flag.

`[agent]` is included, so a clone carries the command `work` runs on the machine that cloned it. Restricting which tables a repository may set would buy a threat model a person who has read the repository already handles, at a table-by-table exception to the one rule the rest of the surface has.

No flag is a layer, because none of them sets a key: `--model` and `--effort` are values a command may place, and `--shell` and `--editor` choose which command runs. Nor is the environment. `$SHELL`, `$VISUAL` and `$EDITOR` reach the templates as values, `XDG_CONFIG_HOME` locates the user's file, and none of them overrides a key.

### One key, one command

Each key is a whole command for one thing, so no template chooses between modes: `agent.resume-session` returns to a conversation and never has to also describe starting one. Only starting varies by target kind. Nothing requires the command to be an agent; a plain shell command simply leaves no conversation to return to.

These commands are what an existing worktree offers and what a new one opens on. A [screen](interactive-launcher.md) picks between them, and `--shell` and `--editor`, exclusive with each other, name one from the command line. `open.shell` serves every existing worktree, [whatever its branch names](../explanation/worktree-per-ticket.md). `open.editor` on a new worktree still vets and claims; `--shell` keeps the escape hatch it has today and does neither.

### Returning to a conversation

`agent.resume-session` needs no identifier. A session is filed by the directory it ran in, so `claude --continue` resumes the conversation the worktree holds, and `claude --resume` is the agent's own picker for older ones.

What a worktree carries is still named on entry, from `internal/sessions`, and that reading is not a setting.

### How a template is written

`[branch]` values are [text/template](https://pkg.go.dev/text/template) patterns rendering one string. A command is a list of them, one per argv element, and an element rendering to nothing is dropped, so an optional flag is written as the single element `{{with .Value}}--flag={{.}}{{end}}` rather than a pair: `--flag={{.Value}}` renders `--flag=` and is passed on. A *first* element rendering to nothing leaves no command, and the handoff is refused naming the setting and the value that came out empty.

| Value | Rendered by | Is |
|---|---|---|
| `.Name` | every command | what the target is retyped as: the ticket id, `pr-<n>`, or the branch |
| `.Dir` | every command | the worktree, which the process has already changed into |
| `.Model`, `.Effort` | every command | what `--model` and `--effort` were given, empty when they were not |
| `.ID` | `branch.ticket`, `agent.start-ticket` | the ticket id |
| `.Slug` | `branch.ticket` | the title slugged as `slug` in `internal/work/provision.go`, empty for a title that slugs to nothing |
| `.Number` | `branch.pull-request`, `agent.start-pull-request` | the pull request number |
| `.Title` | `agent.start-ticket` | the ticket title |
| `.Shell` | `open.shell` | `$SHELL`, or `/bin/sh` |
| `.Editor` | `open.editor` | `$VISUAL`, else `$EDITOR` |

A pull request's title is `gh`'s to give and only [the picker asks it](../references/cli.md); an existing worktree's would cost a `bd` call on the path that deliberately makes none.

### What the branch template costs

The branch is the one setting read back, because a target's worktree is [the one checked out on its branch](../explanation/worktree-per-ticket.md), wherever git reports it.

A ticket's branch is attributed by rendering `branch.ticket` for each id `bd` knows, with every other value matched as a wildcard and `{{with}}` matching either of its arms. `feature/{{.ID}}-{{.Slug}}` attributes `feature/bd-42-anything` to `bd-42`, so a prefix costs nothing and a ticket retitled after its worktree exists still finds it. The longest matching id wins, so `bd-1` does not claim `bd-12`'s branch.

Nothing lists pull requests on that path, so `branch.pull-request` is matched with `.Number` captured as digits instead, and a capture that does not render back to the branch it came from is no match: `pr-007` stands for no worktree, since `Resolve` canonicalises it to `pr-7`, whose worktree is elsewhere. That rendered branch doubles as the name a person retypes.

A template that does not name its kind's identifier attributes nothing and is refused at load.

### The loader

`work.Open` reads both files into a config struct that `work.Env` carries, so a front end passes on options alone. A key the struct does not know is refused by name, so a typo does not silently do nothing.

Decoding is `github.com/BurntSushi/toml`, whose `MetaData.Undecoded` gives that refusal and which requires no further module. Rejected:

- **viper**, which answers discovery, layering and env vars out of the box, but drags in a module tree of its own for keys read once, against cobra as this module's only direct dependency.
- **JSON**, which needs no dependency at all, but takes no comments in a file people hand-edit.

## Out of scope

- Where session history is read from. It stays Claude Code's transcript layout, keyed on the working directory.
- Per-worktree model and effort memory, which is the [screen](interactive-launcher.md)'s state rather than a default.
- A `work config` subcommand for reading or writing either file.
- Worktree directories outside the repository root.
- Swapping the tracker or the forge. `bd` names, vets and claims, and `gh` answers for pull requests, each through several calls; reaching either through configured commands is the shape `[agent]` sets, at a size of its own.
