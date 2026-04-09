# Installation

## Homebrew (recommended)

```bash
brew install matcra587/tap/pagerduty-client
```

Homebrew sets up shell completions automatically.

## GitHub Releases

Download a pre-built binary from the
[releases page](https://github.com/matcra587/pagerduty-client/releases)
and place it on your `PATH`.

## Go

Requires Go `1.26+`.

```bash
go install github.com/matcra587/pagerduty-client/cmd/pdc@latest
```

## Updating

pdc can update itself regardless of install method:

```bash
pdc update
```

It detects how it was installed and delegates accordingly:

| Method | Detection | Action |
|--------|-----------|--------|
| Homebrew | Binary path under Homebrew prefix | `brew upgrade matcra587/tap/pagerduty-client` |
| `go install` | Module path in embedded build info | `go install .../cmd/pdc@latest` |
| Binary | Any other path | Downloads the latest release asset and replaces the binary in place |

### Update Channels

pdc supports two update channels:

| Channel | Tracks | Use case |
|---------|--------|----------|
| `stable` (default) | Latest tagged GitHub release | Production use |
| `dev` | Latest commit on `main` | Testing unreleased changes |

Set the channel per invocation, via environment variable or in config:

```bash
# Flag
pdc update --channel dev

# Environment variable
PDC_UPDATE_CHANNEL=dev pdc update

# Config (persisted)
pdc config set defaults.update_channel dev
```

Channel support varies by install method:

| Method | `stable` | `dev` |
|--------|----------|-------|
| Homebrew (stable install) | `brew upgrade` | Switch required (see below) |
| Homebrew (HEAD install) | Switch required (see below) | `brew upgrade --fetch-HEAD` |
| `go install` | `@latest` | `@main` |
| Binary | Self-replace from release | Not supported |

Switching Homebrew between stable and HEAD requires a reinstall:

```bash
# Stable to dev
brew uninstall pagerduty-client
brew install --HEAD matcra587/tap/pagerduty-client

# Dev to stable
brew uninstall pagerduty-client
brew install matcra587/tap/pagerduty-client
```

When running on the dev channel, pdc checks whether your build is
behind the latest commit on `main` and notifies you on each run
(cached for 24 hours).

## Shell Completion

Homebrew sets up completions automatically. If you installed via
GitHub Releases or `go install`, run:

```bash
pdc --install-completion
```

Completions include dynamic lookups that query the PagerDuty API for
resource IDs (incidents, services, teams, etc.). These require a valid
API token (via `PDC_TOKEN` or the OS keyring) and enforce a 5-second
timeout to keep tab completion responsive.

For best results, set a default team and/or service in your config.
Without filters, dynamic lookups fetch all resources across your
account, which can be slow on large organisations.
