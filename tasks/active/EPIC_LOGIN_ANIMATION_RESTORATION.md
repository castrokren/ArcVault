# EPIC: Restore Login Page CSS Animation Layers

**ID:** EPIC-LOGIN-ANIM-001  
**Type:** Epic  
**Priority:** High  
**Status:** Planning  
**Created:** 2026-06-29  
**Owner:** James (Orchestrator)  
**Workspace:** ArcVault2.0

---

## Executive Summary

**Problem:** The `login-animation.css` file (180 lines) defines visual effects for the login page — aurora blobs, starfield, data-comet beams, shield watermark, and animated halos around the brand icon. These CSS classes are globally imported in `main.js:17` but have **no DOM elements to attach to** in Login.vue after the OrbitField migration. The visual effects are orphaned and not rendering.

**Goal:** Restore these CSS animation effects by re-integrating the required HTML elements into Login.vue's template. The effects should layer alongside OrbitField's canvas animation to create a rich, atmospheric login experience.

**Scope:** 5 major animation systems + 1 entrance animation

**Success Criteria:**
- All 6 animation systems render correctly on the login page
- Visual effects layer properly (z-index hierarchy: canvas → CSS effects → login card)
- No conflicts between CSS animations and OrbitField/motion-v
- Reduced-motion preference respected for all new elements
- Performance: 60fps on high tier, graceful degradation on low tier
- All tests pass (63 JS, 175 Go)
- Build succeeds with no errors

---

## Architecture Overview

### Current State (Broken)

```
<div class="login-page">
  <OrbitField />              ← Canvas orbit animation (z-index: 0)
  <div class="login-shell">   ← Card container (z-index: 1)
    <brand>, <card>, <form>, <footer>
  </div>
</div>
```

**CSS classes defined but unused:**
- `.login-aurora-1` through `.login-aurora-4` (4 gradient blur blobs)
- `.login-watermark` + SVG (shield watermark)
- `.login-stars` + `.login-star` (starfield container + stars)
- `.login-beams` + `.login-beam-wrap` + `.login-beam` (comet beams)
- `.login-brand-icon-wrap` + `.login-halo-outer` + `.login-halo-inner` (halo rings)
- `.login-shell` (entrance animation) — ONLY ONE CURRENTLY USED

### Target State (Fixed)

```
<div class="login-page">
  <!-- Layer 0: Canvas base -->
  <OrbitField motion="bold" />
  
  <!-- Layer 0: CSS atmosphere effects -->
  <div class="login-aurora login-aurora-1"></div>
  <div class="login-aurora login-aurora-2"></div>
  <div class="login-aurora login-aurora-3"></div>
  <div class="login-aurora login-aurora-4"></div>
  
  <div class="login-watermark">
    <svg>...</svg>
  </div>
  
  <div class="login-stars">
    <div class="login-star" style="left:X%; top:Y%; animation-delay:Zs;"></div>
    <!-- ...more stars (20-30 total) -->
  </div>
  
  <div class="login-beams">
    <div class="login-beam-wrap" style="left:X%; top:Y%; transform:rotate(Adeg);">
      <div class="login-beam"></div>
    </div>
    <!-- ...more beams (12 total converging on logo) -->
  </div>
  
  <!-- Layer 1: Card with halo-enhanced brand -->
  <div class="login-shell">
    <div class="login-brand-icon-wrap">
      <div class="login-halo-outer"></div>
      <div class="login-halo-inner"></div>
      <div class="login-brand-icon">
        <svg>...</svg>
      </div>
    </div>
    <brand>, <card>, <form>, <footer>
  </div>
</div>
```

---

## Task Hierarchy

### TASK 1: Analysis & Design (Est: 45 min)
**Owner:** @maya (Requirements) + @david (Architecture)  
**Status:** Not Started  
**Dependencies:** None  
**Blocks:** TASK 2, TASK 3, TASK 4, TASK 5, TASK 6

#### TASK 1.1: Audit CSS Classes & Keyframes
**Owner:** @maya  
**Est:** 15 min  
**Deliverable:** `tasks/active/login-animation-audit.md`

- **TASK 1.1.1:** Read full `login-animation.css` (180 lines)
  - Document all class selectors (20 total)
  - Document all @keyframes (8 total)
  - Map classes → keyframes dependencies

