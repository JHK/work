# External tools

`work` bundles nothing. Five binaries are reached for on `PATH`, each by one path through the program.

| Tool | Reached for | When it is missing |
|---|---|---|
| `git` | every invocation | nothing resolves |
| `bd` | naming, vetting and claiming a bead, creating its worktree, and the picker's list of ready beads | every bead name fails; pull requests and open worktrees still work |
| `claude` | the launcher a new worktree is handed to, and the resume lines printed on re-entry | the worktree is created and the bead claimed, then the handoff fails |
| `fzf` | `work` with no argument | the picker fails |
| `mise` | `mise trust` in a new worktree | the worktree's first session prompts about its configs |

Building the binary needs the Go toolchain, which [mise tasks](mise-tasks.md) provision.

Each call site: `internal/beads/`, `internal/git/`, `internal/cli/pick.go`, `internal/mise/`, and the launcher argv in `internal/work/handoff.go`.
