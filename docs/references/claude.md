# `claude`

A new worktree is handed to the [command](configuration.md#commands) `claude.command` names. The agent renders that one command over the values below, and reads nothing else about the worktree.

## Values

Every value is always rendered, empty where nothing behind the worktree has one. The core fills all but `.Subject`, which the [resolver that answered](systems.md) spells out.

| Value | Is |
|---|---|
| `.Source` | the [system](systems.md) that answered: `beads`, `github`, or `plain` for a name of your own |
| `.ID` | the ticket id, the pull request number, or the name |
| `.Title` | the target's title |
| `.Name` | what the target is retyped as: the ticket id, `pr-<n>`, or the branch |
| `.Dir` | the worktree, which the process has already changed into |
| `.Subject` | the target spelled out, as its resolver reads it |

## The default

The compiled-in default opens a `claude` session named `.Subject`, a ticket's also carrying `/start <id>`. [`work config dump`](cli.md#config) prints it as it resolved. A worktree nothing spells a `.Subject` for renders no command, and is [handed back](cli.md#handoff).
