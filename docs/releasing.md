# Releasing

## Version scheme

Releases follow [semver](https://semver.org/): `MAJOR.MINOR.PATCH`.
Tags take the form `v0.6.12`.
Breaking changes increment MAJOR.
New features increment MINOR.
Bug fixes increment PATCH.

## How to release

Tag the commit you want to ship and push the tag:

```bash
git tag v0.6.12
git push origin v0.6.12
```

The `release` workflow triggers on any tag matching `v[0-9]*.[0-9]*.[0-9]*`.
It runs GoReleaser, which builds binaries, creates the GitHub release,
uploads assets and updates the Homebrew tap.
Nothing else to do after pushing the tag.

The workflow uses a `concurrency` group scoped to the ref.
Each tag runs independently, but re-pushing the same tag cancels the
in-flight run. Wait for the job to finish before retagging to avoid
race conditions with the Homebrew tap update.

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

The changelog excludes commits with types `docs`, `style`, `chore`, `ci` and `test`.

## Version embedding

GoReleaser injects version metadata at link time via `-ldflags`.
The variables live in `internal/version/version.go` and default to
`"dev"` or `"unknown"` in local builds.

| Variable | Value injected |
|----------|---------------|
| `version.Version` | Git tag (e.g. `0.6.12`) |
| `version.Commit` | Short commit hash |
| `version.Branch` | Branch name |
| `version.BuildTime` | Commit timestamp (RFC3339) |
| `version.BuildBy` | `goreleaser` |

`task build` injects the same fields using `git describe` and
`git rev-parse`, so local binaries also report meaningful version info.

Run `pdc version` to inspect the embedded values.

## Homebrew tap

GoReleaser updates the Homebrew cask in
[matcra587/homebrew-tap](https://github.com/matcra587/homebrew-tap)
after each release.
It uses the `HOMEBREW_TAP_TOKEN` secret, which must be a PAT with
write access to that repository.

The tap repository must be public for `brew install` to work without authentication.
The source repository (`pagerduty-client`) must also be public so GoReleaser can publish the release assets the cask points to.

Users install or upgrade with:

```bash
brew install matcra587/tap/pagerduty-client
# or
brew upgrade matcra587/tap/pagerduty-client
```

## Self-update

`pdc update` detects the install method and delegates accordingly:

| Method | Detection | Action |
|--------|-----------|--------|
| Homebrew | Binary path under `/opt/homebrew/`, `/usr/local/Cellar/` or `/home/linuxbrew/` | Runs `brew upgrade matcra587/tap/pagerduty-client` |
| `go install` | Module path in embedded build info matches `github.com/matcra587/pagerduty-client` | Runs `go install .../cmd/pdc@latest` |
| Binary | Any other path | Downloads the latest release asset and replaces the binary in place |

The command checks the latest tag via the GitHub API first.
If the installed version is already current, it exits early.
