# CONTEXT — dashboard

Last updated: 2026-07-22

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
- Vue components: PascalCase.
- API contract layer is Zod (`src/schemas/`) — don't bypass it with raw `fetch`/`axios` calls.
