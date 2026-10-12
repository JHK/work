# Releases

Pushing a `v*` tag to GitHub publishes a release with no further step. [release.yml](../../.github/workflows/release.yml) runs goreleaser over [.goreleaser.yaml](../../.goreleaser.yaml), which builds, archives, checksums and creates the GitHub release with goreleaser's default changelog.

## What a release publishes

- One `work_<version>_<os>_<arch>.tar.gz` per target: linux/amd64, linux/arm64 and darwin/arm64. mise's github backend matches these names with no `asset_pattern`. `<version>` drops the tag's leading `v`.
- Each archive holds the `work` binary, `LICENSE` and `README.md`.
- `checksums.txt`, the SHA-256 of each archive.

## Homebrew

The formula in [JHK/homebrew-tap](https://github.com/JHK/homebrew-tap) builds from one tag's source tarball, pinned by URL and SHA-256. A release reaches `brew` only once the formula names its tag.

## Pins

The release builds with the Go pinned in `[tools]` of [mise.toml](../../mise.toml).

[Renovate](../../renovate.json) opens one PR per dependency group each week. The Go PR moves `mise.toml` and the `go` line of `go.mod` together.
