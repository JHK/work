# work

`work` covers the *work* step of a worktree-per-ticket flow: capture, refine, **work**, review, land, close. It turns a ticket, a pull request, or an already open worktree into a place to work in: the worktree, made to exist and be usable, and a coding agent inside it, whether a fresh session, a resumed one, or just a shell. Finding that target is half the job, whether it is named outright or picked from the worktrees a repo has open and the sessions they carry.

It is the boundary between the shell and the agent: the point where a ticket becomes a place with a branch, a checkout, and a session history.

That one step is the whole remit. `work` runs, hands the terminal to the session, and is gone, leaving the steps on either side to the tools that own them; anything it can do interactively it can also do from a flag, so a script reaches all of it too. [The flow it serves](docs/explanation/worktree-per-ticket.md) sets out where the rest of the steps live.

## Install

```
mise run install
```

Installing the binary is the whole setup. `work` provisions, then replaces itself with the session, so the shell it was launched from is waiting where you left it once the session ends, and stays valid after that worktree is merged away.

## Requires

Each tool is reached for by one path, and only that path pays when it is missing.

| Tool | Needed for |
|---|---|
| `git` | everything |
| [`bd`](https://github.com/steveyegge/beads) | creating a bead's worktree, claiming it, and vetting it |
| `fzf` | `work` with no argument |
| `claude` | `--start` |
| `mise` | trusting a new worktree's configs, so its first session starts clean |

## Use

```
work                                          # pick from the repo's worktrees and ready beads
work <id|pr|url>                              # a shell in the target's worktree, creating it if needed
work <id> --start [--model m] [--effort e]    # claim the bead and launch a session on /start
work <id> --shell                             # a shell for a look around; the bead stays as it is
```

Every flag and what it touches: [the command line](docs/references/cli.md). Every build and install task: [mise tasks](docs/references/mise-tasks.md).
