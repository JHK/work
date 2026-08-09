# The scope of `work`

`work` exists to shorten the distance between deciding to work on something and standing in the place that work happens. The place is a git worktree, and most of what makes it the right one is knowable from a ticket, a pull request or a name.

## The remit

The worktrees of a repository, and the sequence around them: which ones exist, which one is being asked for, making it exist where it does not, and opening it. That sequence holds whatever tracker files the work and whatever the worktree is opened in, which is what makes it the tool's own.

## Worktrees and agentic work

A worktree is a checkout of its own, so two of them never contend for the same files or the same branch. For a person switching context that is a convenience. For an agent it is closer to a requirement, since a session editing a tree another session is editing produces a change neither can be judged by, and agentic work is worth running more than one at a time.

Isolation is only half of it. An agent also needs to arrive knowing what it is there for, which is the same ticket or pull request the worktree was named after. Creating the worktree and opening the session with that context already in hand is one motion, and `work` was built for it.

## Resolvers

A worktree needs a branch, a name and a place to fork from, and something has to decide all three. The identifier a person is already holding usually carries the decision: it came from the tracker that filed the work or the forge that hosts the review, and everything the worktree needs sits at the other end of it. A resolver is what asks. It speaks to one of those systems, turns an identifier into the worktree belonging to it, and carries along what came back.

That is convenience where a plain name would have done, and it is the only source of context where a plain name would not. It is also where a system's own vocabulary stays: what claiming a ticket means, what fetching a pull request's head means, what a branch is named after. A plain name is resolved too, by the resolver whose answer is that a name is a name.

## Actions

A worktree is where work happens, and none of the work is `work`'s. An action is what the worktree drains into: the agent session that edits it, the shell someone types in, the editor it opens, the diff read before a review. Agentic work is the case this earns the most, since the session and the context it needs arrive together.

Opening is not the whole of it. A worktree coming into being says work has started and a worktree going away says it has finished, and those are the moments a tracker wants to hear about and the moments a repository's tooling wants to act on, mise trusting the config the tools inside the worktree will look for. An action is handed the worktree along with which moment it is, and decides for itself whether it has anything to do.

## Adding to either side

Everything ships in the one binary, and neither side is a wall around the tool. What sits behind them is what would otherwise be welded into the sequence, where a branch per tracker, per forge and per editor lands in the path every invocation runs through, and the file that should change least changes for everything the loop picks up.

The core calls resolvers and actions without naming any of them, so a system `work` was never built against arrives under its own name beside the ones already there rather than displacing them. [The command line](../references/cli.md) and [the configuration](../references/configuration.md) carry the set that ships.
