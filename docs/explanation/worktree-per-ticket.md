# The worktree-per-ticket flow

`work` is one verb in a larger loop, and its shape only makes sense once that loop is visible. This describes the flow and the vocabulary it borrows, neither of which live in this repository.

## The vocabulary

**Bead.** A ticket in [`bd`](https://github.com/steveyegge/beads), a local dependency-aware issue tracker. Its database sits under `.beads/` in the repository and is shared by every worktree of it. A bead carries a status, a type, acceptance criteria and dependencies on other beads, and `bd ready` lists the ones whose dependencies are satisfied.

**Skill.** A set of instructions a coding agent loads on demand, invoked in a session as a slash command. The steps on either side of `work` are usually packaged this way, though which ones exist and what they are called is a matter of how you set your agent up. `work` hardcodes one name per kind of target: `/start <id>` for a bead, `/code-review <n>` for a pull request. Those are the only two it knows.

**Worktree.** A second checkout of the same repository on its own branch, made with `git worktree`. One ticket gets one worktree, created under `.worktrees/` and used wherever it sits.

## The loop

Capture, refine, **work**, review, land, close. An idea is filed cheaply as an unrefined bead. Refining fills in acceptance criteria and dependencies until it is workable. `work` opens the place to work on it. Everything from there happens inside the worktree, in an agent session or a shell or both in turn, until the change has landed and the worktree can go.

Every step but one has the work in front of it. `work` is the exception: it runs outside, in the shell, before the worktree necessarily exists. It launches one of two things, a shell or an agent session, which is the small part. Which of the two follows from the worktree: one that has to be created is being created because the work is starting, so it opens on a session, while one that already exists is being returned to, so it opens on a shell. The part that earns it is what it can tell you before you pick: whether the ticket is workable and why not, which sessions the worktree already carries, whether there is a worktree at all. What counts as workable is the tracker's convention rather than this tool's: a ticket that is unrefined, closed, an epic, missing acceptance criteria, or waiting on another ticket is refused with the reason.

That is also where its job ends. Refining a ticket, opening a pull request, merging a branch and closing a bead are judgements made with the work in front of you, so they belong to you and whatever agent you are working with. By then `work` has handed over the terminal and exited.

## Why a worktree for each ticket

A session edits files, runs tests and holds a working tree in a particular state for as long as it lasts. Switching branches underneath it invalidates all of that, so parallel tickets need parallel checkouts rather than one checkout and a branch each.

## Why the branch is the key

Each worktree checks out a branch named for its ticket: `pr-<n>` for a pull request, the bead id ahead of a slug of its title for a bead. That branch is what identifies the worktree afterwards, and `git worktree list` reports it alongside the path, so every worktree the repository has is discoverable and answers for its ticket wherever on disk it lives. Keying on the directory instead loses every worktree created by hand or moved since: a ticket whose worktree cannot be seen reads as fresh, and working it opens a second checkout of the same branch.

Ids and title slugs both contain dashes, so a branch is recognised by the ids the tracker knows rather than parsed apart. A branch matching no ticket is still a worktree, offered under its branch name and entered with a shell. That keeps the tracker off the path that finds worktrees at all: when it will not answer, the labels and the ready tickets go, the worktrees stay.

The directory keeps one job. The agent's transcripts are filed by working directory, so the path git reports is what makes a worktree's prior sessions findable.

## What it is scoped to

`work` is bounded to one moment: it runs, hands the terminal to whatever it launched, and exits. Keeping sessions visible and switching between them wants a process that outlives the launch, which is a terminal multiplexer.

Two worktrees whose branches both match one ticket are not disambiguated: one of them wins, and which is not defined.
