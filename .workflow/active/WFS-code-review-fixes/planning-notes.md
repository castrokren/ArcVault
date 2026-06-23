# Planning Notes

**Session**: WFS-code-review-fixes
**Created**: 2026-06-12

## User Intent (Phase 1)

- **GOAL**: Implement all fixes from the code review report (15 findings) to improve code quality, security, and readability
- **KEY_CONSTRAINTS**: Fix 3 High-severity + 8 Medium-severity + 4 Low-severity issues across dashboard/src/api.ts, dashboard/src/views/Users.vue, and coordinator/static/dist/index.html; Code review completed 2026-06-12; findings span Correctness, Security, Performance, and Readability dimensions

---

## Context Findings (Phase 2)

### Critical Files Identified
- **dashboard/src/api.ts** (95 relevance): 9 issues across all dimensions (correctness, security, performance, readability)
- **dashboard/src/views/Users.vue** (90 relevance): 4 issues (correctness, security, readability)
- **coordinator/static/dist/index.html** (20 relevance): Build output, no direct edits needed

### Architecture Patterns Discovered

#### Correctness Patterns
1. **Pagination & Concurrency**: Users.vue uses `watch(page, () => fetchUsers())` without loading check. Jobs.vue and Agents.vue use safer `goToPage()` -> explicit `load()` pattern.
2. **Error Handling**: Delete operation in Users.vue closes modal before error visible. Jobs.vue keeps modal open for error display.
3. **Endpoint Consistency**: Mixed paths (`/downloads/installer` vs `/api/admin/bootstrap.ps1`) need documentation.

#### Security Patterns
1. **Token Storage**: Both `arcvault_jwt` and `arcvault_token` keys stored/retrieved in api.ts and useAuth.js. Used by 5 files (UpdateModal, AgentUpdateModal, Credentials, Users.vue, Jobs.vue).
2. **Filename Handling**: No sanitization of Content-Disposition header. Simple regex allows path traversal and special characters.
3. **CSRF Protection**: No CSRF token headers. Backend verification required to determine if needed.
4. **Error Display**: Uses browser `alert()` which is poor UX and exposes error details.

#### Performance Patterns
1. **Response Validation**: getAgents and getJobs skip validation (contrast: getGroups, login use validateResponse). Schemas exist (AgentListSchema, JobListSchema) but unused.
2. **Schema Coverage**: All paginated endpoints could benefit from validation consistency.

#### Readability Patterns
1. **Function Definition Inconsistency**: login() uses raw fetch; logout() uses request() helper.
2. **Magic Strings**: Token keys repeated 8+ times across api.ts and useAuth.js. Should extract to STORAGE_KEYS constant.
3. **Missing JSDoc**: 0% coverage on public API functions (getUsers, createUser, updateUserRole, deleteUser, getAgents, getJobs).
4. **Variable Naming**: buildQuery uses `q`, `k`, `v` instead of `queryString`, `key`, `value`.
5. **Component Complexity**: Users.vue has 157 lines of script logic managing 7 modals (create, edit role, delete confirm) + token copy + installer download. Candidate for composable extraction.

### Conflict Risk Assessment
**RISK LEVEL: LOW**

All fixes are localized with minimal cross-component dependencies:
- Token consolidation affects api.ts and useAuth.js only (same commit strategy)
- Race condition fix is isolated to Users.vue
- Filename sanitization is isolated to api.ts:downloadInstaller()
- Magic string extraction creates new constant in api.ts (optional useAuth.js import)
- No architectural conflicts; all changes are additive or local refactors

### Integration Touch Points
1. **Token Keys**: api.ts (definition) → useAuth.js (override) → 5 consumers (UpdateModal, AgentUpdateModal, Credentials, Users.vue, Jobs.vue)
2. **Download Functions**: downloadInstaller() and downloadBootstrapScript() should share error handling pattern
3. **Pagination Pattern**: Users.vue should adopt same pattern as Jobs.vue and Agents.vue for consistency

### Build System Notes
- **Build Tool**: Vite
- **Output**: coordinator/static/dist/index.html (auto-generated, don't edit directly)
- **After fixes**: Run `npm run build` in dashboard/ to regenerate dist/

---

## Conflict Decisions (Phase 3)
(To be filled if conflicts detected)

## Consolidated Constraints (Phase 4 Input)
1. Fix 3 High-severity + 8 Medium-severity + 4 Low-severity issues across api.ts, Users.vue, and build outputs
2. Code review completed 2026-06-12; findings span Correctness, Security, Performance, and Readability dimensions
3. **Critical Before Merge**:
   - CORR-001: Fix race condition in Users.vue pagination (line 362)
   - SEC-001: Sanitize filename from Content-Disposition header (api.ts:302-303)
   - SEC-002: Consolidate token storage to single key (api.ts + useAuth.js)
4. **Token Consolidation Strategy**: 
   - Create STORAGE_KEYS constant in api.ts with single TOKEN key
   - Update saveToken() to remove arcvault_jwt, keep only arcvault_token
   - Update getToken() to use single key (with migration fallback for arcvault_jwt on first read)
   - Update useAuth.js to import and use same STORAGE_KEYS
   - Update handle401() to clear only arcvault_token
5. **Filename Sanitization**: 
   - Match only filename without path separators: `/filename=([^/\\]+)$/`
   - Whitelist safe characters: `.replace(/[^a-zA-Z0-9._-]/g, '_')`
6. **Files to Modify**:
   - dashboard/src/api.ts (9 issues)
   - dashboard/src/views/Users.vue (4 issues)
   - dashboard/src/composables/useAuth.js (affected by token consolidation)
7. **No build changes needed**: coordinator/static/dist/index.html is auto-generated

---

## Task Generation (Phase 4)
(To be filled by action-planning-agent)

## N+1 Context
### Decisions
| Decision | Rationale | Revisit? |
|----------|-----------|----------|

### Deferred
- [ ] (For N+1)