- **TASK 1.1.2:** Cross-reference with Login.vue current state
  - Identify which classes are currently used (`.login-shell` only)
  - Identify which are orphaned (19 classes)
  - Document z-index requirements for layering

- **TASK 1.1.3:** Analyze animation timing & triggers
  - Aurora: 16-26s infinite ease-in-out loops
  - Stars: Twinkle pattern (inline delays)
  - Beams: 3s converging rush
  - Watermark: 18s drift
  - Halos: 2.3-4.8s spin + 2.8s pulse
  - Shell: 0.55s entrance

#### TASK 1.2: Template Reconstruction Planning
**Owner:** @maya  
**Est:** 20 min  
**Deliverable:** `tasks/active/login-animation-template-plan.md`

- **TASK 1.2.1:** Aurora blobs — determine element structure
  - 4 absolute-positioned divs
  - No children needed
  - No inline styles needed (all in CSS)

- **TASK 1.2.2:** Watermark — determine SVG source & positioning
  - Single container div
  - SVG child (shield icon) — need to define path
  - Inline styles: left/top centering

- **TASK 1.2.3:** Starfield — determine star count & distribution
  - Container div `.login-stars`
  - 20-30 `.login-star` children
  - Inline styles per star: `left`, `top`, `animation-delay`
  - Random distribution algorithm (or fixed positions?)

- **TASK 1.2.4:** Beams — determine beam count & angle distribution
  - Container div `.login-beams`
  - 12 beam pairs (outer `.login-beam-wrap` + inner `.login-beam`)
  - Inline styles per beam: `left`, `top` (logo center), `transform: rotate(Xdeg)`
  - Angles: 0°, 30°, 60°, 90°, 120°, 150°, 180°, 210°, 240°, 270°, 300°, 330°

- **TASK 1.2.5:** Brand icon halos — determine wrapping structure
  - Replace current `.brand-icon` with `.login-brand-icon-wrap`
  - Children: `.login-halo-outer`, `.login-halo-inner`, `.login-brand-icon`
  - Merge with existing brand SVG

#### TASK 1.3: Conflict & Integration Analysis
**Owner:** @david  
**Est:** 10 min  
**Deliverable:** Risk assessment in `login-animation-template-plan.md`

- **TASK 1.3.1:** OrbitField canvas compatibility
  - Both are `position: fixed; inset: 0; z-index: 0; pointer-events: none`
  - Canvas renders first, CSS effects layer on top
  - No interaction conflicts

- **TASK 1.3.2:** motion-v spring animations compatibility
  - motion-v animates `.login-shell` children (brand, card, form)
  - CSS `.login-shell` provides container entrance only
  - No property conflicts (CSS = animation, scoped = layout)

- **TASK 1.3.3:** Reduced-motion handling
  - OrbitField already respects `prefers-reduced-motion`
  - New CSS elements need conditional rendering or `animation: none`
  - Decision: Conditionally render vs CSS media query

- **TASK 1.3.4:** Performance impact assessment
  - 4 aurora blobs with blur filters
  - 20-30 stars with opacity animations
  - 12 beams with translateX animations
  - 2 halos with rotate animations
  - Total: ~40 animated elements + OrbitField canvas
  - Risk: Medium on low-tier devices

---

### TASK 2: Aurora Blobs Implementation (Est: 20 min)
**Owner:** @sofia (Frontend Implementation)  
**Status:** Not Started  
**Dependencies:** TASK 1 (complete)  
**Blocks:** TASK 7 (Integration Testing)

#### TASK 2.1: Add Aurora Elements to Login.vue Template
**Est:** 10 min  
**File:** `dashboard/src/views/Login.vue`

- **TASK 2.1.1:** Insert 4 aurora divs after OrbitField, before login-shell
  ```vue
  <div class="login-aurora login-aurora-1"></div>
  <div class="login-aurora login-aurora-2"></div>
  <div class="login-aurora login-aurora-3"></div>
  <div class="login-aurora login-aurora-4"></div>
  ```

- **TASK 2.1.2:** Add reduced-motion conditional
  - Option A: `v-if="!prefersReducedMotion"` on each div
  - Option B: CSS media query in `login-animation.css`
  - Decision pending TASK 1.3.3

#### TASK 2.2: Verify Aurora Rendering
**Est:** 10 min  
**Owner:** @sofia

