# The loop `work` is a verb in

`work` is one verb in a larger loop. That loop does not live in this repository, and it is not a premise: the tool moves between the worktrees of a repository whether or not anything files the work.

## The loop runs from capture to close

The one it was built against runs:

1. **Capture.** An idea is filed cheaply as an unrefined ticket.
2. **Refine.** The ticket is filled in until there is enough to judge whether it can be started.
3. **Work.** `work` opens the place to work on it, and everything from there happens inside the worktree, in an agent session or a shell or both in turn.
4. **Review.** The change is read, by whoever or whatever the flow puts on it.
5. **Land.** [Whatever that means here](#landing-is-the-persons-definition), after which the worktree can go.
6. **Close.** The ticket is done with, whenever the flow says it is.

## One verb covers creation and discovery

`work` runs in the shell, before the worktree may even exist, so the verb has to cover both:

- **Creation.** The identifier that names a fresh worktree usually says what it is for as well. The plain case is switching between worktrees the way one switches branches. Hand a fresh worktree straight to an agent, with the ticket it was named after already in hand, and that is [the same handing over](scope-of-work.md#none-of-the-work-in-a-worktree-is-works) wired for [agentic work](scope-of-work.md#isolation-is-half-of-what-an-agent-needs).
- **Discovery.** A worktree that already exists has to be found before it can be returned to. A return names the worktree and nothing more, so what it opens on is answered separately from what a creation opens on.

The verb earns its keep by what it surfaces: whether there is a worktree at all, and which work is worth opening one for.

## Workability is the tracker's judgement

A tracker that can say which of its tickets are workable keeps vague or unaligned work from becoming a checkout an agent is turned loose in. That judgement is the tracker's, encoded rather than invented. Work the tracker calls unworkable is refused with the reason.

## Landing is the person's definition

Nothing in the tool defines landing. The branch reaching `main`, the worktree going away and the ticket closing need not be one moment, and need not fall in that order. A worktree may earn its keep well past the merge, for the experiments that follow it. A ticket may not count as delivered until what it changed is running in production. A change small enough may do without the ceremony altogether. The tracker someone keeps for their own work need not be the one their team keeps.

What the tool contributes instead is a moment to hang a definition on: a worktree came into being. That end announces, [to whatever is wired at it](scope-of-work.md#none-of-the-work-in-a-worktree-is-works), and a reading of what starting work means can live in that wiring. A script is one form; instructions an agent loads when the moment triggers it are another.

## What is left out is not a missing feature

Refining a ticket, opening a pull request, merging a branch and closing the ticket are judgements that want the work in front of you. None of them is navigation, so none is ever the tool's, however well an agent might one day be trusted with them.

Nor is `work` answerable for the work itself, at either end. It gets a person or an agent to the place the work happens and takes the place away again; whether the work got done is not its business. Automation belongs at that moment instead, where a worktree just created can be a good place to resolve a ticket's context and turn an autonomous agent loose in a sandbox. Making that possible is the tool's part; doing it is not.

## The invocation ends at the handoff

`work` runs, hands over, and exits. It holds no list of open sessions. Keeping those visible and flipping between them wants a process that outlives the launch, which is a terminal multiplexer.

Routing back to them is a different job, and that one is the tool's. Where sessions are bound to the directory they ran in, the worktree is what makes a ticket's sessions findable again. Reaching the worktree is reaching them.

Two questions stand between `work` and the handoff: what to work on, and what to hand it to. An identifier answers the first, and standing configuration answers the second. A worktree opens on one action. Handing over is the last thing that happens, so nothing asks anything of its own once it has. That bound settles who owns the work the invocation opened on, and what has to outlive the launch to be there when the person comes back.
