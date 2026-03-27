---
description: >
  Security invariants: HTTP client hardening, credential handling,
  context lifecycle and retry bounds. Loaded for all Go files.
paths:
  - "**/*.go"
---

# Security

## HTTP Client

- Disable redirect following on API clients. API servers do not
  redirect; following redirects leaks the Authorization header to
  the redirect target. Set `CheckRedirect` to return
  `http.ErrUseLastResponse`.
- Validate `baseURL` scheme before use. Require `https://` unless
  the host is `localhost` or `127.0.0.1`. A plain-HTTP base URL
  sends the token in cleartext.
- Cap `Retry-After` at 60 seconds. An unbounded value lets a
  malicious server stall the client indefinitely (context timeout
  is the only backstop otherwise).

## Credentials

- Never log token values. Log the token *source* ("flag", "env",
  "file", "keyring"), never the token itself.
- Never write tokens to config files. `Config.Token` uses
  `koanf:"-"` to prevent serialisation.
- Prefer `--token-file` over `--token` for programmatic use.
  `--token` exposes the value in process listings (`ps`,
  `/proc/cmdline`). The two flags are mutually exclusive.
  Resolution order: `--token` | `--token-file` > `PDC_TOKEN` >
  keyring.

## Context Lifecycle

- Cancel derived contexts on shutdown. Any `context.WithCancel`
  stored in a struct must have its cancel function called when the
  owning component stops (e.g. TUI quit, command completion).