- **TASK 2.2.1:** Visual inspection in browser
  - 4 gradient blobs visible
  - Blur filter applied (60px)
  - Animations running (20-26s loops)
  - Positioned correctly (absolute, corners)

- **TASK 2.2.2:** Performance check
  - Chrome DevTools Performance tab
  - Target: 60fps on high tier
  - Verify: No layout thrashing

---

### TASK 3: Watermark Implementation (Est: 25 min)
**Owner:** @sofia (Frontend Implementation)  
**Status:** Not Started  
**Dependencies:** TASK 1 (complete)  
**Blocks:** TASK 7 (Integration Testing)

#### TASK 3.1: Design Shield SVG
**Est:** 10 min  
**Owner:** @sofia

- **TASK 3.1.1:** Create shield icon SVG path
  - Match ArcVault brand (shield with checkmark)
  - Large size: 560x560 viewBox
  - Stroke-only, no fill
  - Color: `rgba(139,92,246,.05)` (very subtle)

- **TASK 3.1.2:** Add opacity/blur for watermark effect
  - CSS already has `opacity` in `avWatermarkDrift` keyframes
  - Verify subtle visibility (barely visible, drift effect)

#### TASK 3.2: Add Watermark Element to Login.vue Template
**Est:** 10 min  
**File:** `dashboard/src/views/Login.vue`

- **TASK 3.2.1:** Insert watermark div after aurora, before stars
  ```vue
  <div class="login-watermark">
    <svg viewBox="0 0 560 560" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="..." stroke="rgba(139,92,246,.05)" stroke-width="2" />
    </svg>
  </div>
  ```

- **TASK 3.2.2:** Add reduced-motion conditional (same as aurora)

#### TASK 3.3: Verify Watermark Rendering
**Est:** 5 min  
**Owner:** @sofia

- **TASK 3.3.1:** Visual inspection
  - Shield visible at center (barely, watermark opacity)
  - Drift animation running (18s loop, scale + rotate)
  - No z-index conflicts

---

### TASK 4: Starfield Implementation (Est: 35 min)
**Owner:** @sofia (Frontend Implementation)  
**Status:** Not Started  
**Dependencies:** TASK 1 (complete)  
**Blocks:** TASK 7 (Integration Testing)

#### TASK 4.1: Generate Star Positions & Delays
**Est:** 15 min  
**Owner:** @sofia

- **TASK 4.1.1:** Decide: Random generation vs fixed positions
  - Option A: Fixed array of 25 positions (deterministic, consistent)
  - Option B: Random on mount (dynamic, different each load)
  - Recommendation: Fixed for consistency

- **TASK 4.1.2:** Define 25 star positions
  - Spread across viewport (avoid clustering)
  - Inline style format: `left: X%; top: Y%;`
  - Example: `{ left: 12, top: 8 }`, `{ left: 45, top: 22 }`, ...

- **TASK 4.1.3:** Assign animation delays
  - Inline style: `animation: avTwinkle 3.2s ease-in-out 1.4s infinite;`
  - Random delays: 0-4s range
  - Mix of 2px and 3px stars (`.login-star` vs `.login-star.lg`)

#### TASK 4.2: Add Starfield Elements to Login.vue Template
**Est:** 15 min  
**File:** `dashboard/src/views/Login.vue`

- **TASK 4.2.1:** Insert stars container after watermark, before beams
  ```vue
  <div class="login-stars" v-if="!prefersReducedMotion">
    <div class="login-star" :style="{ left: '12%', top: '8%', animation: 'avTwinkle 3.2s ease-in-out 1.4s infinite' }"></div>
    <div class="login-star lg" :style="{ left: '45%', top: '22%', animation: 'avTwinkle 2.8s ease-in-out 0.6s infinite' }"></div>
    <!-- ...23 more stars -->
  </div>
  ```

- **TASK 4.2.2:** Consider script-based generation
  - Alternative: Define stars array in `<script setup>`, v-for loop
  - Cleaner template, easier to adjust count
  - Example:
    ```js
    const stars = [
      { left: 12, top: 8, delay: 1.4, duration: 3.2, large: false },
      { left: 45, top: 22, delay: 0.6, duration: 2.8, large: true },
      // ...
    ]
    ```
  - Template:
    ```vue
    <div class="login-stars" v-if="!prefersReducedMotion">
      <div
        v-for="(star, i) in stars"
        :key="i"
        :class="['login-star', { lg: star.large }]"
        :style="{
          left: `${star.left}%`,
          top: `${star.top}%`,
          animation: `avTwinkle ${star.duration}s ease-in-out ${star.delay}s infinite`
        }"
      ></div>
    </div>
    ```

