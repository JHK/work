# The vocabulary

What holds the shared words apart. What each word means is its doc comment in [`internal/worktree/`](../../internal/worktree/), and why they sit in a package of their own is [the partition behind the seams](../explanation/seam-partition.md#the-compiler-decided-where-the-shared-words-go-and-the-design-did-not).

## R7 — A word of the vocabulary cannot stand in for another

Every concept `internal/worktree` names is a defined type of its own. A value of one cannot reach a field or a parameter that wants another: a repository root cannot reach a worktree path, and a branch cannot reach a name. A word of the vocabulary carries its type wherever it crosses a package. A word that is git's, bd's or the settings' own stays text: a refspec, a ticket status, a branch pattern.

Text reaching work from a settings value, a flag or a tool's output is converted by the package that reads it. A type says which concept a value is, never that the value is well formed.

*Enforced by:* the compiler.
