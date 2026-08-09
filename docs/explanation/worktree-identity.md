# A worktree's identity

Which worktree an ask reaches, and whether one exists to reach at all, both rest on what a worktree is taken to be.

## The unit is the checkout, not the branch

A branch each would be cheaper. One checkout, a branch per piece of work, and moving between them is a single command. It loses because a piece of work in flight is more than its commits: a build tree, a test run part way through, a shell sitting somewhere in the directory, an agent holding a picture of what is where. Changing the branch underneath all of that invalidates it, and what survives has to be stashed and restored by hand.

A branch each therefore serialises work that has no reason to be serial, which is why [agentic work](scope-of-work.md#worktrees-and-agentic-work) needs worktrees rather than merely liking them.

## What a worktree is for is not the tool's call

A ticket each is one way to use them, and the one agentic work tends toward: a ticket cut small enough to finish in a session is a checkout cut to the same size. Several tickets worked in one worktree is another way. So is a worktree opened for an experiment, an investigation or a comparison that no ticket describes.

None of them is a rule or a default. A tool that picked one would have to refuse the others, and refusing a checkout because no ticket justifies it is a process argument dressed as navigation. `work` is a verb inside whichever process a person keeps.

## The branch is the identity, not the path

Every worktree has exactly one branch checked out, and that branch is what the worktree is. A branch nothing has checked out is outside the remit.

A path is chosen by whoever made the worktree and changed by any later move, while git reports every worktree with the branch it has out. So an ask after a piece of work finds its checkout wherever it lives.

A directory key would lose every worktree made by hand or moved since, and a lost worktree is worse than an unfound one. The work reads as unstarted, so the ask goes on to create. Git refuses to check a branch out twice, so it fails on a worktree that was there all along. [The command line](../references/cli.md#identifiers) holds how an identifier reaches a branch.

## A worktree with no ticket is first-class

Ask for a name of your own and you get a branch of your own, with [nothing to vet and nothing to claim](../references/tickets.md). Not everything worth a checkout is worth a ticket, and a tool that only opened the worktrees a tracker had blessed would be that tracker's front end rather than a switcher.

That also keeps the tracker off the path that finds worktrees at all. When it will not answer, what goes is [the judgement of which work is ready](work-loop.md#workability-is-the-trackers-judgement), not the worktrees.

## The link to a ticket is inferred, never recorded

Where a ticket did name a worktree, nothing stores the pairing. The branch carries what the tracker calls the work, and finding the worktree again is recognising it there. A stored pairing would be a second thing to keep true, in a file no clone shares, and it would go wrong silently. A name goes wrong visibly.

The cost is that the link is a heuristic. It holds up because the identifier a tracker issues is stable while the words around it are not: retitling a ticket leaves the recognition untouched, and renaming a worktree moves its directory and its branch together. What no heuristic survives is a branch renamed out from under the tool by hand. The worktree is still there and still usable, and its ticket now reads as unworked.

## What this does not solve

Identity travelling with the branch does not carry everything a worktree accumulates. An agent's prior conversations are filed under the directory they ran in, so a worktree that moves keeps its identity and loses its history. [Reaching the worktree is how those are reached again](work-loop.md#the-invocation-ends-at-the-handoff), which leaves the path load-bearing in the one place the model says it is not.
