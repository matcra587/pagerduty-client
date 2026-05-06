# Contributing

## Prerequisites

*   [mise](https://mise.jdx.dev/) - manages Go, platform tools, and project tasks
*   Go 1.26.2 (mise installs this for you)

## Setup

```bash
mise install          # Install Go and platform tools from .mise.toml
mise run deps         # Download Go dependencies
mise run build        # Build binary to ./dist/pdc-<os>-<arch>
```

That's it. The binary is ready to use.

## Git Hooks

The repo uses [hk](https://hk.jdx.dev/) for Git hooks. mise installs `hk`
and the hook tools, including `actionlint`, `rumdl`, `shellcheck`, and
`zizmor`.

Install hooks once per clone:

```bash
mise exec -- hk install --mise
```

On Git 2.54+, you can install hk once for every repo instead:

```bash
mise exec -- hk install --global --mise
```

Do not run both local and global hk installs for the same repo; Git will
run both hook entries.

Useful hook commands:

```bash
mise run pre-commit    # Run the pre-commit hook against staged files
hk check --all --check # Check all tracked files without applying fixes
hk fix --all           # Apply hook fixes across all tracked files
```

## Development Workflow

1.  Write a failing test.
1.  Implement until the test passes.
1.  Run `mise run check` before pushing.

```bash
mise run test             # Run unit tests
mise run test:integration # Run integration tests (Stoplight mock, needs network)
mise run lint             # Lint with golangci-lint
mise run lint:fix         # Run golangci-lint with auto-fix
mise run fmt              # Format with gofumpt
mise run vet              # Run go vet
mise run security         # Run govulncheck
mise run deps:update      # Update direct Go dependencies to latest patch releases
```

## Code Style

*   Format with `gofumpt` (`mise run fmt`).
*   Lint with `golangci-lint` (`mise run lint`). Fix all warnings before pushing.
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
