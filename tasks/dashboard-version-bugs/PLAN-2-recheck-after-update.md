# Plan 2 — Refresh version state after a coordinator self-update completes

## Repo context (read this first)

- Project: ArcVault 2.0, `C:\Projects\ArcVault2.0`. The dashboard is Vue 3 + Vite in
  `dashboard/`, embedded into a Go coordinator that can self-update from GitHub releases
  and restart its own Windows service while the browser tab stays open.
- Run dashboard tests: `cd dashboard; npx vitest run` (vitest + @vue/test-utils + jsdom
  installed; existing tests co-located, e.g. `src/views/Login.test.js`).
- Depends on Plan 1 (`PLAN-1-version-source.md`) being applied first — it exports
  `getVersion` from `dashboard/src/api.ts` and removes hardcoded version defaults.
  If Plan 1 is not applied yet, apply it first.
- Do NOT touch Go code. Do NOT run any deploy scripts.

## The bug

The self-update flow: admin clicks update in `dashboard/src/components/UpdateModal.vue`
→ coordinator downloads the new binary, swaps it, restarts → the modal polls until the
server is back, then shows "Update Complete". The state machine lives in UpdateModal.vue
(~line 125):

```js
const state = ref('confirm') // confirm, progress, reconnecting, success, success_manual, error
```

State transitions happen in a `watch` on `props.lastEvent` (~line 144): WebSocket
`update_progress` events set `state` to `success` (step `done`), `success_manual`
(step `done_manual`), `error`, or `reconnecting` (step `restarting`, which starts
`startReconnectPolling()` — that poller flips state to `success` when the restarted
server responds).

**Nothing re-fetches version/update state after success.** `App.vue` owns the shared
`updateStore` (provided to children) and only populates it once in `onMounted`. So after
a successful update:

- The navbar still shows the pre-update version until a manual page refresh.
- `src/views/Agents.vue` (~line 177) compares each agent's version against
  `updateStore.current` to show the "outdated" indicator — with a stale `current`, agents
  running the OLD version appear up-to-date, so the operator doesn't know to update them.

This is the observed failure: coordinator updated 0.5.1 → 0.6.0 successfully, but the UI
kept saying 0.5.1 and showed no agent updates pending until a hard refresh.

## The fix

1. `dashboard/src/components/UpdateModal.vue`:
   - Add `'updated'` to the emits: `const emit = defineEmits(['close', 'updated'])`.
   - Emit it at every transition into a terminal success state:
     - where the event watch sets `state.value = 'success'` (step `done`),
     - where `state.value = 'success_manual'` is set (step `done_manual`),
     - inside `startReconnectPolling()` where reconnection success sets
       `state.value = 'success'`.
   - Simplest shape: a small helper `function succeed(newState) { state.value = newState; emit('updated') }`
     used at all three sites. Do not emit on `error`.
2. `dashboard/src/App.vue`:
   - The modal is mounted at ~line 101:
     `<UpdateModal :isOpen="updateModalOpen" :lastEvent="lastEvent" @close="updateModalOpen = false" />`
     Add `@updated="handleUpdated"`.
   - Add `handleUpdated()`: it should call the existing `checkForUpdates()` AND re-fetch
     `getVersion()` into `updateStore.current` (same calls Plan 1 wired into `onMounted`;
     extract a shared `refreshVersion()` helper rather than duplicating).
     Note for the `success_manual` case (terminal mode, binary swapped but service not
     restarted): the running server is still the old version, so the re-fetch truthfully
     shows the old version — that is correct, do not special-case it.
3. No other components need changes — `Agents.vue` reads the shared reactive
   `updateStore`, so it updates automatically once `current` is refreshed.

## Tests (write these, make them pass)

Create `dashboard/src/components/UpdateModal.test.js`, mocking `../api` and the auth
composable (mirror the mocking style of existing tests):

1. **Emits `updated` on done**: mount UpdateModal with `isOpen: true`, then update the
   `lastEvent` prop to `{ type: 'update_progress', payload: { step: 'done', pct: 100, message: '' } }`.
   Assert `wrapper.emitted('updated')` has one entry and rendered state shows
   "Update Complete". This FAILS on current code (no such emit exists).
2. **Emits `updated` on done_manual**: same, with `step: 'done_manual'`.
3. **Does NOT emit on error**: send `{ step: 'error', message: 'boom' }`; assert
   `emitted('updated')` is undefined.

App-side wiring (choose whichever is testable without heavy scaffolding; if App.vue can
be mounted with stubs per Plan 1's test, prefer a real assertion):

4. Mount App with the api module mocked; drive the modal to emit `updated`; assert
   `checkUpdate` and `getVersion` mocks were called again (i.e. more times than the
   initial onMounted calls).

Run: `cd dashboard; npx vitest run` — all tests (new and pre-existing, including Plan 1's)
must pass. Then `npm run build` and confirm it compiles.

## Acceptance criteria

- After the update flow reaches any success state, `updateStore.current`/`latest`/
  `available` are re-fetched from the server with no page reload.
- `error` path triggers no refresh.
- `npx vitest run` green, `npm run build` green.
- Do not commit; leave changes in the working tree for review.