#### TASK 4.3: Verify Starfield Rendering
**Est:** 5 min  
**Owner:** @sofia

- **TASK 4.3.1:** Visual inspection
  - 25 stars visible
  - Twinkle animations staggered (not all in sync)
  - Mix of 2px and 3px stars
  - Color correct: `#d8ccff`

---

### TASK 5: Beams Implementation (Est: 40 min)
**Owner:** @sofia (Frontend Implementation)  
**Status:** Not Started  
**Dependencies:** TASK 1 (complete)  
**Blocks:** TASK 7 (Integration Testing)

#### TASK 5.1: Calculate Beam Angles & Logo Position
**Est:** 15 min  
**Owner:** @sofia

- **TASK 5.1.1:** Determine logo center position
  - Logo is in `.brand` (inside `.login-shell`)
  - `.login-shell` is centered: `align-items: center; justify-content: center`
  - Logo position: approximately `left: 50%; top: 26%` (estimate, verify in browser)
  - Store in script: `const logoCenter = { left: '50%', top: '26%' }`

- **TASK 5.1.2:** Define 12 beam angles
  - Converging on logo from all directions
  - Angles: `[0, 30, 60, 90, 120, 150, 180, 210, 240, 270, 300, 330]` degrees
  - Each beam: `transform-origin: 0 0; transform: rotate(Xdeg);`

- **TASK 5.1.3:** Define beam widths (thin/thick mix)
  - CSS has `.login-beam.thin` (1px) and `.login-beam.thick` (2px)
  - Default is 1.5px
  - Assign randomly or pattern: 8 normal, 2 thin, 2 thick

#### TASK 5.2: Add Beams Elements to Login.vue Template
**Est:** 20 min  
**File:** `dashboard/src/views/Login.vue`

- **TASK 5.2.1:** Define beams array in script
  ```js
  const beams = [
    { angle: 0, width: 'normal' },
    { angle: 30, width: 'thin' },
    { angle: 60, width: 'normal' },
    { angle: 90, width: 'thick' },
    // ... 8 more
  ]
  const logoCenter = { left: '50%', top: '26%' }
  ```

- **TASK 5.2.2:** Insert beams container after stars, before login-shell
  ```vue
  <div class="login-beams" v-if="!prefersReducedMotion">
    <div
      v-for="(beam, i) in beams"
      :key="i"
      class="login-beam-wrap"
      :style="{
        left: logoCenter.left,
        top: logoCenter.top,
        transform: `rotate(${beam.angle}deg)`
      }"
    >
      <div
        :class="['login-beam', beam.width]"
      ></div>
    </div>
  </div>
  ```

#### TASK 5.3: Verify Beams Rendering
**Est:** 5 min  
**Owner:** @sofia

- **TASK 5.3.1:** Visual inspection
  - 12 beams visible
  - Converging on logo from all angles (radial pattern)
  - Animation: Rush inward (280px → 2px, 3s loop)
  - Gradient correct: transparent → `rgba(196,181,253,.75)` → `rgba(240,232,255,.95)`

- **TASK 5.3.2:** Adjust logo center if needed
  - If beams don't converge perfectly, adjust `logoCenter` position
  - Use browser inspector to measure exact logo position

---

### TASK 6: Brand Icon Halo Implementation (Est: 30 min)
**Owner:** @sofia (Frontend Implementation)  
**Status:** Not Started  
**Dependencies:** TASK 1 (complete)  
**Blocks:** TASK 7 (Integration Testing)

#### TASK 6.1: Refactor Brand Icon Structure
**Est:** 20 min  
**File:** `dashboard/src/views/Login.vue`

- **TASK 6.1.1:** Replace current `.brand-icon` with `.login-brand-icon-wrap`
  - Current structure:
    ```vue
    <div class="brand">
      <div class="brand-icon">
        <svg>...</svg>
      </div>
      <span class="brand-name">ArcVault</span>
    </div>
    ```
  - New structure:
    ```vue
    <div class="brand">
      <div class="login-brand-icon-wrap">
        <div class="login-halo-outer"></div>
        <div class="login-halo-inner"></div>
        <div class="login-brand-icon">
          <svg>...</svg>
        </div>
      </div>
      <span class="brand-name">ArcVault</span>
    </div>
    ```

