# Systems

A system is what `work` reaches for beyond git: a tracker, a forge, a tool, an agent. Each goes by one name in [the settings' list](configuration.md#systems), and none runs until you name it.

`work` bundles none of them: what a system needs, it looks for on `PATH` when it is asked a question. A command that is not there is refused. A refusal reaches stderr once, naming the command that was run and what it answered with.

## beads

The tracker, over `bd`. It names the bead behind a worktree, answering only for an id `bd` lists, [vets and claims it](tickets.md), creates the worktree, and lists the ready beads [the picker](cli.md#the-picker) and [the completion](cli.md#completion) offer. It fills [`.Subject`](claude.md#values) with `<id>: <title>`. One system on both sides of the worktree: `internal/resolve/beads/` names and creates, `internal/action/beads/` claims, over the one client in `internal/beads/`.

## github

The forge, over `gh`, pinned to origin's URL. It reads origin's open pull requests, drafts included, and their titles, for the picker and the completion. A review's worktree checks out the head git fetches from `pull/<n>/head`, so only the listing is `gh`'s. It fills [`.Subject`](claude.md#values) with `PR #<n>: <title>`, the number alone where no listing named a title. It is `internal/resolve/github/`.

## mise

The tool trust, over `mise`. It runs `mise trust` in a worktree that was just created. It is `internal/action/mise/`.

## claude

[The agent](claude.md), `claude` by default, whatever the [`claude.command`](configuration.md#commands) setting names. A new worktree is handed to it, where [the settings name the verb that created it](configuration.md#opening-on-a-session). It is `internal/action/claude/`, over the command in `internal/config/command.go`.
