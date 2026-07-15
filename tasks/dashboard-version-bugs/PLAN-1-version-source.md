# Plan 1 — Navbar version must come from the server, not a hardcoded string

## Repo context (read this first)

- Project: ArcVault 2.0, `C:\Projects\ArcVault2.0`. The dashboard is Vue 3 + Vite in
  `dashboard/`, built and embedded into a Go coordinator binary.
- Run dashboard tests: `cd dashboard; npx vitest run` (vitest + @vue/test-utils + jsdom
  are already installed; existing tests live next to their components, e.g.
  `src/views/Login.test.js`).
- Do NOT touch Go code for this plan. Do NOT run any deploy scripts.

## The bug

`dashboard/src/App.vue` line ~146 hardcodes the coordinator version shown in the navbar:

```js
const updateStore = reactive({
  current: 'v0.5.1',   // <-- hardcoded; navbar renders this at App.vue line 11
  latest: 'v0.5.1',
  available: false,
  releaseUrl: '',
  assetUrl: ''
})
```

`updateStore.current` is only ever overwritten by `checkForUpdates()` (App.vue ~line 175),
which calls `checkUpdate()` → `GET /api/update/check`. That endpoint is **admin-only**
(server wraps it in `adminRoute`). Consequences:

1. Non-admin users see the baked-in string forever (their check gets 403).
2. If the GitHub-backed update check fails for any reason, everyone sees the baked-in string.
3. After every release, the hardcoded literal is stale until someone edits it — this is
   exactly what happened: coordinator was running v0.6.0 while the navbar said v0.5.1.
4. `src/views/Agents.vue` line ~177 uses `updateStore.current` to decide whether an agent
   is outdated (`normalize(agent.version) !== normalize(updateStore.current)`), so a stale
   `current` also makes outdated agents look up-to-date.

There is a correct, always-available source: `GET /api/version` returns
`{"version":"v0.6.0", ...}` from the running binary (ldflags-injected). Server route is
`adminTokenViewerRoute` — any authenticated user can call it. The client function already
exists in `dashboard/src/api.ts` line ~183 but is **not exported**:

```ts
const getVersion = async (): Promise<Types.VersionResponse> => {
  const res = await request('GET', '/api/version')
  return validateResponse('/api/version', VersionResponseSchema, res)
}
```

## The fix

1. `dashboard/src/api.ts`: export `getVersion`.
2. `dashboard/src/App.vue`:
   - Change the reactive defaults to empty strings (`current: ''`, `latest: ''`).
   - In `onMounted`, when authenticated, call `getVersion()` and set
     `updateStore.current = data.version` (do this in addition to the existing
     `checkForUpdates()` call — keep both; `checkForUpdates` still supplies
     `latest`/`available` for admins, and when it succeeds it may overwrite `current`
     with the same value, which is fine).
   - Guard the navbar span so an empty version renders nothing rather than an empty
     badge: `<span class="nav-version" v-if="updateStore.current">{{ updateStore.current }}</span>`.
3. Kill the sibling hardcoded fallbacks (same disease):
   - `dashboard/src/components/UpdateModal.vue` ~line 117: inject default has
     `current: 'v0.2.0', latest: 'v0.2.0'` → change both to `''`.
   - Check `dashboard/src/components/AgentUpdateModal.vue` and
     `dashboard/src/views/Agents.vue` inject defaults for version literals; change any
     to `''`. (Agents.vue's `hasUpdate` already returns `false` when
     `updateStore.current` is empty — that behavior is correct: unknown ≠ outdated.)
4. Grep the whole `dashboard/src` tree for any remaining `v0\.\d` literal outside tests
   and remove/neutralize it.

## Tests (write these, make them pass)

Create `dashboard/src/App.test.js` (or, if mounting App.vue proves too entangled with
router/websocket setup, test the extracted behavior — but prefer mounting with stubs,
mirroring the mocking style used in `src/views/Login.test.js`):

1. **Version comes from /api/version**: mock the api module so `getVersion` resolves
   `{ version: 'v9.9.9' }` and `checkUpdate` rejects (simulating a non-admin 403).
   Mount App (stub `router-view`, mock the auth composable as authenticated). Assert the
   navbar text contains `v9.9.9`. This test FAILS on current code (would show `v0.5.1`).
2. **No baked version when both calls fail**: `getVersion` and `checkUpdate` both reject.
   Assert the rendered navbar version text is empty — specifically that it does NOT
   contain `v0.5` or any `v0.` literal.

Run: `cd dashboard; npx vitest run` — all tests (new and pre-existing) must pass.
Then run `npm run build` in `dashboard/` and confirm it compiles.

## Acceptance criteria

- No hardcoded semver literals remain in `dashboard/src` (outside test files).
- Navbar version renders the value served by `/api/version` for ANY authenticated role.
- `npx vitest run` green, `npm run build` green.
- Do not commit; leave changes in the working tree for review.
