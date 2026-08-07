# Command line

One command, an optional identifier and flags, and `init` for the shell integration. `work --help` sketches the forms and lists the flags; what follows is what each invocation does to the repository, which the help text does not say.

## Identifiers

| Argument | Resolves to |
|---|---|
| `feature/x` | the worktree already open under that name, ahead of everything below |
| `bd-42` | that bead, for any name matching `[A-Za-z0-9][A-Za-z0-9._-]*` |
| `7`, `007` | pull request 7 |
| [`pr-7`](configuration.md) | pull request 7, so a worktree name can be retyped |
| `https://host/owner/repo/pull/7`, with any trailing path | pull request 7 |
| omitted | an fzf list of the repository's worktrees, open pull requests and ready beads |

Leading dashes, path separators, `.` and `..` are refused: the identifier becomes a directory of its own and an argument to `bd` and `git`. `init` and `help` are commands, so neither reaches the table at all.

A pull request number is read against the current repository, whatever host the URL names.

## What the picker offers

Every worktree git reports, less the main checkout, then the open pull requests of `origin` and the ready beads without one. Each worktree is read off its branch, and one whose branch names neither a bead nor a pull request is offered under that branch and entered with `$SHELL`.

A row is titled by whichever tool names that kind: `bd` a bead, `gh pr list` a pull request, drafts included. gh is pinned to origin's URL: left to resolve a checkout with several remotes it favours `upstream`, and lists pull requests whose head the fetch from origin then cannot find.

One tool answering costs nothing of the other, and an adapter that will not answer costs its own rows and its own titles: the worktrees list either way, unlabeled.

## What each invocation does

A target's worktree is the one checked out on its [branch](configuration.md), wherever git reports it. Only a new worktree needs a directory chosen for it.

Creating a worktree is the moment work on that target begins, so it also claims the bead and runs the [command](configuration.md#commands) its kind of target opens on. Entering one that already exists hands over `$SHELL`, names the conversations it carries and prints the line that returns to them. Provisioning is idempotent, so every form below re-enters an open worktree.

| Invocation | On a target with no worktree |
|---|---|
| `work <id>` | vet, create, claim, and launch |
| `work <pr>` | create, and launch |
| `work <id> --shell` | create only; the bead is left as it is |
| `work <id> --editor` | vet, create, claim, and open the editor |
| `work` | as for the target picked |

Vetting is [bead-workflow policy](../explanation/worktree-per-ticket.md): a deferred, closed, epic, criteria-less or dependency-blocked bead is refused, and the message says which. It guards the claim, so the paths that do not claim do not vet.

## Flags

`--editor` runs `<editor> <dir>` on the worktree, where the editor is `$VISUAL`, else `$EDITOR`, and names none of the conversations a shell entry names. With neither set the invocation is refused before anything is created or claimed. It is exclusive with `--shell`.

`--model` and `--effort` are accepted on every invocation and are passed to the [command](configuration.md#commands) that is launched. One placing neither drops both without a word, as `--shell` and `--editor` do.

`--version` prints and exits, touching no repository. It reports the `git describe --tags --always --dirty` of the checkout the binary was [built from](mise-tasks.md#version-stamping), or `dev` when nothing stamped it.

## Shell integration

`work init fish | source` in `config.fish` completes the identifier with what the picker offers.

## Handoff

`work` changes into the worktree and replaces itself with the command, so the calling shell keeps its own directory. A failure to enter is one line on stderr and exit 1; a dismissed picker exits 1 silently.

Which command each path runs is `internal/work/handoff.go`.
