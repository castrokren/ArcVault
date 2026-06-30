# TASK 2: Aurora Blobs Implementation

**Estimate:** 20 min  
**Owner:** @sofia  
**Status:** Not Started  
**Dependencies:** TASK-01-analysis (complete)  
**Blocks:** TASK-07-integration-testing

## Description
Add the 4 aurora gradient-blob divs to Login.vue template. These are simple positioned divs with no children and no inline styles — all animation is in the CSS.

## Implementation

### 2.1 Add Elements to Template
Insert after OrbitField, before .login-shell:
```vue
<div class="login-aurora login-aurora-1" v-if="!prefersReducedMotion"></div>
<div class="login-aurora login-aurora-2" v-if="!prefersReducedMotion"></div>
<div class="login-aurora login-aurora-3" v-if="!prefersReducedMotion"></div>
<div class="login-aurora login-aurora-4" v-if="!prefersReducedMotion"></div>
```

**File:** `dashboard/src/views/Login.vue` (template section, after <OrbitField>)

### 2.2 Verify Rendering
- 4 gradient blobs visible in browser
- Blur filter (60px) applied
- Drift animations running (16-26s loops)
- Positioned at corners correctly
- No z-index conflicts with canvas or card

## Acceptance Criteria
- [ ] 4 aurora divs render
- [ ] Hidden when prefersReducedMotion
- [ ] Aurora animations run correctly
- [ ] No visual glitches with OrbitField underneath
