# Tickets

What `work` asks of a ticket before it creates a worktree for it. The tracker is [`bd`](tools.md); workability is [the flow's convention](../explanation/work-loop.md#workability-is-the-trackers-judgement).

## Vetting

A worktree about to be created for a ticket is vetted first, whatever it will open on. Refused, each naming its reason: a ticket in any status but `open` or `in_progress`, an epic, one with no acceptance criteria, and an open one whose dependencies are not satisfied. `vetBead` in [internal/work/work.go](../../internal/work/work.go) is the list and the messages.

Nothing reaches past it. A worktree that already exists is entered without vetting, and one [asked for by name](cli.md#add) has no ticket to vet.

## Claiming

Creating a ticket's worktree claims it, whatever that worktree opens on. [`--no-claim`](cli.md#switch) declines the claim and nothing else.
