# Systems

A system is what `work` reaches for beyond git: a tracker, a forge, a tool, an agent. Each has [a table of its own](configuration.md#systems) with an `enabled` key in it, and none runs until a repository asks for it.

`work` bundles none of them. What a system needs, it looks for on `PATH` when it is asked a question, never when it is built, so a repository that asked for none starts nothing of its own. Where a command is not there, `work` says so once on stderr and asks nothing more of it that run.

## beads

The tracker, over `bd`. It names the bead behind a worktree, [vets and claims it](tickets.md), creates the worktree, and lists the ready beads [the picker](cli.md#the-picker) and [the completion](cli.md#completion) offer. One system on both sides of the worktree: `internal/resolve/beads/` names and creates, `internal/action/beads/` claims, over the one client in `internal/beads/`.

## github

The forge, over `gh`, pinned to origin's URL. It reads origin's open pull requests, drafts included, and their titles, for the picker and the completion. A review's worktree checks out the head git fetches from `pull/<n>/head`, so only the listing is `gh`'s. It is `internal/resolve/github/`.

## mise

The tool trust, over `mise`. It runs `mise trust` in a worktree that was just created, and nothing on the way back into one. It is `internal/action/mise/`.

## claude

The agent, `claude` by default, whatever the [`[claude]` commands](configuration.md#commands) name. A worktree is handed to it, and [the transcripts](claude.md#the-contract) under `~/.claude` are what say which conversations an existing worktree carries. It is `internal/action/claude/`, over the commands in `internal/config/command.go`.
