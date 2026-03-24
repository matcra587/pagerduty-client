---
description: >
  Contributing standards: conventional commits, branch naming
  and pull request workflow.
paths:
  - "**/*"
---

# Contributing

## Commits

Follow [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/).

Format: `<type>[(<scope>)][!]: <subject>`

### Mandatory Types

| Type | Semver | Use |
|------|--------|-----|
| feat | MINOR | Introduce a new feature |
| fix | PATCH | Patch a bug |

### Additional Types

The spec permits other types. This project uses the
`@commitlint/config-conventional` set:

| Type | Use |
|------|-----|
| build | Build system, dependencies |
| chore | Tooling, housekeeping |
| ci | CI/CD workflows |
| docs | Documentation only |
| perf | Performance improvement |
| refactor | Neither fix nor feature |
| style | Formatting, no logic change |
| test | Add or update tests |

These carry no implicit semver effect unless they include a breaking change.

### Breaking Changes

A breaking change correlates with MAJOR in semver.
Signal it in either of two ways (or both):

- Append `!` after the type/scope: `feat!: remove v1 endpoint`
- Add a `BREAKING CHANGE:` footer in the commit body

A breaking change can be part of any commit type.

### Rules

- Type and subject are required. Scope is optional.
- Subject: imperative mood, lowercase, no full stop.
- Body: wrap at 72 characters. Explain *why*, not *what*.
- Footers other than `BREAKING CHANGE:` follow
  [git trailer format](https://git-scm.com/docs/git-interpret-trailers).
