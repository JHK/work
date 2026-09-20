# Package boundaries

What the core may reach, what an integration may not, and what the vocabulary may not. What each integration does is [the integrations](../references/integrations.md).

## R3 — The core reaches the vocabulary, git and the settings

`internal/work` imports `internal/worktree`, `internal/git`, `internal/config` and the standard library, and nothing else. Git is there because worktrees are git's, the settings because the core reads which action a moment opens on. Why the packages are cut this way is [the partition behind the seams](../explanation/seam-partition.md).

*Enforced by:* `TestCoreReachesNothingElse` in `internal/module/boundaries_test.go`, which reads the package's imports off `go list`.

## R4 — No integration reaches another integration

No package under `internal/integration/` imports a package of another integration, test files included. An integration is one directory there, with whatever it holds: the seams it fills, and the tool it goes through.

R4 reaches further than R3, which reads one package. It is keyed on the path, so `internal/integration` itself is outside it: what sits there is shared by two or more integrations and is no integration's own.

*Enforced by:* `TestNoIntegrationReachesAnother` in `internal/module/boundaries_test.go`, which reads every implementation's imports off `go list`, test compilations included.

## R8 — The vocabulary reaches nothing of the module

`internal/worktree` imports the standard library and nothing else. Every package speaks its words, so a word reaching one package would carry that package into all of them. Its own test files stand on [the ground](test-isolation.md) every package's tests stand on, and are outside the rule.

*Enforced by:* `TestTheVocabularyReachesNothing` in `internal/module/boundaries_test.go`, which reads the package's imports off `go list`.
