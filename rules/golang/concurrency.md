---
paths:
  - "**/*.go"
  - "**/go.mod"
  - "**/go.sum"
---
# Go Concurrency

## Goroutines

- Always know when a goroutine stops — use `sync.WaitGroup` or errgroup.
- Never leak goroutines: pair every `go` call with a cancellation path.

## Channels

- Owner goroutine writes; consumers read. Close from the writer side only.
- Buffered channels for bounded backpressure; unbuffered for synchronization.
- Use `for range` to read until the channel is closed.

## Synchronization

- `sync.Mutex` for critical sections; `sync.RWMutex` for read-heavy workloads.
- Prefer `sync.Map` only for hot-path lock-free patterns; otherwise use `map + sync.RWMutex`.
- Use `golang.org/x/sync/errgroup` for fan-out with error propagation.

## Context

- Pass `context.Context` as the first parameter of any blocking function.
- Never store context in a struct — pass it explicitly.
