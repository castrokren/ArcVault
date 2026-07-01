---
paths:
  - "**/*.go"
  - "**/go.mod"
  - "**/go.sum"
---
# Go Error Handling

> This file extends [common/error-handling.md](../common/error-handling.md) with Go specific content.

## Principles

- Handle every error — no bare `_` assignments for error returns.
- Prefer `errors.Is` / `errors.As` over type assertions for sentinel errors.

## Wrapping

```go
if err != nil {
    return fmt.Errorf("read config: %w", err)
}
```

- Use `%w` to preserve the error chain for `errors.Is`/`errors.As`.
- Use `%v` only when you intentionally want to break the chain.

## Panics

- Panic only for truly unrecoverable states (e.g., failed invariant in `init()`).
- Recover in top-level goroutines only; log the stack trace on recovery.

## Logging

- Log errors at the boundary (handler, CLI entrypoint), not in the middle.
- Include enough context to trace the failure: `log.Printf("user %d: %v", id, err)`.
