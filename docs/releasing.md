# Releasing

## Version scheme

Releases follow [semver](https://semver.org/): `MAJOR.MINOR.PATCH`.
Tags take the form `v0.8.1`.
Breaking changes increment MAJOR.
New features increment MINOR.
Bug fixes increment PATCH.

## How to release

Run the release task with the version you want to ship:

```bash
mise run release -- v0.8.1
```

The task validates the version, checks that `main` is clean and up to date,
runs the local CI and GoReleaser checks, then creates and pushes an annotated
tag.
Use `mise run release -- --dry-run v0.8.1` to run the checks without creating
or pushing the tag.

The `release` workflow triggers on any tag matching `v[0-9]*.[0-9]*.[0-9]*`.
It runs GoReleaser, which builds binaries, creates the GitHub release,
uploads assets, signs the checksum file and creates a GitHub artifact attestation.
It also publishes the Homebrew formula to `matcra587/homebrew-tap`.
Nothing else to do after pushing the tag.

The workflow uses a `concurrency` group scoped to the ref.
Each tag runs independently, but re-pushing the same tag cancels the
in-flight run. Wait for the job to finish before retagging to avoid
race conditions with the Homebrew tap update.

The release workflow uses a GitHub App installation token to update the tap.
It needs `APP_CLIENT_ID` as a `deploy` environment variable and
`APP_PRIVATE_KEY` as a `deploy` environment secret. The app installation must
have write access to `matcra587/homebrew-tap`.

## What GoReleaser produces

GoReleaser builds three binaries from `./cmd/pdc`:

| OS | Arch |
|----|------|
| Linux | amd64 |
| Linux | arm64 |
| macOS | arm64 |

macOS amd64 is excluded (see `ignore` in `.goreleaser.yml`).

Each binary is archived as `pagerduty-client_<version>_<os>_<arch>.tar.gz`.
A `checksums.txt` covering all archives is published alongside them.
GoReleaser signs that checksum file with cosign as `checksums.txt.sigstore.json`.
The workflow also creates a GitHub-native provenance attestation from
`checksums.txt` for the released archives.

The changelog excludes commits with types `docs`, `style`, `chore`, `ci` and `test`.

## Verifying artifacts

Download the archive you want to verify, then ask GitHub for its provenance
attestation:

```bash
gh attestation verify pagerduty-client_0.8.1_linux_amd64.tar.gz \
  --repo matcra587/pagerduty-client \
  --signer-workflow matcra587/pagerduty-client/.github/workflows/release.yml
```

You can also verify the cosign signature over `checksums.txt`:

```bash
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  checksums.txt
```

## Version embedding

GoReleaser injects version metadata at link time via `-ldflags`.
The variables live in `internal/version/version.go` and default to
`"dev"` or `"unknown"` in local builds.

| Variable | Value injected |
|----------|---------------|
| `version.Version` | Git tag (e.g. `0.8.1`) |
| `version.Commit` | Short commit hash |
| `version.Branch` | Branch name |
| `version.BuildTime` | Commit timestamp (RFC3339) |
| `version.BuildBy` | `goreleaser` |

`mise run build` injects the same fields using `git describe` and
`git rev-parse`, so local binaries also report meaningful version info.

Run `pdc version` to inspect the embedded values.

## Homebrew tap

The Homebrew formula lives in
[matcra587/homebrew-tap](https://github.com/matcra587/homebrew-tap).
The release workflow updates it from GoReleaser's `checksums.txt` using
`matcra587/github-actions/packages/homebrew-publish-formula`.

Users install or upgrade with:

```bash
brew install matcra587/tap/pagerduty-client
# or
brew update && brew upgrade matcra587/tap/pagerduty-client
```

## Self-update

`pdc update` detects the install method and delegates accordingly:

| Method | Detection | Action |
|--------|-----------|--------|
| Homebrew | Binary path under `/opt/homebrew/`, `/usr/local/Cellar/` or `/home/linuxbrew/` | Refreshes `matcra587/tap`, then runs `brew upgrade matcra587/tap/pagerduty-client` |
| `go install` | Module path in embedded build info matches `github.com/matcra587/pagerduty-client` | Runs `go install .../cmd/pdc@latest` |
| Binary | Any other path | Downloads the latest release asset and replaces the binary in place |

The command checks the latest tag via the GitHub API first.
If the installed version is already current, it exits early.
