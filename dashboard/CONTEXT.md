# CONTEXT — dashboard

Last updated: 2026-07-24

## What happens here
The Vue 3 + Vite web UI. Built with `npm run build`, then embedded into the coordinator
binary at `coordinator/static/dist` — the coordinator serves it directly, no separate web
server. Design system: "Obsidian Pro" (token-based).

## Process — how work flows
- `src/api.ts` — API client; all coordinator calls go through here.
- `src/schemas/` — Zod schemas validating API responses against contract at runtime.
- `src/router/` — Vue Router; admin-only views carry `meta: { requiresRole: 'admin' }`,
  enforced by a `beforeEach` navigation guard.
- `src/views/` — page-level components (Agents, Users, Federation, Alerts, Credentials, ...).
- `src/components/` — reusable UI components.
- `src/composables/` — shared reactive logic (stores, polling, WebSocket state).
- `src/docs/` — vitest contract test for `docs/frontend.md` (route + role-guard inventory).
- `docs/frontend.md` — the doc itself, checked against `src/router/index.js` by the test above.

## What files live here
- `src/` — application source (see above)
- `public/` — static assets
- `design.md` — design system notes
- `vite.config.js` — build config
- `package.json` — dependencies, scripts

## Standards / rules specific to this workspace
- **Admin UI routes are guarded router-side** — router-level role checks (`beforeEach`)
  using `meta: { requiresRole: 'admin' }`. Applied to federation, users, groups, alerts,
  credentials routes. (Coordinator enforces the same routes server-side — this is
  defense in depth, not the only gate.)
- **Contract-tested doc:** `dashboard/docs/frontend.md` locks every router route + its role guard,
  checked by `dashboard/src/docs` (vitest). Add/change/remove a view or route → update the
  CONTRACT block in the same commit, or the pre-commit hook blocks it.
- **Version staleness compares against the latest *release*, not the running coordinator.**
  Baseline is `updateStore.latest || updateStore.current` (a push-update installs the latest
  release; the fallback covers non-admins who can't reach admin-only `/api/update/check`).
  Use `versionBehind(a, b)` from `src/utils/format.js` — a semver strictly-older check that
  fails safe on missing values. Never `!==` on version strings. Drift badges render for
  federated agents too, labelled "stale" since Update is local-only.
- **Never surface the coordinator admin token in the UI.** It is fleet-wide and never expires,
  so any XSS could `fetch()` it (POST instead of GET does not help — CSRF was never the
  threat). Enrolling a machine goes through `downloadBootstrapScript(hostname)` →
  `GET /api/admin/bootstrap.ps1`, which mints a 1-hour scoped token. An endpoint returning
  the admin token was deliberately removed on 2026-07-08; do not reintroduce it.
- Vue components: PascalCase.
- API contract layer is Zod (`src/schemas/`) — don't bypass it with raw `fetch`/`axios` calls.

## Known cleanup, awaiting a decision
Run `fallow dead-code --format json --quiet` from this directory (config:
`dashboard/.fallowrc.json`; note `duplicates.minOccurrences: 3` hides pair-only clones).

- Unused files: `src/components/Sparkline.vue` (likely a deliberate leftover — the Fleet
  Console shipped without per-agent sparklines) and `src/components/orbit/OrbitField.vue`
  (only referenced as a stub in `views/Login.test.js`, which `Login.vue` no longer uses).
- Duplication: the page-header block is copied 4× across `FederationHealth.vue`, `Groups.vue`,
  `Users.vue`, `admin/Credentials.vue` — candidate for a shared `<PageHeader>`. A CSS rule in
  `style.css:310-320` is also duplicated into `Groups.vue` and `Users.vue`.
- **Watch for `vi.mock` paths that don't resolve.** Vitest matches mocks by resolved module,
  so a wrong relative path is a *silent no-op* and the test runs against the real module.
  `Login.test.js` mocked `../../composables/useAuth.js` (one level too many) for an unknown
  period while appearing to be mocked.
