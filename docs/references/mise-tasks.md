# Mise tasks

`mise tasks` lists them and [mise.toml](../../mise.toml) defines them. What follows is where each one puts things.

| Task | Result |
|---|---|
| `build` | `./work` in the repository root, which `.gitignore` covers |
| `install` | `work` in the Go bin directory, which is on `PATH` |
| `uninstall` | removes the binary `install` wrote |
| `test` | `go test ./...` |
| `lint` | `go vet ./...` then `golangci-lint run` |

The toolchain versions pinned in `[tools]` are what CI and a fresh checkout resolve to.

## Where the binary lands

`install` and `uninstall` both resolve `$GOBIN`, falling back to `$GOPATH/bin`. Read the live value with `go env GOBIN`.

When mise supplies the Go toolchain it also sets `GOBIN` inside that toolchain's own directory, which carries the version in its path. Raising `go` in `[tools]` therefore moves `GOBIN`, leaving the installed binary behind in the old directory and off `PATH`. Reinstall after a toolchain bump, or set `GOBIN` somewhere version-independent before installing.
