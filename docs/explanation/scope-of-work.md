# The scope of `work`

`work` exists to shorten the distance between deciding to work on something and standing in the place that work happens. The place is a git worktree, and most of what makes it the right one is knowable from a ticket, a pull request or a name.

## The remit is the sequence, not the systems at its ends

The sequence runs over the worktrees of a repository: which ones exist, which one is being asked for, making it exist where it does not, and opening it. A tracker can resolve work to be opened in the worktree, and creating one can trigger an agent session as an action.

## Isolation is half of what an agent needs

A worktree is a checkout of its own, so two of them never contend for the same files or the same branch. For a person switching context that is a convenience. For an agent it is closer to a requirement, since a session editing a tree another session is editing produces a change neither can be judged by, and agentic work is worth running more than one at a time.

The other half is the agent arriving knowing what it is there for, which is the same ticket or pull request the worktree was named after. Creating the worktree and opening the session with that context already in hand is one motion, and `work` was built for it.

## The identifier already knows what the worktree needs

A worktree needs a branch, a name and a place to fork from, and something has to decide all three. The identifier a person is already holding came from the tracker that filed the work or the forge that hosts the review, and everything the worktree needs sits at the other end of it. A resolver is what asks. It speaks to one of those systems, turns an identifier into the worktree belonging to it, and carries along what came back.

That is convenience where a plain name would have done, and it is the only source of context where a plain name would not. It is also where a system's own vocabulary stays: what claiming a ticket means, what fetching a pull request's head means, what a branch is named after. A plain name is resolved too, by the resolver whose answer is that a name is a name.

## None of the work in a worktree is `work`'s

An action is what the worktree drains into: the agent session that edits it, the shell someone types in, the editor it opens, the diff read before a review. Agentic work is the case this earns the most, since the session and the context it needs arrive together.

Opening is not the whole of it. A worktree coming into being says work has started, which is the moment a tracker wants to hear about and the moment a repository's tooling wants to act on, mise trusting the config the tools inside the worktree will look for. An action is handed that worktree and decides for itself whether it has anything to do.

## The core calls both sides without naming them

Everything ships in the one binary, and neither resolvers nor actions are a wall around the tool. What sits behind them is what would otherwise be welded into the sequence, where a branch per tracker, per forge and per editor lands in the path every invocation runs through, and the file that should change least changes for everything the loop picks up.

A system `work` was never built against therefore arrives under its own name beside the ones already there, rather than displacing them. [The command line](../references/cli.md) and [the configuration](../references/configuration.md) carry the set that ships.