- **TASK 6.1.2:** Update scoped CSS for `.brand-icon`
  - Current scoped `.brand-icon` sets layout (36px, border-radius, bg, border)
  - Global `.login-brand-icon` sets animation (avIconPulse pulse)
  - Merge or rename to avoid conflict
  - Option A: Rename scoped `.brand-icon` to `.brand-icon-local` (keep both)
  - Option B: Remove scoped `.brand-icon`, use only global `.login-brand-icon` (risky, styles differ)
  - Recommendation: Keep scoped for layout, global for animation (non-conflicting properties)

- **TASK 6.1.3:** Add reduced-motion conditional for halos only
  - Halos (spinning rings) should hide on reduced-motion
  - Icon pulse can stay (subtle box-shadow change)
  - Conditional: `v-if="!prefersReducedMotion"` on `.login-halo-outer` and `.login-halo-inner`

#### TASK 6.2: Verify Halo Rendering
**Est:** 10 min  
**Owner:** @sofia

- **TASK 6.2.1:** Visual inspection
  - 2 rings visible around brand icon
  - Outer ring: 1px border, partial arc (top + right), 4.8s counter-clockwise spin
  - Inner ring: 2px border, partial arc (top + right), 2.3s clockwise spin
  - Icon pulse: Box-shadow glow (18px → 40px, 2.8s loop)

- **TASK 6.2.2:** Check scoped CSS conflicts
  - Verify `.brand-icon` scoped styles still apply (layout)
  - Verify `.login-brand-icon` global styles apply (animation)
  - No property collisions

---

### TASK 7: Integration & Testing (Est: 45 min)
**Owner:** @aisha (QA & Verification)  
**Status:** Not Started  
**Dependencies:** TASK 2, TASK 3, TASK 4, TASK 5, TASK 6 (all complete)  
**Blocks:** TASK 8 (Documentation)

#### TASK 7.1: Visual QA — All Effects Together
**Est:** 15 min  
**Owner:** @aisha

- **TASK 7.1.1:** Full-page visual inspection
  - All 6 animation systems rendering simultaneously
  - Z-index layering correct: canvas → aurora → watermark → stars → beams → shell (card)
  - No visual glitches or overlap conflicts

- **TASK 7.1.2:** Animation timing verification
  - Aurora: Slow drift (16-26s loops)
  - Watermark: Slow drift (18s loop)
  - Stars: Twinkle stagger (varied delays)
  - Beams: Fast rush (3s loop)
  - Halos: Medium spin (2.3-4.8s loops)
  - Shell: One-time entrance (0.55s)

#### TASK 7.2: Reduced-Motion Testing
**Est:** 10 min  
**Owner:** @aisha

- **TASK 7.2.1:** Enable `prefers-reduced-motion: reduce` in OS/browser
  - macOS: System Preferences → Accessibility → Display → Reduce motion
  - Windows: Settings → Ease of Access → Display → Show animations
  - Chrome DevTools: Rendering tab → Emulate CSS media feature

- **TASK 7.2.2:** Verify behavior
  - Aurora, watermark, stars, beams, halos: Not rendered (`v-if="!prefersReducedMotion"`)
  - OrbitField: Static frame (already respects reduced-motion)
  - motion-v: Instant appearance (no spring animations)
  - Shell entrance: Instant (or CSS media query disables animation)

#### TASK 7.3: Performance Testing
**Est:** 10 min  
**Owner:** @aisha

- **TASK 7.3.1:** High-tier device (modern desktop/laptop)
  - Chrome DevTools → Performance tab → Record 10s
  - Target: 60fps sustained
  - No dropped frames, no layout thrashing

- **TASK 7.3.2:** Low-tier device simulation
  - Chrome DevTools → Performance tab → CPU throttling 4x slowdown
  - OrbitField should degrade to static (currentTier: 'low')
  - CSS animations should continue (or optionally disable on low tier)
  - Target: No browser freeze, UI remains responsive

- **TASK 7.3.3:** Memory usage
  - Chrome DevTools → Memory tab → Heap snapshot
  - Verify no memory leaks (navigate away, back, check delta)

#### TASK 7.4: Cross-Browser Testing
**Est:** 10 min  
**Owner:** @aisha

