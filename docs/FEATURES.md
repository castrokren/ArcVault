# Moved

The features inventory has been folded into the tested architecture docs:

- Dashboard views (user-visible features) → **[frontend.md](../dashboard/docs/frontend.md)** — the route list is a
  test-enforced contract.
- Backend routes and agent features → **[backend.md](backend.md)**.
- Windows service behaviour (install, `run-service`, error 1067, logs) → **[service.md](service.md)**.

These docs are checked against the code by `internal/docs` and `dashboard/src/docs`, so they cannot
silently drift the way this hand-maintained list did.
