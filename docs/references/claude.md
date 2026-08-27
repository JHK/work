# `claude`

A worktree is handed to the [command](configuration.md#commands) the `[claude]` keys name, `claude` by default.

The compiled-in defaults open a session named after whatever the worktree was made for: a ticket's id and title, a pull request's number, or the worktree's own name. A ticket's also carries `/start <id>` as the prompt. [`work config dump`](cli.md#config) prints them as they resolved.
