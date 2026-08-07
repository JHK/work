# External tools

`work` bundles nothing. What it reaches for on `PATH`, each by one path through the program:

| Tool | Reached for | When it is missing |
|---|---|---|
| `git` | every invocation | nothing resolves |
| `bd` | naming, vetting and claiming a bead, creating its worktree, and the ready beads the picker and [completion](cli.md#shell-integration) list | every bead name fails; pull requests and open worktrees still work |
| `gh` | origin's open pull requests and their titles, for the picker and completion | both open on the worktrees and ready beads alone, silently |
| the [agent](configuration.md#commands), `claude` by default | what a new worktree is handed to | the worktree is created and the bead claimed, then the handoff fails |
| `fzf` | `work` with no argument | the picker fails |
| `mise` | `mise trust` in a new worktree | the worktree's first session prompts about its configs |

Listing what a worktree carries reaches for nothing: `internal/sessions` reads Claude Code's transcripts under `~/.claude` directly.

Building the binary needs the Go toolchain, which [mise tasks](mise-tasks.md) provision.

Each call site: `internal/beads/`, `internal/forge/`, `internal/git/`, `internal/cli/pick.go`, `internal/mise/`, and the default commands in `internal/config/command.go`.
