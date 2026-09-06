# Package boundaries

What the core may reach, and what an integration may not. What each integration does is [the integrations](../references/integrations.md).

## R3 — The core reaches the vocabulary, git and the settings

`internal/work` imports `internal/worktree`, `internal/git`, `internal/config` and the standard library, and nothing else. Git is there because worktrees are git's, the settings because the core reads which action a moment opens on. Why the packages are cut this way is [the partition behind the seams](../explanation/seam-partition.md).

*Enforced by:* `TestCoreReachesNothingElse` in `internal/work/imports_test.go`, which reads the package's imports off `go list`.

## R4 — No integration reaches another integration

No package under `internal/resolve/` or `internal/action/` imports a package of another integration, test files included. An integration is one directory under either of those two, with whatever it holds. The two halves of one integration meet at a client of their own, `internal/beads/` for the tracker.

R4 reaches further than R3, which reads one package. It is keyed on the path, so an implementation that reaches another integration's client passes it.

*Enforced by:* `TestNoIntegrationReachesAnother` in `internal/work/imports_test.go`, which reads every implementation's imports off `go list`, test compilations included.
