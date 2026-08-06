# The worktree-per-ticket flow

`work` is one verb in a larger loop, and its shape only makes sense once that loop is visible. This describes the flow and the vocabulary it borrows, neither of which live in this repository.

## The vocabulary

**Bead.** A ticket in [`bd`](https://github.com/steveyegge/beads), a local dependency-aware issue tracker. Its database sits under `.beads/` in the repository and is shared by every worktree of it. A bead carries a status, a type, acceptance criteria and dependencies on other beads, and `bd ready` lists the ones whose dependencies are satisfied.

**Skill.** A set of instructions a coding agent loads on demand, invoked in a session as a slash command. The steps on either side of `work` are usually packaged this way, though which ones exist and what they are called is a matter of how you set your agent up. `work` hardcodes one name per kind of target: `/start <id>` for a bead, `/code-review <n>` for a pull request. Those are the only two it knows.

**Worktree.** A second checkout of the same repository on its own branch, made with `git worktree`. One ticket gets one worktree under `.worktrees/<id>`.

## The loop

Capture, refine, **work**, review, land, close. An idea is filed cheaply as an unrefined bead. Refining fills in acceptance criteria and dependencies until it is workable. `work` opens the place to work on it. Everything from there happens inside the worktree, in an agent session or a shell or both in turn, until the change has landed and the worktree can go.

Every step but one has the work in front of it. `work` is the exception: it runs outside, in the shell, before the worktree necessarily exists. It launches one of two things, a shell or an agent session, which is the small part. Which of the two follows from the worktree: one that has to be created is being created because the work is starting, so it opens on a session, while one that already exists is being returned to, so it opens on a shell. The part that earns it is what it can tell you before you pick: whether the ticket is workable and why not, which sessions the worktree already carries, whether there is a worktree at all. What counts as workable is the tracker's convention rather than this tool's: a ticket that is unrefined, closed, an epic, missing acceptance criteria, or waiting on another ticket is refused with the reason.

That is also where its job ends. Refining a ticket, opening a pull request, merging a branch and closing a bead are judgements made with the work in front of you, so they belong to you and whatever agent you are working with. By then `work` has handed over the terminal and exited.

## Why a worktree for each ticket

A session edits files, runs tests and holds a working tree in a particular state for as long as it lasts. Switching branches underneath it invalidates all of that, so parallel tickets need parallel checkouts rather than one checkout and a branch each.

Once each ticket has a directory, the directory becomes the natural key for everything else. The branch is derived from the bead id and its title. The agent's transcripts are filed by working directory, so the worktree is also what makes prior sessions findable. Entering a ticket is therefore a single question: which directory, and what to run in it.

## What it is scoped to

`work` is bounded to one moment: it runs, hands the terminal to whatever it launched, and exits. Keeping sessions visible and switching between them wants a process that outlives the launch, which is a terminal multiplexer.
