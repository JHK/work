# The interactive launcher

*Epic: work-cli-39a*

## Intent

`work` is [bounded to one moment](../explanation/worktree-per-ticket.md): it resolves a target, hands the terminal over and exits. The screen is what a person meets in that moment, and where the choice of what to hand over belongs.

## Desired end state

Two questions stand between `work` and the handoff: what to work on, and what to hand it to. A `<name>` answers the first, a flag answers the second, and only what neither the arguments nor the [configuration](../references/configuration.md) answer reaches the screen. Each question that does is its own `fzf` invocation, one after the other.

The actions are agent, shell, editor, diff and ask, each one a boolean flag. The first four name a command the configuration defines; ask names the screen, and is the choice between the other four. The screen orders them and says which apply; it does not define what they run. Handing over is the last thing that happens, so no action asks anything of its own — except agent, which may.

Configuration answers the second question standing, once per moment: one key for the worktree just created, another for the one already there. The first is agent, the second is ask.

Agent is a single action however many conversations a worktree carries, and does the right thing for what it finds there: none starts one, one continues it, several are offered as a list.

Nothing is carried per worktree: `--model` and `--effort` go, and the configuration names a model and an effort where either is wanted at all.

## Open questions

1. **What `--agent` can ask of the agent.** Whether the several-conversation case is a list the launcher draws or one `claude` draws itself, and how much of the three cases configuration has to name, follows from behaviour nobody has tested.
