# Contributing

## Prerequisites

*   [mise](https://mise.jdx.dev/) - manages Go, task, actionlint, rumdl and zizmor
*   Go 1.26+ (mise installs this for you)

## Setup

```bash
mise install          # Install platform tools and Go
task deps             # Download Go dependencies
task build            # Build binary to ./dist/pdc-<os>-<arch>
```

That's it. The binary is ready to use.

## Development Workflow

1.  Write a failing test.
1.  Implement until the test passes.
1.  Run `task lint && task test` before pushing.

```bash
task test             # Run unit tests
task test:integration # Run integration tests (Stoplight mock, needs network)
task lint             # Lint with golangci-lint
task fmt              # Format with gofumpt
task vet              # Run go vet
task security         # Run govulncheck
```

## Code Style

*   Format with `gofumpt` (`task fmt`).
*   Lint with `golangci-lint` (`task lint`). Fix all warnings before pushing.
*   Spelling follows the existing codebase: colour, behaviour, organisation.
*   Use `gechr/clog` for structured logging. `log` and `log/slog` are banned by `depguard`.
*   Keep business logic out of `cmd/`. Wire commands there; implement in `internal/`.

## Commit Conventions

Follow [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/).

Format: `<type>[(<scope>)][!]: <subject>`

| Type | Use |
|------|-----|
| `feat` | New feature (MINOR) |
| `fix` | Bug fix (PATCH) |
| `perf` | Performance improvement |
| `docs` | Documentation only |
| `test` | Add or update tests |
| `refactor` | Neither fix nor feature |
| `build` | Build system, dependencies |
| `ci` | CI/CD workflows |
| `chore` | Tooling, housekeeping |

Subject: imperative mood, lowercase, no full stop.
Body: explain why, not what. Wrap at 72 characters.

Breaking changes: append `!` after the type (`feat!: remove v1 endpoint`)
or add a `BREAKING CHANGE:` footer.

## Pull Request Process

1.  Fork the repository.
1.  Create a branch from `main`.
1.  Push your branch and open a pull request against `main`.
1.  Describe what changed and why in the PR description.

Keep commits atomic - one logical change per commit.
