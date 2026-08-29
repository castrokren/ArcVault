---
paths:
  - "**/*.go"
  - "**/go.mod"
  - "**/go.sum"
---
# Go Coding Standards

## Formatting

- Run **`gofmt`** (or `go fmt ./...`) on all code before committing.
- Use `gofmt -s` for code simplification.

## Naming

- `camelCase` for unexported, `PascalCase` for exported identifiers.
- Single-letter names only for short-lived loop variables.
- Acronyms are all-upper (`HTTP`, `URL`, `ID`) or all-lower (`http`, `url`, `id`).

## Project Layout

- Follow the [Go Project Layout](https://go.dev/doc/modules/layout) conventions.
- One package per directory. Avoid `internal` exports leaking.
- Error types end in `Error` suffix (`ValidationError`), not `Exception`.

## Imports

- Group: stdlib → external → internal, separated by a blank line.
- Use `goimports` to auto-manage import order.
