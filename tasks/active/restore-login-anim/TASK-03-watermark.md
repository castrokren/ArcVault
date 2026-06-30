# TASK 3: Watermark Implementation

**Estimate:** 25 min  
**Owner:** @sofia  
**Status:** Not Started  
**Dependencies:** TASK-01-analysis (complete)  
**Blocks:** TASK-07-integration-testing

## Description
Add the shield watermark SVG to Login.vue. A large, barely-visible shield icon that drifts slowly behind the login card.

## Implementation

### 3.1 Add Shield SVG
Create an SVG shield path matching ArcVault's brand icon:
```vue
<div class="login-watermark" v-if="!prefersReducedMotion">
  <svg viewBox="0 0 560 560" fill="none" xmlns="http://www.w3.org/2000/svg">
    <path
      d="M280 70L490 175V280C490 392 406 476 280 504C154 476 70 392 70 280V175L280 70Z"
      stroke="currentColor"
      stroke-width="2"
      opacity="0.05"
    />
    <path
      d="M210 280L252 322L350 238"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
      opacity="0.05"
    />
  </svg>
</div>
```

**File:** `dashboard/src/views/Login.vue` (template, between aurora and stars)

### 3.2 Verify Rendering
- Shield visible at center (barely, watermark opacity)
- Drift animation running (18s loop)
- No z-index conflicts

## Acceptance Criteria
- [ ] Watermark SVG renders
- [ ] Hidden when prefersReducedMotion
- [ ] Drift animation runs correctly
- [ ] Opacity is subtle (barely visible)
