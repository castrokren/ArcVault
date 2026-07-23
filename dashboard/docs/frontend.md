# ArcVault Frontend (Vue 3 dashboard)

The dashboard is a Vue 3 + Vite SPA, built and then **embedded into the coordinator binary**
(`coordinator/static/dist`). This doc describes its structure and the **test-enforced** list of
routes. The `CONTRACT:routes` block is checked by `dashboard/src/docs/frontend.doc.test.js` against
`router.getRoutes()` in `dashboard/src/router/index.js`. Add a view/route and this doc must change
with it — the pre-commit hook blocks a drifting commit.

> Supersedes the dashboard half of the old `docs/FEATURES.md` (now a redirect).

## Build → embed flow

`dashboard/` is built with `npm run build` (Vite) → `dashboard/dist`, which
`scripts/rebuild-and-restart.ps1` syncs into `coordinator/static/dist` before compiling the
coordinator. The coordinator serves it at `/`. Never hand-edit `static/dist`; it is generated.

## Layers

- **`src/router/index.js`** — hash-history router + `beforeEach` guard. Unauthenticated users are
  sent to `/login`; routes with `meta.requiresRole` are gated on `useAuth().hasRole()`.
- **`src/views/`** — one component per route (PascalCase). `src/views/admin/` for admin-only views.
- **`src/schemas/`** — Zod contract layer; API responses are parsed/validated here.
- **`src/api.ts`** — typed API client the views call. All backend routes are in [backend.md](backend.md).
- **`src/composables/`** — shared reactive state (`useAuth`, etc.).
- Design system: "Obsidian Pro" / Kiln tokens (`src/design.md`).

## Routes (test-enforced)

Format: `<path> -> <role>` where role is `any` (any authenticated user) or the required role from
`meta.requiresRole`. `/` redirects to `/agents`.

<!-- CONTRACT:routes — auto-checked by dashboard/src/docs/frontend.doc.test.js; do not hand-drift -->
- `/ -> any`
- `/login -> any`
- `/agents -> any`
- `/jobs -> any`
- `/history -> any`
- `/templates -> any`
- `/federation -> admin`
- `/federation/health -> admin`
- `/users -> admin`
- `/groups -> admin`
- `/alerts -> admin`
- `/admin/credentials -> admin`
<!-- /CONTRACT:routes -->

## What each view does

- **Agents** (`/agents`, `Agents.vue`) — fleet console: agent table (hostname/OS/version/last-seen),
  search + status filter, status rail, per-agent **Update** button (`AgentUpdateModal`, 4-step WS
  progress) and **Get token** button (`AgentTokenModal`, for installing an agent on a new machine).
- **Jobs** (`/jobs`) — list/create/edit jobs, run and cancel, live progress.
- **History** (`/history`) — job-run history.
- **Templates** (`/templates`) — reusable job templates (admin/operator to mutate).
- **Federation** (`/federation`, `/federation/health`) — admin-only peer coordinator failover config.
- **Users** (`/users`) — admin-only user CRUD + role assignment (admin/operator/viewer).
- **Groups** (`/groups`) — admin-only agent groups.
- **Alerts** (`/alerts`) — admin-only alert rules.
- **Credentials** (`/admin/credentials`) — admin-only encrypted backup credentials.
- **Login** (`/login`) — unauthenticated entry; the nav header is hidden here.

Coordinator self-update UI (check/apply from a GitHub release, `UpdateModal`) is surfaced in the
app header, not a dedicated route.

## Adding a view

1. Create the component in `src/views/` and import it in `src/router/index.js`; add the route (with
   `meta.requiresRole` if admin-only).
2. Add the `path -> role` line to the `CONTRACT:routes` block above (or run the test — it prints it).
3. Add any new backend calls to `src/api.ts` and document the route in [backend.md](backend.md).
