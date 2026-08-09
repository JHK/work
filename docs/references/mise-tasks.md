# Mise tasks

Where each task puts things. `mise tasks` lists them and [mise.toml](../../mise.toml) defines them.

| Task | Result |
|---|---|
| `build` | `./work` in the repository root, which `.gitignore` covers |
| `install` | `work` in the Go bin directory, which is on `PATH` |
| `uninstall` | removes the binary `install` wrote |
| `test` | `go test ./...` |
| `lint` | `go vet ./...` then `golangci-lint run` |

The toolchain versions pinned in `[tools]` are what CI and a fresh checkout resolve to.

## Version stamping

`build` and `install` share the `[vars]` entry `ldflags`, which stamps `main.version` with `git describe --tags --always --dirty` for [`work --version`](cli.md#commands) to report.

The value is read where the task runs: building inside a worktree describes that worktree.

Any other route to a binary, `go build ./cmd/work` among them, leaves the compiled-in default `dev`, and so does a build where `git describe` fails.

## Where the binary lands

`install` and `uninstall` both resolve `$GOBIN`, falling back to `$GOPATH/bin`. Read the live value with `go env GOBIN`.

When mise supplies the Go toolchain it also sets `GOBIN` inside that toolchain's own directory, which carries the version in its path. Raising `go` in `[tools]` moves `GOBIN`, leaving the installed binary behind in the old directory and off `PATH`.
