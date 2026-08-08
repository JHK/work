# The worktree-per-ticket flow

`work` is one verb in a larger loop. Neither that loop nor the vocabulary it borrows lives in this repository.

## The vocabulary

**Ticket.** A unit of work with a name, a status, and enough detail to judge whether it can be started: acceptance criteria and dependencies on other tickets. Which tracker holds it, and the vocabulary that comes with it, is the flow's business rather than this tool's. [`bd`](https://github.com/steveyegge/beads) is the instance `work` is built against, a local dependency-aware tracker whose database sits under `.beads/`, shared by every worktree of the repository. There a ticket is a bead, and `bd ready` lists the ones whose dependencies are satisfied.

**Worktree.** A second checkout of the same repository on its own branch, made with `git worktree`. One ticket gets one worktree, created under a [configured directory](../references/configuration.md) and used wherever it sits.

**Session.** One conversation with a coding agent, filed by the directory it ran in, so a worktree accumulates its own history.

**Skill.** A set of instructions a coding agent loads on demand, invoked in a session as a slash command. The steps on either side of `work` are usually packaged this way. A fresh worktree opens on the one that suits its target: here, `/start <id>` for a bead and `/code-review <n>` for a pull request.

## The loop

Capture, refine, **work**, review, land, close. An idea is filed cheaply as an unrefined ticket. Refining fills in acceptance criteria and dependencies until it is workable. `work` opens the place to work on it. Everything from there happens inside the worktree, in an agent session or a shell or both in turn, until the change has landed and the worktree can go.

`work` runs in the shell, before the worktree may even exist. Creating one means work is starting, so it gets the launcher; entering one that exists means returning to work already under way, so it gets a shell. Asking for a shell outright overrides that.

What earns its keep is what it surfaces: whether there is a worktree at all, and whether the ticket can be worked. Workability is a convention `work` encodes rather than invents, and a ticket that fails it is refused with the reason.

Refining a ticket, opening a pull request, merging a branch and closing the ticket are judgements made with the work in front of you, so they belong to you and your agent.

## Why a worktree for each ticket

A session edits files, runs tests and holds a working tree in a particular state for as long as it lasts. Switching branches underneath it invalidates all of that, so parallel tickets need parallel checkouts rather than one checkout and a branch each.

## Why the branch is the key

Each worktree checks out a branch named for its ticket: `pr-<n>` for a pull request, the ticket's name ahead of a slug of its title otherwise, and a name given outright where there is no ticket. `git worktree list` reports that branch alongside the path, so every worktree answers for its ticket wherever on disk it lives. Keying on the directory instead loses every worktree created by hand or moved since: a ticket whose worktree cannot be seen reads as fresh, and working it opens a second checkout of the same branch.

Names and title slugs both contain dashes, so a branch is recognised by the names the tracker knows rather than parsed apart. A branch a longer known name matches is that ticket's, so `one` never opens `one-two`'s worktree; where several still match one ticket, the branches settle it rather than git's listing order. A branch matching no ticket is still a worktree, offered under its branch name and entered with a shell, and one can be asked for outright: a name of your own is a branch of your own, with nothing to vet or claim. That keeps the tracker off the path that finds worktrees at all: when it will not answer, the labels and the ready tickets go and the worktrees stay.

Sessions are filed by working directory, so the path git reports is what makes a worktree's prior ones findable.

## What it is scoped to

`work` is bounded to one moment: it runs, hands the terminal to whatever it launched, and exits. Keeping sessions visible and switching between them wants a process that outlives the launch, which is a terminal multiplexer.
