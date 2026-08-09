# The loop `work` is a verb in

`work` is one verb in a larger loop. That loop does not live in this repository, and it is not a premise: the tool moves between the worktrees of a repository whether or not anything files the work.

## The loop

The one it was built against runs:

1. **Capture.** An idea is filed cheaply as an unrefined ticket.
2. **Refine.** The ticket is filled in until there is enough there to judge whether it can be started.
3. **Work.** `work` opens the place to work on it, and everything from there happens inside the worktree, in an agent session or a shell or both in turn.
4. **Review.** The change is read, by whoever or whatever the flow puts on it.
5. **Land.** [Whatever that means here](#landing), after which the worktree can go.
6. **Close.** The ticket is done with, whenever the flow says it is.

## Creation and discovery

`work` runs in the shell, before the worktree may even exist, so one verb covers two things:

1. **Creation.** The identifier that names a fresh worktree usually says what it is for as well. Switching between worktrees the way one switches branches is the plain case, and a worktree handed straight to an agent with the ticket it was named after already in hand is [the same seam](scope-of-work.md#actions) wired for [agentic work](scope-of-work.md#worktrees-and-agentic-work).
2. **Discovery.** A worktree that already exists has to be found before it can be returned to, and a return names the worktree and nothing more, so what it opens on can be answered separately from what a creation opens on.

What earns its keep is what it surfaces: whether there is a worktree at all, and which work is worth opening one for. A tracker that can say which of its tickets are workable can keep vague or unaligned work from becoming a checkout an agent is turned loose in. That judgement is the tracker's, encoded rather than invented, and work it calls unworkable is refused with the reason.

## Landing

Nothing in the tool defines it. The branch reaching `main`, the worktree going away and the ticket closing need not be one moment, and need not fall in that order. A worktree may earn its keep well past the merge, for the experiments that follow it. A ticket may not count as delivered until what it changed is running in production. A change small enough may do without the ceremony altogether, and the tracker someone keeps for their own work need not be the one their team keeps.

So the definition is the person's, and what the tool contributes is the moments to hang it on: a worktree came into being, a worktree went away. Both ends announce, [to whatever is wired at them](scope-of-work.md#actions), and a reading of what landing means can live there as readily in instructions an agent loads when a moment triggers it as in a script.

## Judgement left to a person

Refining a ticket, opening a pull request, merging a branch and closing the ticket are judgements that want the work in front of you. None of them is navigation, so none is ever the tool's, however well an agent might one day be trusted with them. They are not missing features.

Nor is `work` answerable for the work itself, at either end. It gets a person or an agent to the place the work happens and takes the place away again; whether the work got done is not its business. Automation belongs at the moments instead, where a worktree just created can be a good place to resolve a ticket's context and turn an autonomous agent loose in a sandbox. Making that possible is the tool's part in it; doing it is not.

## The bound of one invocation

`work` runs, hands the terminal to whatever it launched, and exits. It holds no list of open sessions. Keeping those visible and flipping between them wants a process that outlives the launch, which is a terminal multiplexer.

Routing back to them is a different job, and that one is the tool's. Where sessions are bound to the directory they ran in, the worktree is what makes the ones belonging to a ticket findable again, and reaching the worktree is reaching them.

Two questions stand between `work` and the handoff: what to work on, and what to hand it to. An identifier answers the first, and standing configuration answers the second unless an invocation overrides it. A worktree opens on one command, so the second is one answer whatever supplied it, and asking a person is itself one of the answers. Handing over is the last thing that happens, so nothing asks anything of its own once it has. That bound is what settles who owns the work it opened on, and what has to outlive the launch to be there when the person comes back.
