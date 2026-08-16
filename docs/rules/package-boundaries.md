# Package boundaries

What the core may reach. What each system is and where it lives is [the systems](../references/systems.md).

## R3 — The core reaches the vocabulary, git and the settings

`internal/work` imports `internal/worktree`, `internal/git`, `internal/config` and the standard library, and nothing else. Git is there because worktrees are git's, the settings because the core reads which action a moment opens on. Why the packages are cut this way is [the partition behind the seams](../explanation/seam-partition.md).

*Enforced by:* `TestCoreReachesNothingElse` in `internal/work/imports_test.go`, which reads the package's imports with `UseAllFiles`, so a build tag is no way out of the rule.