- **TASK 7.4.1:** Chrome (primary)
  - All animations render correctly
  - 60fps performance

- **TASK 7.4.2:** Firefox
  - Verify backdrop-filter, blur, animations work
  - Known issue: backdrop-filter requires flag (check support)

- **TASK 7.4.3:** Edge (Chromium)
  - Should match Chrome behavior

- **TASK 7.4.4:** Safari (if macOS available)
  - Verify webkit-backdrop-filter fallback
  - Check animation performance

---

### TASK 8: Automated Testing (Est: 30 min)
**Owner:** @aisha (Test Creation)  
**Status:** Not Started  
**Dependencies:** TASK 7 (complete)  
**Blocks:** TASK 9 (Documentation)

#### TASK 8.1: Vitest Component Tests for Login.vue
**Est:** 20 min  
**File:** `dashboard/src/views/Login.test.js` (create if doesn't exist)

- **TASK 8.1.1:** Test: Aurora elements render
  ```js
  it('renders 4 aurora blobs when motion not reduced', () => {
    const wrapper = mount(Login, { /* ... */ })
    expect(wrapper.findAll('.login-aurora')).toHaveLength(4)
  })
  ```

- **TASK 8.1.2:** Test: Watermark renders
  ```js
  it('renders watermark with shield SVG', () => {
    const wrapper = mount(Login)
    expect(wrapper.find('.login-watermark svg').exists()).toBe(true)
  })
  ```

- **TASK 8.1.3:** Test: Starfield renders 25 stars
  ```js
  it('renders 25 stars when motion not reduced', () => {
    const wrapper = mount(Login)
    expect(wrapper.findAll('.login-star')).toHaveLength(25)
  })
  ```

- **TASK 8.1.4:** Test: Beams render 12 beams
  ```js
  it('renders 12 beams when motion not reduced', () => {
    const wrapper = mount(Login)
    expect(wrapper.findAll('.login-beam-wrap')).toHaveLength(12)
  })
  ```

- **TASK 8.1.5:** Test: Halos render around brand icon
  ```js
  it('renders outer and inner halos when motion not reduced', () => {
    const wrapper = mount(Login)
    expect(wrapper.find('.login-halo-outer').exists()).toBe(true)
    expect(wrapper.find('.login-halo-inner').exists()).toBe(true)
  })
  ```

- **TASK 8.1.6:** Test: Reduced-motion hides animations
  ```js
  it('hides CSS animations when prefers-reduced-motion', () => {
    // Mock window.matchMedia to return true
    window.matchMedia = vi.fn().mockImplementation(query => ({
      matches: query.includes('prefers-reduced-motion'),
    }))
    const wrapper = mount(Login)
    expect(wrapper.findAll('.login-aurora')).toHaveLength(0)
    expect(wrapper.findAll('.login-star')).toHaveLength(0)
    expect(wrapper.findAll('.login-beam')).toHaveLength(0)
    expect(wrapper.find('.login-halo-outer').exists()).toBe(false)
  })
  ```

#### TASK 8.2: Run Full Test Suite
**Est:** 10 min  
**Owner:** @aisha

- **TASK 8.2.1:** Run JS tests
  ```bash
  cd dashboard
  npx vitest run
  ```
  - Target: 63/63 + new tests pass (68-70 total expected)

- **TASK 8.2.2:** Run Go tests (regression check)
  ```bash
  go test ./...
  ```
  - Target: 175/175 pass (no regressions)

- **TASK 8.2.3:** Build dashboard
  ```bash
  cd dashboard
  npm run build
  ```
  - Target: Build succeeds, no errors, 546+ modules

---

### TASK 9: Documentation & Handoff (Est: 20 min)
**Owner:** @elena (Code Review) + James (Orchestrator)  
**Status:** Not Started  
**Dependencies:** TASK 8 (complete)  
**Blocks:** None (Final task)

#### TASK 9.1: Code Review
**Est:** 10 min  
**Owner:** @elena

- **TASK 9.1.1:** Review Login.vue template changes
  - Verify structure matches plan
  - Check for hardcoded values (should use constants where possible)
  - Verify reduced-motion conditionals

- **TASK 9.1.2:** Review script additions (stars/beams arrays)
  - Check for magic numbers
  - Verify array definitions are clear and maintainable

- **TASK 9.1.3:** Review scoped CSS changes (brand-icon merge)
  - Verify no conflicts between scoped and global CSS
  - Check for specificity issues

- **TASK 9.1.4:** Review test coverage
  - All 6 animation systems have tests
  - Reduced-motion test covers all conditionals

#### TASK 9.2: Update Project Documentation
**Est:** 5 min  
**Owner:** James

- **TASK 9.2.1:** Update `CONTEXT.md`
  - Add note: "Login page CSS animations restored (aurora, stars, beams, watermark, halos)"
  - Document reduced-motion behavior

- **TASK 9.2.2:** Update `memory/decisions.md`
  - Record decision: CSS animations layer alongside OrbitField
  - Record pattern: Reduced-motion conditionals on decorative elements

#### TASK 9.3: Create Completion Summary
**Est:** 5 min  
**Owner:** James  
**Deliverable:** `tasks/completed/EPIC_LOGIN_ANIMATION_RESTORATION_SUMMARY.md`

- **TASK 9.3.1:** Summary of changes
  - Files modified: `Login.vue`, `login-animation.css` (optionally)
  - Lines added: ~100-150 (template + script)
  - Tests added: 6-7 new tests

- **TASK 9.3.2:** Verification results
  - JS tests: X/X pass
  - Go tests: 175/175 pass
  - Build: Success
  - Visual QA: Pass
  - Performance: Pass

- **TASK 9.3.3:** Next steps
  - Move task to `tasks/completed/`
  - Archive planning docs
  - Ready for installer build

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Performance degradation on low-tier devices | Medium | High | OrbitField already degrades to static; CSS animations can be disabled via media query or script |
| Z-index conflicts between layers | Low | Medium | Clearly defined layering: canvas (0) → CSS effects (0) → card (1) |
| Reduced-motion not fully respected | Low | High | Use `v-if="!prefersReducedMotion"` on all decorative animation elements |
| CSS animation timing feels wrong vs OrbitField | Medium | Low | Can adjust animation speeds in CSS without template changes |
| Beam convergence point misaligned | Medium | Low | Make logo center position a reactive const, easy to adjust |
| Brand icon halo conflicts with scoped styles | Low | Medium | Keep scoped for layout, global for animation (non-overlapping properties) |
| Stars/beams clutter the page too much | Medium | Low | Can reduce count (25 stars → 15, 12 beams → 8) if visually overwhelming |

---

## Acceptance Criteria

- [ ] All 6 animation systems render on login page (aurora, watermark, stars, beams, halos, shell entrance)
- [ ] Z-index layering correct: OrbitField canvas at 0, CSS effects at 0, login card at 1
- [ ] Reduced-motion preference hides all decorative animations (aurora, watermark, stars, beams, halos)
- [ ] 60fps performance on high-tier devices (Chrome DevTools Performance tab, no dropped frames)
- [ ] No browser freeze on low-tier devices (4x CPU throttling, UI remains responsive)
- [ ] All JS tests pass (63 + 6-7 new tests = 69-70 total)
- [ ] All Go tests pass (175/175, no regressions)
- [ ] Dashboard build succeeds with no errors
- [ ] Visual QA pass: All effects render correctly, no glitches
- [ ] Cross-browser compatibility: Chrome, Firefox, Edge (Safari optional)
- [ ] Code review approved by @elena
- [ ] Documentation updated (CONTEXT.md, memory/decisions.md)
- [ ] Completion summary created

---

## Timeline

| Phase | Tasks | Est. Time | Owner(s) |
|-------|-------|-----------|----------|
| **Phase 1: Analysis** | TASK 1 (1.1, 1.2, 1.3) | 45 min | @maya, @david |
| **Phase 2: Implementation** | TASK 2, 3, 4, 5, 6 | 150 min (2.5 hrs) | @sofia |
| **Phase 3: Testing** | TASK 7, 8 | 75 min (1.25 hrs) | @aisha |
| **Phase 4: Review & Docs** | TASK 9 | 20 min | @elena, James |
| **Total** | | **290 min (4.8 hrs)** | |

**Estimated Completion:** 1 full work day (with breaks) or 2 half-days

---

## Next Steps

1. **Kren approval** — Review this plan, approve or request changes
2. **Kick off TASK 1** — Route to @maya for CSS audit and template planning
3. **Daily check-ins** — Progress updates at end of each phase
4. **Final handoff** — Completion summary + installer build instructions

---

**Status:** Awaiting Kren's approval to proceed
