# Integrations

An integration is what `work` reaches for beyond git: a tracker, a forge, a tool, an agent. Each goes by one name in [the settings' list](configuration.md#integrations), and none runs until you name it.

`work` bundles none of them: what an integration needs, it looks for on `PATH` when it is asked a question. A command that is not there is refused. A refusal reaches stderr once, naming the command that was run and what it answered with.

Each integration takes part at one or both of [the two seams](../explanation/seam-partition.md).

## Resolvers

A resolver answers before a worktree exists, turning an identifier into the place to work.

### beads

The tracker, over `bd`. It names the bead behind a worktree, answering only for an id `bd` lists, creates the worktree, and lists the ready beads [the picker](cli.md#the-picker) and [the completion](cli.md#completion) offer. It fills [`.Subject`](#values) with `<id>: <title>`.

It also reaches into the action space: creating a ticket's worktree claims that ticket.

#### Vetting

A ticket is vetted before its worktree is created, and one that cannot be worked is refused in a line naming why. Refused, in the order read:

- a ticket already closed
- a deferred ticket, in words naming `/refine`
- a ticket in any status but `open` or `in_progress`, named in the refusal
- a ticket carrying no acceptance criteria, in words naming `/refine`
- an open ticket the tracker does not list as ready, refused as blocked by an open dependency

An epic is held to the first alone: refining it is what its worktree is for.

Workability is [the flow's convention](../explanation/work-loop.md#workability-is-the-trackers-judgement).

#### Claiming

Creating a ticket's worktree claims it, whatever that worktree opens on.

### github

The forge, over `gh`, pinned to origin's URL. It reads origin's open pull requests, drafts included, and their titles, for [the picker](cli.md#the-picker) and [the completion](cli.md#completion). A review's worktree checks out `pr-<n>`, on the head git fetches from `pull/<n>/head`, so only the listing is `gh`'s. It fills [`.Subject`](#values) with `PR #<n>: <title>`, the number alone where no listing named a title.

## Actions

An action runs on the worktree that now exists, and is handed the values below.

### Values

Every value is always rendered, empty where nothing behind the worktree has one. The core fills all but `.Subject`, which the [resolver that answered](#resolvers) spells out.

| Value | Is |
|---|---|
| `.Source` | the integration that answered: `beads`, `github`, or `git` where none did |
| `.ID` | the ticket id, the pull request number, or the name |
| `.Title` | the target's title |
| `.Name` | what the target is retyped as: the ticket id, `pr-<n>`, or the branch |
| `.Dir` | the worktree, which the process has already changed into |
| `.Subject` | the target spelled out, as its resolver reads it |

### mise

The tool trust, over `mise`. It runs `mise trust` in a worktree that was just created.

### claude

The agent, whatever the [`claude.command`](configuration.md#commands) setting names. It renders that one command over [the values](#values).

A worktree `work` creates is handed to the agent once, as it comes into being. One that was already open is [handed back](cli.md#handoff), and so is one the command renders nothing for.
