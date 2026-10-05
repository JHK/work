# Releases

Pushing a `v*` tag to GitHub publishes a release with no further step. [release.yml](../../.github/workflows/release.yml) runs goreleaser over [.goreleaser.yaml](../../.goreleaser.yaml), which builds, archives, checksums and creates the GitHub release with goreleaser's default changelog.

To cut one, tag `main` and push the tag to `origin`:

```bash
git tag v0.2.0 && git push origin v0.2.0
```

## What a release publishes

- One `work_<version>_<os>_<arch>.tar.gz` per target: linux/amd64, linux/arm64 and darwin/arm64. `<version>` drops the tag's leading `v`.
- Each archive holds the `work` binary, `LICENSE` and `README.md`.
- `checksums.txt`, the SHA-256 of each archive.

The binaries are static (`CGO_ENABLED=0`) and stripped.

## Install through mise

`mise use -g github:JHK/work` installs the newest release through mise's [github backend](https://mise.jdx.dev/dev-tools/backends/github.html), and `github:JHK/work@<version>` pins one. The backend picks the archive by the `linux`/`darwin` and `amd64`/`arm64` tokens in the archive names above, with no `asset_pattern` in the user's config. An archive name outside goreleaser's default template can force that override on every user.

mise holds back a release younger than its `minimum_release_age`, 24 hours by default: in that window the unpinned form fails and names the pinned one.

## Pins

The workflow follows the latest release within a major version: each action by its major tag, goreleaser by `~> v2`. To hold back a release that breaks, pin its exact version. Go comes from `[tools]` in [mise.toml](../../mise.toml), and `work --version` reports that toolchain.

Renovate keeps the pins current, configured in [renovate.json](../../renovate.json). Once a week it opens one PR per group: Go, the Go modules, the actions, and the other tools in `[tools]`. The Go PR moves `mise.toml` and the `go` line of `go.mod` together.
