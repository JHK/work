# The partition behind the seams

The core runs one sequence over a repository's worktrees. A seam is a place in that sequence where an [integration](../references/integrations.md) takes part, and `work` has two of them. A resolver answers at the first seam, and it turns an identifier into the place to work. An action runs at the second seam, and the core hands it a worktree that exists. One integration can have a half at each seam.

[The scope of `work`](scope-of-work.md) tells you why these two seams exist. This document tells you how the packages divide them.

## The core sets the conditions that an integration must meet

The core decides what happens at a seam. It states what it needs at each step, and an integration that wants to take part meets those conditions. The core declares the interfaces because the core calls them.

Integrations stay independent of each other. They share no state and no logic, and no implementation imports another implementation ([R4](../rules/package-boundaries.md#r4--no-integration-reaches-another-integration)). The interfaces sit in the core, so an implementation imports only what its own work needs.

The set of interfaces can still grow. Every integration meets them in their present form, but a new integration or a new requirement can change them. A change starts in the core, because the core declares them, and every implementation must then meet the new form. Removal has no seam yet.

## A narrow contract lets the core choose how to make the calls

The core asks a resolver a small number of questions, and it can ask them in any order. A REST interface has a similar shape: a small fixed set of verbs rather than a procedure to obey.

The shape is not chosen for elegance. It keeps the two concerns apart. An integration answers a few questions and stays clear of the sequence. The core runs the sequence and stays clear of what an integration does. Each side then stays small enough to hold in mind, and the caller can decide how to make the calls.

Most of the questions are reads, and a read depends on no other answer. Only the creation of a worktree changes anything. The core can therefore issue the reads together. The listing of available work shows the gain. An integration that answers over the network costs about the time of the slowest integration, rather than the sum of all of them.

A richer contract might let an integration choose when the core asks it, or let it read what another integration returned. Either would make the questions depend on each other, so the core could not issue them together. It would also move the decision from the core to the integrations, and only the core can see the whole run.

## The core keeps its logic clear of the concerns of the integrations

The core is responsible for [that sequence](scope-of-work.md#the-remit-is-the-sequence-not-the-integrations-at-its-ends) and nothing more. Everything that the core reaches through a seam is an addition to it. Each addition belongs to the integration that it came from.

[R3](../rules/package-boundaries.md#r3--the-core-reaches-the-vocabulary-git-and-the-settings) holds the two apart. If the core named one implementation, the two concerns would become one concern. The sequence would then carry a tracker's idea of what a claim is. With the rule in place, you can see the boundary in the imports alone.

## What this does not solve

The design rules out dynamic loading. The compiler resolves the seams, and an integration takes part because the wiring puts it in the binary. An integration also cannot state its position in the chain. The wiring chooses the order, and each integration has to be indifferent to it.

Git sits inside the core and not behind a seam of its own. A switcher without git would not be a smaller switcher.

Removal is the open edge. A worktree goes away, but the core asks no resolver and runs no action. The integration that named the worktree therefore never hears that the work stopped. A seam for removal is possible, and nothing has decided against one.
