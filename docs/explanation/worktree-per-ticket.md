# The worktree-per-ticket flow

[One verb of the loop](work-loop.md) is the tool's: opening the place a ticket is worked. This is what makes that place the ticket's own.

## One worktree per ticket

A session edits files, runs tests and holds a working tree in a particular state for as long as it lasts. Switching branches underneath it invalidates all of that, so parallel tickets need parallel checkouts rather than one checkout and a branch each.

## The branch as the key

Each worktree checks out a branch named for its ticket: `pr-<n>` for a pull request, the ticket's name ahead of a slug of its title otherwise, and a name given outright where there is no ticket. `git worktree list` reports that branch alongside the path, so every worktree answers for its ticket wherever on disk it lives. Keying on the directory instead loses every worktree created by hand or moved since: a ticket whose worktree cannot be seen reads as fresh, and working it opens a second checkout of the same branch.

Names and title slugs both contain dashes, so a branch is recognised by the names the tracker knows rather than parsed apart. A branch a longer known name matches is that ticket's, so `one` never opens `one-two`'s worktree; where several still match one ticket, the branches settle it rather than git's listing order. A branch matching no ticket is still a worktree, offered under its branch name and entered with a shell, and one can be asked for outright: a name of your own is a branch of your own, with nothing to vet or claim. That keeps the tracker off the path that finds worktrees at all: when it will not answer, the labels and the ready tickets go and the worktrees stay.

Sessions are filed by working directory, so the path git reports is what makes a worktree's prior ones findable.
