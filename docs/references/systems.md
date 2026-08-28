# Systems

A system is what `work` reaches for beyond git: a tracker, a forge, a tool, an agent. Each goes by one name in [the settings' list](configuration.md#systems), and none runs until you name it.

`work` bundles none of them: what a system needs, it looks for on `PATH` when it is asked a question. A command that is not there is refused. A refusal reaches stderr once, naming the command that was run and what it answered with.

## beads

The tracker, over `bd`. It names the bead behind a worktree, answering only for an id `bd` lists, [vets and claims it](tickets.md), creates the worktree, and lists the ready beads [the picker](cli.md#the-picker) and [the completion](cli.md#completion) offer. One system on both sides of the worktree: `internal/resolve/beads/` names and creates, `internal/action/beads/` claims, over the one client in `internal/beads/`.

## github

The forge, over `gh`, pinned to origin's URL. It reads origin's open pull requests, drafts included, and their titles, for the picker and the completion. A review's worktree checks out the head git fetches from `pull/<n>/head`, so only the listing is `gh`'s. It is `internal/resolve/github/`.

## mise

The tool trust, over `mise`. It runs `mise trust` in a worktree that was just created. It is `internal/action/mise/`.

## claude

The agent, `claude` by default, whatever the [`[claude]` commands](configuration.md#commands) name. A ticket's or a pull request's worktree is handed to it, where [the settings name the verb that created it](configuration.md#opening-on-a-session). It is `internal/action/claude/`, over the commands in `internal/config/command.go`.
