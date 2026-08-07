# The interactive launcher

## Intent

`work` is [bounded to one moment](../explanation/worktree-per-ticket.md): it resolves a target, hands the terminal over and exits. The screen is what a person meets in that moment, and where the choice of what to hand over belongs.

## Current state

`work` with no argument shells out to `fzf` from `internal/cli/pick.go`, which returns one candidate and nothing else. What happens next [follows from existence alone](../references/cli.md), and a worktree that already exists is handed to the shell without a word, because there is nowhere to offer anything else.

## Desired end state

Picking a target that has no worktree creates it and hands over without a second question, which is the fast path and stays one. Picking one that exists asks what to hand it to, because that is the question existence was standing in for.

The actions are the [commands the configuration names](../references/configuration.md): start, resume, shell, editor. The screen orders them and says which apply; it does not define what they run. Resume is one action however many conversations a worktree carries.

The list leads with the most recently opened worktree and marks it, so a worktree opened moments ago in another terminal costs no typing to get back to.

## Open questions

- **What orders the list.** Recency is what makes the worktree just created lead it, and `git worktree list` reports no times. Whether that is the directory's own timestamp, the branch's, or the newest session filed against it is unsettled.
- **Where per-worktree model and effort live.** Reopening a worktree with what it was last opened with is state the screen owns rather than a default, and no store for it exists.
- **What draws it.** A second `fzf` invocation for the second question is the cheap answer; a real screen is the one that can lead with the worktree just created and mark it.
