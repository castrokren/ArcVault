# Phase 22 — API Contract Layer
**Date:** 2026-06-03
**Goal:** Eliminate frontend/backend drift by introducing TypeScript types mirroring Go structs + Zod runtime validation on every API response.

---

## Problem

Frontend breaks silently when:
- API response shapes change (fields renamed, added, removed)
- Endpoints move or are renamed
- Auth/token behavior changes

Root cause: no contract between Go coordinator and Vue dashboard. `api.js` centralizes calls but has no type enforcement — shape changes are invisible until runtime.

---

## Approach

**Option A (chosen): TypeScript types + Zod runtime validation**
- Hand-written TypeScript interfaces in `dashboard/src/types/api.ts` mirroring Go response structs
- Zod schemas in `dashboard/src/schemas/` validating every response before it touches component state
- `api.js` → `api.ts` with typed return signatures
- Drift caught at: dev time (TS) AND runtime (Zod)

**Why not OpenAPI:** Over-engineered for the current team size. Adds spec maintenance burden before the API is stable.

---

## File Structure

```
dashboard/src/
  types/
    api.ts          ← TypeScript interfaces (mirrors Go structs)
  schemas/
    agents.ts       ← Zod schema for agent responses
    jobs.ts         ← Zod schema for job responses
    auth.ts         ← Zod schema for auth/token responses
    groups.ts       ← Zod schema for group responses
  api.ts            ← renamed from api.js, typed, uses schemas
```

---

## Implementation Plan

### Pre-flight
- [ ] Confirm `dashboard/src/api.js` exists and 401 interceptor is present (from Session 5)
- [ ] Confirm `npm run dev` builds clean
- [ ] Confirm `go test ./...` passes clean
- [ ] Branch: `git checkout -b phase-22-api-contract`

---

### Task 1 — Install Zod
```powershell
cd C:\Projects\ArcVault2.0\dashboard
npm install zod
```
Verify: `zod` appears in `package.json` dependencies.

---

### Task 2 — Audit current API response shapes
Read `coordinator/` Go handler files and catalog every response struct. For each endpoint, note:
- HTTP method + path
- Response fields and types

Endpoints to cover (adjust if routes differ):
- `GET /api/agents` → agent list
- `GET /api/jobs` → job list
- `POST /api/jobs` → job create
- `GET /api/groups` → group list
- `POST /api/auth/login` → token response
- `GET /api/status` → coordinator status

---

### Task 3 — Create `dashboard/src/types/api.ts`
Hand-written TypeScript interfaces mirroring Go structs found in Task 2.

```typescript
// Example
export interface Agent {
  id: string
  name: string
  status: 'online' | 'offline' | 'unknown'
  lastSeen: string
  groupId?: string
}

export interface Job {
  id: string
  name: string
  status: 'pending' | 'running' | 'success' | 'failed'
  agentId: string
  dispatchId?: string
  createdAt: string
}

export interface AuthToken {
  token: string
  expiresAt: string
}

// ... all others
```

---

### Task 4 — Create Zod schemas in `dashboard/src/schemas/`

One file per domain. Each schema mirrors the TypeScript interface.

```typescript
// schemas/agents.ts
import { z } from 'zod'

export const AgentSchema = z.object({
  id: z.string(),
  name: z.string(),
  status: z.enum(['online', 'offline', 'unknown']),
  lastSeen: z.string(),
  groupId: z.string().optional()
})

export const AgentListSchema = z.array(AgentSchema)
```

Do the same for: `jobs.ts`, `auth.ts`, `groups.ts`, `status.ts`.

---

### Task 5 — Create `dashboard/src/api.ts` (replaces `api.js`)

Rename `api.js` → `api.ts`. Add:
1. Typed return signatures using interfaces from `types/api.ts`
2. A `safeparse` wrapper that validates responses via Zod before returning
3. On validation failure: log the mismatch with field-level detail, throw a typed `ApiContractError`

```typescript
import { AgentListSchema } from './schemas/agents'

async function fetchAgents(): Promise<Agent[]> {
  const res = await apiClient.get('/api/agents')
  const result = AgentListSchema.safeParse(res.data)
  if (!result.success) {
    console.error('[API Contract] /api/agents shape mismatch:', result.error.format())
    throw new ApiContractError('/api/agents', result.error)
  }
  return result.data
}
```

Retain existing 401 interceptor from Session 5 — do not remove or replace it.

---

### Task 6 — Update Vue components to use typed API

Anywhere a component currently calls `api.js` methods and destructures raw response fields:
- Update import to `api.ts`
- Remove manual field assumptions (`res.data.agent_id` style) — use typed return values
- Add error boundary for `ApiContractError` with a visible console warning

---

### Task 7 — Add a dev-time drift check script

`dashboard/scripts/check-contract.ts` — a small script that:
1. Hits the running coordinator's endpoints
2. Runs each response through its Zod schema
3. Prints PASS / FAIL with field-level detail

Run before every release. Not automated CI yet — manual step for now.

```powershell
npx tsx dashboard/scripts/check-contract.ts
```

---

### Task 8 — Go struct comment annotations

In each Go response struct, add a comment block:

```go
// APIContract: matches dashboard/src/types/api.ts Agent interface
// Last synced: 2026-06-03
type AgentResponse struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Status   string `json:"status"`
    LastSeen string `json:"lastSeen"`
    GroupID  string `json:"groupId,omitempty"`
}
```

This makes drift visible during Go code review — no tooling needed.

---

### Task 9 — Verify and close

```powershell
cd C:\Projects\ArcVault2.0\dashboard
npm run build        # must compile clean with no TS errors
npm run dev          # dashboard loads, no console contract errors
```

```powershell
cd C:\Projects\ArcVault2.0
go test ./...        # coordinator tests still pass
```

Manual smoke test:
- [ ] Login → token validated by `AuthSchema`
- [ ] Agent list loads → each agent validated by `AgentSchema`
- [ ] Job list loads → each job validated by `JobSchema`
- [ ] Intentionally break a Go struct field name → Zod logs contract mismatch in browser console

---

### Task 10 — Commit and tag

```powershell
git add -A
git commit -m "feat: Phase 22 - API contract layer (TypeScript types + Zod validation)"
git checkout main
git merge phase-22-api-contract
git tag v0.3.0
git push origin main --tags
```

---

## Done

Phase 22 is complete when:
- [ ] `npm run build` compiles with zero TypeScript errors
- [ ] Every API response passes its Zod schema on a clean dashboard load
- [ ] A deliberate Go struct rename causes a visible Zod error in the console
- [ ] `go test ./...` still passes
- [ ] Committed and tagged `v0.3.0`
