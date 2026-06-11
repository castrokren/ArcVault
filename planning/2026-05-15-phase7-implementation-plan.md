# Phase 7 — Implementation Plan
**Date:** 2026-05-15
**Spec:** 2026-05-15-phase7-dashboard-improvements-design.md
**Branch:** phase-7-dashboard-improvements

---

## Pre-flight

- [ ] Create branch `phase-7-dashboard-improvements` from `main`
- [ ] Confirm existing 65 tests pass: `go test ./...` from project root

---

## Task 1 — Theme toggle (App.vue)

**File:** `dashboard/src/App.vue`

Steps:
1. Add `theme` ref: `const theme = ref(localStorage.getItem('arcvault-theme') || 'dark')`
2. Add `onMounted` hook: apply saved theme on load by calling `applyTheme(theme.value)`
3. Add `applyTheme(val)` function:
   - Sets `document.documentElement.setAttribute('data-theme', val)`
   - Writes to `localStorage.setItem('arcvault-theme', val)`
4. Add `toggleTheme()` function: flips `theme.value` between `'dark'` and `'light'`, calls `applyTheme`
5. Add icon button to existing header template, right side:
   - Sun icon when `theme === 'dark'` (click to switch to light)
   - Moon icon when `theme === 'light'` (click to switch to dark)
6. Add CSS block for `[data-theme="light"]` overrides — at minimum: background, text, card, and border color variables

**Verify:**
- Toggle switches icon correctly
- Theme persists after page refresh
- Light mode is readable (not broken layout)

---

## Task 2 — Search + filter: Agents view

**File:** `dashboard/src/views/Agents.vue`

Steps:
1. Add refs:
   ```js
   const searchQuery = ref('')
   const statusFilter = ref('all')
   ```
2. Add `filteredAgents` computed:
   ```js
   const filteredAgents = computed(() => {
     return agents.value.filter(a => {
       const matchesSearch = !searchQuery.value ||
         a.id.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
         a.hostname.toLowerCase().includes(searchQuery.value.toLowerCase())
       const matchesStatus = statusFilter.value === 'all' || a.status === statusFilter.value
       return matchesSearch && matchesStatus
     })
   })
   ```
3. Replace `agents` with `filteredAgents` in the template list render (`v-for`)
4. Add search input above the list:
   ```html
   <input v-model="searchQuery" type="text" placeholder="Search agents..." />
   ```
5. Add filter chips row below search:
   - Chips: All, Online, Offline
   - Active chip styled distinctly (e.g. filled vs outlined)
   - Click sets `statusFilter`
6. Add empty state below list:
   ```html
   <div v-if="filteredAgents.length === 0">No agents match your search</div>
   ```

**Verify:**
- Search filters by id and hostname, case-insensitive
- Status chips filter correctly
- AND logic: both active simultaneously narrows results
- Empty state appears when no match
- WebSocket agent update does not reset filters

---

## Task 3 — Search + filter: Jobs view

**File:** `dashboard/src/views/Jobs.vue`

Steps:
1. Add refs:
   ```js
   const searchQuery = ref('')
   const statusFilter = ref('all')
   ```
2. Add `filteredJobs` computed:
   ```js
   const filteredJobs = computed(() => {
     return jobs.value.filter(j => {
       const matchesSearch = !searchQuery.value ||
         j.name.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
         j.agent_id.toLowerCase().includes(searchQuery.value.toLowerCase())
       const matchesStatus = statusFilter.value === 'all' || j.status === statusFilter.value
       return matchesSearch && matchesStatus
     })
   })
   ```
3. Replace `jobs` with `filteredJobs` in the template list render (`v-for`)
4. Add search input above the list:
   ```html
   <input v-model="searchQuery" type="text" placeholder="Search jobs..." />
   ```
5. Add filter chips row below search:
   - Chips: All, Running, Scheduled, Success, Failed
   - Active chip styled distinctly
   - Click sets `statusFilter`
6. Add empty state:
   ```html
   <div v-if="filteredJobs.length === 0">No jobs match your search</div>
   ```

**Verify:**
- Search filters by name and agent_id, case-insensitive
- All 5 status chips work correctly
- AND logic works
- Empty state appears when no match
- WebSocket job update does not reset filters

---

## Task 4 — Rebuild dashboard embed

Steps:
1. `cd dashboard && npm run build`
2. Confirm no build errors
3. `go build ./...` from project root — confirms embedded static files compile cleanly

**Verify:**
- Build succeeds with no errors
- `go test ./...` — all 65 tests still pass

---

## Task 5 — Manual smoke test

Run `coordinator start` and open the dashboard in a browser.

Checklist:
- [ ] Theme toggle visible in header
- [ ] Theme persists across refresh
- [ ] Light mode renders correctly
- [ ] Agents view: search input + chips present
- [ ] Jobs view: search input + chips present
- [ ] Search filters work live as you type
- [ ] Status chips filter correctly
- [ ] Empty state shows when no results
- [ ] No console errors

---

## Task 6 — Commit

```
git add dashboard/src/App.vue dashboard/src/views/Agents.vue dashboard/src/views/Jobs.vue dashboard/src/components/ 
git commit -m "feat: theme toggle, search, and filter for agents and jobs views"
```

---

## Done

All tasks complete → use finishing-a-development-branch skill to merge and tag.
