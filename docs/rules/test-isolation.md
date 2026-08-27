# Test isolation

What a test may see of the machine it runs on. What the ground under a test offers is [`internal/testenv/`](../../internal/testenv/).

## R6 — A test cannot reach the machine it runs on

A test sees a settings home and a home directory nobody has written to. It sees a git configured by nothing but the repository it was handed, and no way back to the shell that started the run. It answers the same on a maintainer's machine as on a bare one.

Importing `internal/testenv` is what isolates a process. A package whose test binary reaches git, the settings or the agent pulls that package in, whether the tests reach one of the three or the code under test does. A helper handed a directory that is not absolute stops the test: such a path resolves against the directory `go test` runs the package in, which is the package's own source.

*Enforced by:* `TestEveryTestPackageReachingGitOrTheSettingsIsIsolated` in `internal/testenv/isolation_test.go`, which reads each test binary's dependencies for a way to one of the three without a way to `internal/testenv`. It reads them by running `go list`, the one child a test starts under the machine's own home, because the go tool keeps its caches there. `absolute` in `internal/testenv/testenv.go`, which `testenv.Git` and `testenv.Write` call on the directory they are handed.
