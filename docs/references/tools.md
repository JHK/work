# External tools

`work` bundles nothing. What it reaches for on `PATH`:

| Tool | Reached for | When it is missing | Run from |
|---|---|---|---|
| `git` | every invocation | nothing resolves | `internal/git/` |
| `bd` | naming, vetting and claiming a bead, creating its worktree, and the ready beads the picker and [completion](cli.md#shell-integration) list | every bead name fails; pull requests and open worktrees still work | `internal/beads/` |
| `gh` | origin's open pull requests and their titles, for the picker and completion | both open on the worktrees and ready beads alone, silently | `internal/forge/` |
| the [agent](configuration.md#commands), `claude` by default | what a new worktree is handed to | the worktree is created and the bead claimed, then the handoff fails | `internal/config/command.go` |
| the [editor](cli.md#flags), `$VISUAL` else `$EDITOR` | `--editor` | unset, the invocation is refused before anything is created or claimed | `internal/work/handoff.go` |
| `fzf` | `work` with no argument | the picker fails | `internal/cli/pick.go` |
| `mise` | `mise trust` in a new worktree | the worktree's first session prompts about its configs | `internal/mise/` |

Listing what a worktree carries reaches for nothing: `internal/sessions` reads Claude Code's transcripts under `~/.claude` directly.

Building the binary needs the Go toolchain, which [mise tasks](mise-tasks.md) provision.
