# Command line

One command, an optional identifier, and flags. `work --help` prints the flag list; what follows is what the flags do to the repository, which the help text does not say.

## Identifiers

| Argument | Resolves to |
|---|---|
| `bd-42`, any name matching `[A-Za-z0-9][A-Za-z0-9._-]*` | that bead |
| `7`, `007` | pull request 7 |
| `pr-7` | pull request 7, so a worktree name can be retyped |
| `https://host/owner/repo/pull/7`, with any trailing path | pull request 7 |
| omitted | an fzf list of the repository's worktrees and its ready beads |

Leading dashes, path separators, `.` and `..` are refused: the identifier becomes a directory under `.worktrees/` and an argument to `bd`, `gh` and `git`.

A pull request number is read against the current repository, whatever host the URL names.

## What each invocation does

Provisioning is idempotent, so any of these re-enters a worktree that already exists.

| Invocation | Vets | Creates the worktree | Claims the bead | Hands off to |
|---|---|---|---|---|
| `work <id>` | when it will claim | yes | yes | `$SHELL` |
| `work <id> --start` | yes | yes | yes | `claude` on `/start <id>` |
| `work <id> --shell` | no | yes | no | `$SHELL` |
| `work <pr>` | not applicable | yes | not applicable | `$SHELL` |
| `work` | as for the target picked | after confirmation | as for the target picked | as for the target picked |

Vetting is [bead-workflow policy](../explanation/worktree-per-ticket.md): a deferred, closed, epic, criteria-less or dependency-blocked bead is refused, and the message says which. It guards the claim, so the paths that do not claim do not vet.

Confirmation guards a target reached through the picker, where backing out should leave the repository as it was. A named target is acted on as given.

## Flags

| Flag | Effect |
|---|---|
| `--start` | vet, claim, and launch a session on `/start <id>`; beads only |
| `--shell` | enter without claiming; mutually exclusive with `--start` |
| `--model <m>` | passed to the launched session; requires `--start` |
| `--effort <e>` | passed to the launched session as `low`, `medium`, `high`, `xhigh` or `max`; requires `--start` |

## Handoff

`work` changes into the worktree and replaces itself with the session, so the calling shell keeps its own directory. A failure to enter is one line on stderr and exit 1; a declined confirmation or a dismissed picker exits 1 silently.

The session command is built in `internal/work/handoff.go`, and the same builder renders the resume lines printed on entry.
