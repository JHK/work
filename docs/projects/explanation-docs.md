# The explanation docs

*Epic: work-cli-4r4*

`docs/explanation/` answers why `work` is shaped the way it is. Two readers arrive there: a person deciding whether the tool is theirs before installing any of it, and an agent that would otherwise infer the model and spend a session acting on the wrong one. What belongs there comes from the brief and from the questions those two arrive with. Reading the build returns implementation detail from a build that is mid-development and never was the authority on intent.

The folder holds three docs. [The scope of `work`](../explanation/scope-of-work.md) stands as written. Nothing else is added: every other question collected sorts out of the type.

## What counts as an explanation

Divio's [explanation quadrant](https://docs.divio.com/documentation-system/explanation/) defines the type, and four of its tests do the sorting.

- **Understanding-oriented.** The reader is away from the keyboard rather than mid-task. A question asked with a terminal open is a how-to's.
- **Wider than one piece of machinery.** A doc defined by a command, a flag, a key or a call is reference material however much reasoning surrounds it.
- **Free to weigh alternatives.** This is the one type where an option that lost earns its space, with the grounds it lost on.
- **Bounded by judgement.** A doc covers what is a reasonable area to hold in mind at once, which is the author's call.

A fifth test is this repository's. **An explanation is a concept, not an integration.** The shell `work` prints, the tracker it speaks to, the picker it needs and the way it reaches the terminal are the state of a build; they change without the model changing. A question whose answer names a tool fails the test however much *why* it carries, and lands in the reference docs or the README instead.

## The set

| Doc | Answers |
|---|---|
| `scope-of-work.md` | What is `work` for, what will it never do, and why does it have two sides? |
| `work-loop.md` | What loop is `work` a verb in, when does work land, and where does the verb stop? |
| `worktree-identity.md` | Why a checkout of its own, and what makes a worktree reachable again? |

The bullets under each doc are its questionnaire: intent the repository cannot supply, to be answered before writing starts.

### `scope-of-work.md`

It answers its set at the right altitude, states the remit as a boundary rather than a feature list, and the systems it names appear as instances of a kind rather than as the set that ships. It is the worked example the other two are held to.

### `work-loop.md`

The loop from capture to close, where `work` sits in it, and what it deliberately leaves to a person. The vocabulary the flow borrows — ticket, worktree, session, skill — is not defined here: a glossary is technical description, and each term earns its place only where the loop turns on something the word does not already carry. It absorbs a question neither of the others answers squarely: **when does work land.** [The scope of `work`](../explanation/scope-of-work.md) says what a worktree coming into being means; the end of the loop wants saying as plainly.

Where the verb stops is that same boundary one scale down, so it belongs here rather than to a doc of its own. Bounding `work` to a single invocation is what settles who owns the work it opens on, and what has to outlive the launch.

Not here: which tracker holds the tickets, what vetting and claiming do, and the flags, key and screen that answer what to open on.

- When has work landed? The branch reaching `main`, the worktree going away, the ticket closing — do those have to be one moment?
- Once it has handed over, is `work` responsible for anything, at either end of the loop?
- Is exiting a principle, or the shape the build happens to have? Was a resident process weighed and rejected?
- Creating opens on the agent without asking and entering asks. Is that asymmetry a claim about the two moments, or a convenience?
- Why is what to open on configurable at all, rather than one good default?
- Why a dependency-aware tracker rather than a queue? What does readiness buy the loop that an ordered list would not?
- Which steps are left to a person because they need judgement, and which only because they are not built?
- Is someone working without a tracker still a user, or is the loop the tool's premise?

### `worktree-identity.md`

Why a session needs a checkout of its own rather than a branch of its own, what identifies a worktree once it has one, and what a worktree matching no ticket is.

Not here: how a branch is matched to a ticket, and what git reports.

- Is one worktree per ticket a rule or a default? Is there work you would want two on?
- The branch carries the identity. Is that the claim, or is the branch standing in for something you would rather key on?
- A worktree with no ticket: a first-class case, or a fallback?
- What becomes of a worktree's identity when the ticket is renamed?

## What sorts out

| Question | Where |
|---|---|
| Which commands, flags and keys exist, and what each does | [the command line](../references/cli.md), [the configuration](../references/configuration.md) |
| Why the command line reads verb-first | [the command line](../references/cli.md) |
| Why the core has two seams, and what earns a configuration key | [the switcher core](worktree-switcher-core.md) until that work lands; the concept is `scope-of-work.md`'s already |
| Why the binary is sourced into the shell rather than run | an integration: [the command line](../references/cli.md#init) |
| Why an absent tool fails its own path rather than degrading it | an integration: [the README](../../README.md) |
| Which tracker, forge, shell and picker are wired | [the README](../../README.md), which holds it as the moment it is |
| How to install it, wire the shell, or reach a worktree | [the README](../../README.md), and `docs/howto/` for anything with steps |
| What must not break when branch matching changes | `docs/rules/`, if it is ever worth enumerating |
| Why a call is ordered as it is | a code comment |

## Out of scope

- `docs/rules/`, `docs/gotchas/` and `docs/howto/`, which stay as they are.
- The reference docs' altitude. Only what moves out of `docs/explanation/` is at stake.
