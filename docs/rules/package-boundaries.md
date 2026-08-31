# Package boundaries

What the core may reach, and what a system may not. What each system does is [the systems](../references/systems.md).

## R3 — The core reaches the vocabulary, git and the settings

`internal/work` imports `internal/worktree`, `internal/git`, `internal/config` and the standard library, and nothing else. Git is there because worktrees are git's, the settings because the core reads which action a moment opens on. Why the packages are cut this way is [the partition behind the seams](../explanation/seam-partition.md).

*Enforced by:* `TestCoreReachesNothingElse` in `internal/work/imports_test.go`, which reads the package's imports off `go list`.

## R4 — No system reaches another system

No package under `internal/resolve/` or `internal/action/` imports a package of another system, test files included. A system is one directory under either of those two, with whatever it holds. The two halves of one system meet at a client of their own, `internal/beads/` for the tracker.

R4 reaches further than R3, which reads one package. It is keyed on the path, so an implementation that reaches another system's client passes it.

*Enforced by:* `TestNoSystemReachesAnother` in `internal/work/imports_test.go`, which reads every implementation's imports off `go list`, test compilations included.
