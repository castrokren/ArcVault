# TASK 6: Brand Icon Halos Implementation

**Estimate:** 30 min  
**Owner:** @sofia  
**Status:** Not Started  
**Dependencies:** TASK-01-analysis (complete)  
**Blocks:** TASK-07-integration-testing

## Description
Replace the plain `.brand-icon` div with the `login-brand-icon-wrap` structure that adds animated spinning halo rings around the ArcVault brand icon.

## Implementation

### 6.1 Refactor Brand Icon Template
Replace current:
```vue
<div class="brand-icon">
  <svg>...</svg>
</div>
```
With:
```vue
<div class="login-brand-icon-wrap">
  <div class="login-halo-outer" v-if="!prefersReducedMotion"></div>
  <div class="login-halo-inner" v-if="!prefersReducedMotion"></div>
  <div class="login-brand-icon">
    <svg>...</svg>
  </div>
</div>
```

**File:** `dashboard/src/views/Login.vue` (template, inside `.brand` div)

### 6.2 Verify scoped CSS compatibility
- Scoped `.brand-icon` sets layout (36px, border-radius, bg, border)
- Global `.login-brand-icon` sets animation (box-shadow pulse)
- These handle different properties → no conflict expected
- If conflict arises, rename scoped `.brand-icon` to `.brand-icon-local`

### 6.3 Verify Rendering
- Outer halo: 1px partial arc, 4.8s counter-clockwise spin, drop-shadow
- Inner halo: 2px partial arc, 2.3s clockwise spin, stronger drop-shadow
- Icon pulse: box-shadow glow cycles (18px ↔ 40px, 2.8s)
- Halos hidden on reduced-motion
- Icon pulse still works on reduced-motion (subtle, stays)

## Acceptance Criteria
- [ ] Outer and inner halos render
- [ ] Halos hidden when prefersReducedMotion
- [ ] Icon pulse animation still active on reduced-motion
- [ ] No scoped CSS conflicts
- [ ] Layout unchanged (icon size, spacing, brand alignment)
