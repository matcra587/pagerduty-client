---
description: >
  Go language standards: code style, naming, error handling, spec rules,
  concurrency patterns and Go 1.25/1.26 features. Loaded for all Go files.
paths:
  - "**/*.go"
---

# Go Standards

Rules for writing Go in this project. The
[Go 1.26 Language Specification](https://go.dev/ref/spec) is the
authority when behaviour is unclear.

## Project Style

- Keep code clear, concise and easy to follow.
- Favour simplicity over cleverness. Don't build what you don't need (YAGNI).
- Don't create helpers, abstractions or packages until a second use proves the need.
- Go 1.26 - use modern idioms (range-over-func, `errors.AsType`, etc.)
- Format with `gofumpt` (stricter than gofmt)
- Lint with golangci-lint and `.golangci.yml` (strict revive rules)
- Never use naked returns
- Log with `gechr/clog`; never import `log` or `log/slog` (enforced by depguard)
- Place all packages under `internal/` - this is a binary, not a library
- Keep `cmd/` to Cobra command wiring only; put logic in `internal/`
- Return errors; never log and swallow them
- Pass `context.Context` as the first argument to every API-calling function

## Naming

- `MixedCaps` for exported, `mixedCaps` for unexported. Never `snake_case` or `ALL_CAPS`.
- Initialisms are all-caps or all-lower: `URL`/`url`, `ID`/`id`. Never `Url`, `Id`.
- No type in variable names: `users` not `userSlice`, `count` not `numUsers`.
- Spell words fully: `Sandbox` not `Sbx`.
- Name length proportional to scope. Single-letter fine in small scopes.
- No `Get` prefix on getters: `Counts()` not `GetCounts()`.
- Don't repeat receiver type in method names: `(c *Config) WriteTo()` not `WriteConfigTo()`.
- Receiver names: 1-2 letters, abbreviation of type, consistent.
  Never `this`/`self`.
- Constants use `MixedCaps`, named by role not value.
- Package names: lowercase, no underscores. Avoid `util`, `helper`, `common`.
- Don't repeat package name in exports: `widget.New()` not `widget.NewWidget()`.

## Imports

- Group: (1) stdlib, (2) external packages. Blank line between groups.
- Don't rename unless collision. No dot imports.
- Side-effect imports only in `main` or tests.

## Error Handling

- `error` always last return value.
- Return `error` interface, not concrete types, from exported functions.
- Error strings: no capital, no trailing punctuation.
- Handle errors before proceeding: `if err != nil { return }`
  then normal flow. No `else` after error return.
- Don't `panic` for normal errors. Return errors.
- Wrap with `%w` at end: `fmt.Errorf("context: %w", err)`. Use `%v` at system boundaries.
- Don't add redundant context when wrapping.
- Use `errors.Is`/`errors.AsType` (Go 1.26), not string matching or `==`.
- No in-band error values (-1, empty string). Use `(value, error)` or `(value, ok)`.

## Context

- `context.Context` always first parameter. Never store in struct.
- `context.Background()` only in main/init/test entrypoints.

## Interfaces

- Define in consuming package, not implementing package.
- Don't define before needed (YAGNI).
- Don't use interface param if only one concrete type will ever be passed.
- The zero value of an interface is `nil`. Guard method calls on
  interfaces that might be `nil`.
- Method sets follow value-vs-pointer receiver rules. Call only methods
  in the method set of the static type.

## Structs and Composite Literals

- Always use field names in struct literals for types from other packages.
- Omit zero-value fields when it improves clarity.
- Don't copy structs containing `sync.Mutex` or `bytes.Buffer`.
- Use composite literals (`T{...}`) to build structs, arrays, slices,
  and maps. Each evaluation of `&T{}` allocates a distinct value.
- Use `make` for slices, maps, and channels that need specific length
  or capacity. Use literals for fixed, known contents.

## Receivers

- Pointer receiver if: mutates, contains uncopyable fields, or large struct.
- Value receiver for: maps, channels, functions, small immutable structs.
- When in doubt, pointer receiver. Keep all methods on a type consistent.

## Variables and Zero Values

- `:=` for non-zero init. `var x T` for zero-value.
- `var t []string` (nil slice) over `t := []string{}` (empty).
- `len(s) == 0` to check emptiness, not `s == nil`.
- Initialise maps before writing. Reading nil map is safe.
- Use `any` not `interface{}`.
- Rely on zero values: `0`, `false`, `""`, `nil` for pointers, functions,
  interfaces, slices, maps, and channels. Never write
  `var s []T = nil` or `var n int = 0`.

## Types, Conversions, and Constants

- Types with the same underlying representation remain distinct; convert
  explicitly (e.g. `int32` to `int`).
- Untyped constants acquire a default type in typed contexts. Never rely
  on implementation-specific widening.
- `int`, `uint`, and `uintptr` are at least 32 bits; assume nothing more.

## Maps, Slices, and Strings

- Map iteration order is unspecified and varies between runs. Never
  depend on it. Sort keys explicitly when deterministic output matters.
- `m[k]` yields the zero value when `k` is absent. Use the comma-ok
  form (`v, ok := m[k]`) to distinguish presence from zero. This
  applies equally to type assertions on nested JSON: always guard with
  `if v, ok := raw["key"].(map[string]any); ok`.
- Substrings and subslices share the underlying array. Mutations through
  one affect the other. Copy defensively when slices cross API boundaries:
  `append([]string(nil), vs...)`.
- Treat strings as immutable byte sequences.

## Concurrency and Channels

- Never start goroutine without knowing how it stops.
- Prefer synchronous functions. Let caller add concurrency.
- Specify channel direction where possible.
- Unbuffered sends and receives block until the other side is ready.
  Buffered channels block only when full or empty.
- Senders close channels; receivers observe `ok == false` from
  `v, ok := <-ch`.
- When multiple `select` cases are ready, Go picks one pseudo-randomly.
  Never assume ordering.
- Sends and receives on nil channels block forever.
- Use `sync.WaitGroup.Go()` (Go 1.25+) for goroutine fan-out. For
  bounded concurrency, combine with a channel semaphore:
  ```go
  sem := make(chan struct{}, maxConcurrent)
  for _, item := range items {
      sem <- struct{}{}
      wg.Go(func() {
          defer func() { <-sem }()
          process(item)
      })
  }
  ```

## Panics, Errors, and Recover

- Reserve `panic` for unrecoverable conditions or programming errors.
  Wrap all other failures in `error` values.
- Use `recover` only from deferred functions, only to turn panics into
  logged errors or clean shutdown. Never use it for control flow.
- Defers run in LIFO order. Their arguments evaluate at the `defer`
  statement, not when the deferred function runs.

## Comparisons

- Compare only types the spec marks as comparable. Slices, maps, and
  functions compare only to `nil`.
- Map keys must be comparable.
- Avoid relying on floating-point comparison semantics (NaN, infinity).

## Evaluation Order

- Operands and function arguments evaluate left to right.
- All RHS and LHS expressions in an assignment evaluate before any
  assignment takes place.
- Avoid expressions whose correctness depends on side effects combined
  with complex indexing.

## Documentation

- All exported names must have doc comments starting with the name.
- Comments that are sentences: capitalised, punctuated.
- Don't document obvious things. Don't restate what's in the type signature.

## Strings

- `+` for simple concat. `fmt.Sprintf` for formatted. `strings.Builder` for loops.
- Write to `io.Writer` with `fmt.Fprintf`; avoid building temporary strings.

## Miscellaneous

- `crypto/rand` for keys, never `math/rand`.
- No redundant `break` in `switch` (Go breaks automatically).
- Don't pass pointers to strings/interfaces to "save memory".
- Watch for variable shadowing with `:=` in inner scopes.
- Never use `fmt.Sprintf("%s:%d", host, port)` for host:port strings;
  use `net.JoinHostPort` (IPv6 safe).

## Go 1.25 Features

- **`sync.WaitGroup.Go()`**: Replaces `wg.Add(1); go func() { defer wg.Done() }`.
- **`testing/synctest`**: Virtualised time for testing concurrent code.
- **Container-aware GOMAXPROCS**: Reads cgroup CPU limits automatically.
- **`go vet` analysers**: `waitgroup` (misplaced Add) and `hostport`
  (Sprintf host:port) run automatically.
- **Runtime flight recorder**: `trace.FlightRecorder` for post-mortem traces.

## Go 1.26 Features

- **Green Tea GC** on by default. Cuts GC overhead 10-40%.
  Opt out with `GOEXPERIMENT=nogreenteagc`; disappears in Go 1.27.
- **`new()` accepts expressions**: `new(yearsSince(born))` replaces
  the two-step `v := expr; p := &v` pattern.
- **Self-referential generic types**: `type Adder[A Adder[A]] interface`.
- **`go fix` modernisers**: Run `go fix ./...` for modern idioms.
- **`errors.AsType[T](err)`**: Generic, type-safe alternative to
  `errors.As`. Prefer it:
  ```go
  // Old
  var apiErr *APIError
  if errors.As(err, &apiErr) { ... }

  // Go 1.26 - prefer
  if apiErr, ok := errors.AsType[*APIError](err); ok { ... }
  ```
- **`io.ReadAll`** ~2x faster with ~50% fewer allocations.
- **`bytes.Buffer.Peek(n)`**: Read n bytes without advancing.
- **`reflect` iterators**: `Type.Fields()`, `Type.Methods()` instead
  of index loops.
- **Goroutine leak detection** (experimental):
  `GOEXPERIMENT=goroutineleakprofile`.
- **Heap base randomisation** on 64-bit platforms.
- **cgo calls ~30% faster**.
