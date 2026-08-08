# External tools

`work` bundles nothing. What it reaches for on `PATH`:

| Tool | Reached for | When it is missing | Run from |
|---|---|---|---|
| `git` | every invocation | nothing resolves | `internal/git/` |
| `bd` | naming, vetting and claiming a bead, creating its worktree, and the ready beads the picker and [completion](cli.md#shell-integration) list | every bead name fails; pull requests and open worktrees still work | `internal/beads/` |
| `gh` | origin's open pull requests and their titles, for the picker and completion | both open on the worktrees and ready beads alone, silently | `internal/forge/` |
| the [agent](configuration.md#commands), `claude` by default | what a worktree is handed to, and the transcripts under `~/.claude` that say [what an existing one carries](cli.md#what-a-worktree-carries) | the handoff fails, a new worktree having already been created and its bead claimed | `internal/config/command.go`, `internal/sessions/` |
| the [editor](configuration.md#keys), `$VISUAL` else `$EDITOR` by default | the `editor` action, `--editor` included | unset, the invocation is refused before anything is created or claimed, and the screen leaves the row off | `internal/config/command.go` |
| `fzf` | `work` with no argument, and [the screen](cli.md#what-the-screen-offers) `ask` reaches | that question fails, the screen's after the vetting and before anything is created | `internal/cli/pick.go` |
| `mise` | `mise trust` in a new worktree | the worktree's first session prompts about its configs | `internal/mise/` |

Building the binary needs the Go toolchain, which [mise tasks](mise-tasks.md) provision.
