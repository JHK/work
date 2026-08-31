# The partition behind the seams

The core runs one sequence over a repository's worktrees. A seam is a place in that sequence where a [system](../references/systems.md) takes part, and `work` has two of them. A resolver answers at the first seam, and it turns an identifier into the place to work. An action runs at the second seam, and the core hands it a worktree that exists. One system can have a half at each seam.

[The scope of `work`](scope-of-work.md) tells you why these two seams exist. This document tells you how the packages divide them.

## The core sets the conditions that a system must meet

The core decides what happens at a seam. It states what it needs at each step, and a system that wants to take part meets those conditions. The core declares the interfaces because the core calls them.

Systems stay independent of each other. They share no state and no logic, and no implementation imports another implementation ([R4](../rules/package-boundaries.md#r4--no-system-reaches-another-system)). The interfaces sit in the core, so an implementation imports only what its own work needs.

The set of interfaces can still grow. Every system meets them in their present form, but a new system or a new requirement can change them. A change starts in the core, because the core declares them, and every implementation must then meet the new form. Removal has no seam yet.

## A narrow contract lets the core choose how to make the calls

The core asks a resolver a small number of questions, and it can ask them in any order. A REST interface has a similar shape: a small fixed set of verbs rather than a procedure to obey.

The shape is not chosen for elegance. It keeps the two concerns apart. A system answers a few questions and stays clear of the sequence. The core runs the sequence and stays clear of what a system does. Each side then stays small enough to hold in mind, and the caller can decide how to make the calls.

Most of the questions are reads, and a read depends on no other answer. Only the creation of a worktree changes anything. The core can therefore issue the reads together. The listing of available work shows the gain. A system that answers over the network costs about the time of the slowest system, rather than the sum of all of them.

A richer contract might let a system choose when the core asks it, or let it read what another system returned. Either would make the questions depend on each other, so the core could not issue them together. It would also move the decision from the core to the systems, and only the core can see the whole run.

## The core keeps its logic clear of the concerns of the systems

The core is responsible for [that sequence](scope-of-work.md#the-remit-is-the-sequence-not-the-systems-at-its-ends) and nothing more. Everything that the core reaches through a seam is an addition to it. Each addition belongs to the system that it came from.

[R3](../rules/package-boundaries.md#r3--the-core-reaches-the-vocabulary-git-and-the-settings) holds the two apart. If the core named one implementation, the two concerns would become one concern. The sequence would then carry a tracker's idea of what a claim is. With the rule in place, you can see the boundary in the imports alone.

## The compiler decided where the shared words go, and the design did not

The core, the two sides of the seams, and the settings share a small vocabulary. It names a place, a worktree that exists, the command that a worktree opens on, and the values that command renders with. A package of its own declares these words. That package imports only the standard library.

No part of the partition depends on that position. These words are the core's conditions, and they would sit with the other conditions in the core. But the core imports the settings, and the settings use these words too. They fill a repository's commands with the values that a worktree supplies.

If the core held the words, the core would import a package that imports the core. The compiler refuses this, so the words sit in a leaf package. The position carries no intent.

## What this does not solve

The design rules out dynamic loading. The compiler resolves the seams, and a system takes part because the wiring puts it in the binary. A system also cannot state its position in the chain. The wiring chooses the order, and each system has to be indifferent to it.

Git sits inside the core and not behind a seam of its own. A switcher without git would not be a smaller switcher.

Removal is the open edge. A worktree goes away, but the core asks no resolver and runs no action. The system that named the worktree therefore never hears that the work stopped. A seam for removal is possible, and nothing has decided against one.
