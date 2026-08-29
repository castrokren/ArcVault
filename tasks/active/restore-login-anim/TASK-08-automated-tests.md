# TASK 8: Automated Tests

**Estimate:** 30 min  
**Owner:** @aisha  
**Status:** Not Started  
**Dependencies:** TASK-07-integration-testing (complete)  
**Blocks:** TASK-09-docs-handoff

## Description
Write Vitest tests for the new Login.vue animation elements.

## Test Cases

### 8.1 Aurora (3 min)
- `renders 4 aurora blobs when motion not reduced`
- `hides aurora blobs when prefers-reduced-motion`

### 8.2 Watermark (3 min)
- `renders watermark with shield SVG`
- `hides watermark when prefers-reduced-motion`

### 8.3 Starfield (4 min)
- `renders 25 stars when motion not reduced`
- `hides stars when prefers-reduced-motion`
- `each star has unique position styles`

### 8.4 Beams (4 min)
- `renders 12 beams when motion not reduced`
- `hides beams when prefers-reduced-motion`
- `beams have correct rotation angles`

### 8.5 Halos (4 min)
- `renders outer and inner halos when motion not reduced`
- `hides halos when prefers-reduced-motion`
- `icon pulse still renders on reduced-motion`

### 8.6 Shell Entrance (2 min)
- `login-shell has animation style applied`

### 8.7 Full Suite Run (10 min)
```bash
cd dashboard && npx vitest run
```
- Expected: 63 + 6-7 new = 69-70 total, all pass

## Acceptance Criteria
- [ ] Tests cover all 6 animation systems
- [ ] Reduced-motion tests for decorative animations
- [ ] All 63 existing tests still pass
- [ ] All new tests pass
