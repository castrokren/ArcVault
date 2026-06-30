# TASK 1 Analysis Results — Login CSS Animation Restoration

## 1. CSS Class Audit

Document all 19 class selectors found in `dashboard/src/login-animation.css`:

| # | Class | Type | Animation | Keyframes Used |
|---|-------|------|-----------|----------------|
| 1 | `.login-aurora` | Base class (position, blur) | None (shared) | — |
| 2 | `.login-aurora-1` | Aurora blob | 20s ease-in-out infinite | `avAuroraShift1` |
| 3 | `.login-aurora-2` | Aurora blob | 26s ease-in-out infinite 7s | `avAuroraShift2` |
| 4 | `.login-aurora-3` | Aurora blob | 16s ease-in-out infinite 10s | `avAuroraShift1` |
| 5 | `.login-aurora-4` | Aurora blob | 22s ease-in-out infinite 3s | `avAuroraShift2` |
| 6 | `.login-watermark` | Shield watermark container | 18s ease-in-out infinite | `avWatermarkDrift` |
| 7 | `.login-stars` | Starfield container | None (shared) | — |
| 8 | `.login-star` | Individual star | Via inline style | `avTwinkle` |
| 9 | `.lg` | Large star modifier | Same as `.login-star` | — |
| 10 | `.login-beams` | Beam scene container | None (shared) | — |
| 11 | `.login-beam-wrap` | Beam rotation wrapper | None (rotation via inline style) | — |
| 12 | `.login-beam` | Beam element | 3s ease-in infinite | `avBeamRush` |
| 13 | `.thin` | Thin beam modifier (1px) | Same as `.login-beam` | — |
| 14 | `.thick` | Thick beam modifier (2px) | Same as `.login-beam` | — |
| 15 | `.login-brand-icon-wrap` | Brand icon wrapper for halos | None (shared) | — |
| 16 | `.login-halo-outer` | Outer slow ring | 4.8s linear infinite reverse | `avSpin` |
| 17 | `.login-halo-inner` | Inner fast ring | 2.3s linear infinite | `avSpin` |
| 18 | `.login-brand-icon` | Brand icon (animation) | 2.8s ease-in-out infinite | `avIconPulse` |
| 19 | `.login-shell` | Card shell entrance | 0.55s cubic-bezier both | `avShellIn` |

Note: The task spec says "20 total" — the 20th would be the implicit `.login-watermark svg` descendant selector (not a class) or counting `.login-star.lg` as a separate compound selector. The audit found 19 unique class names.

## 2. @keyframes Audit (8 total)

| # | Keyframe Name | Lines | Used By | Description |
|---|---------------|-------|---------|-------------|
| 1 | `avAuroraShift1` | 147-151 | `.login-aurora-1`, `.login-aurora-3` | translate + scale + opacity drift |
| 2 | `avAuroraShift2` | 152-156 | `.login-aurora-2`, `.login-aurora-4` | translate + scale + opacity drift (alt) |
| 3 | `avWatermarkDrift` | 157-160 | `.login-watermark` | Subtle scale(1→1.04) + rotate(0→1.2deg) + opacity |
| 4 | `avBeamRush` | 161-166 | `.login-beam` | translateX(280px→2px) with opacity fade |
| 5 | `avIconPulse` | 167-170 | `.login-brand-icon` | box-shadow glow expansion (18px→40px) |
| 6 | `avSpin` | 171 | `.login-halo-outer`(reverse), `.login-halo-inner` | 360deg rotation |
| 7 | `avShellIn` | 172-175 | `.login-shell` | translateY(16px) + opacity entrance |
| 8 | `avTwinkle` | 176-179 | `.login-star` (via inline style) | Opacity pulse (0.12↔0.85) |

## 3. Live vs Orphaned Classes

**Currently LIVE in Login.vue template:**
- `.login-shell` — line 7 of template, entrance animation works
- `.brand-icon` — line 15 of template, **BUT this is SCOPED CSS**, NOT the global `.login-brand-icon`

**Currently ORPHANED (defined in CSS, no DOM element):**
- Aurora system: `.login-aurora`, `.login-aurora-1`, `.login-aurora-2`, `.login-aurora-3`, `.login-aurora-4`
- Watermark: `.login-watermark`
- Starfield: `.login-stars`, `.login-star`, `.lg`
- Beams: `.login-beams`, `.login-beam-wrap`, `.login-beam`, `.thin`, `.thick`
- Halos: `.login-brand-icon-wrap`, `.login-halo-outer`, `.login-halo-inner`, `.login-brand-icon`

Total orphaned: 17 class names (19 - `.login-shell` - `.login-star` counted in both since `.login-star` has no element yet)

## 4. Template Reconstruction Plan

### 4.1 Target Structure (z-index layered)

```
<div class="login-page">
  <!-- LAYER 0: Canvas (z-index: 0) -->
  <OrbitField motion="bold" />

  <!-- LAYER 0: CSS effects (z-index: 0, renders above canvas via DOM order) -->
  <div class="login-aurora login-aurora-1"></div>
  <div class="login-aurora login-aurora-2"></div>
  <div class="login-aurora login-aurora-3"></div>
  <div class="login-aurora login-aurora-4"></div>

  <div class="login-watermark">
    <svg viewBox="0 0 560 560" fill="none">
      <path d="M280 20L540 140v200C540 380 430 460 280 500 130 460 20 380 20 340V140L280 20Z" stroke="rgba(139,92,246,.05)" stroke-width="2"/>
      <path d="M200 280l60 60 120-120" stroke="rgba(139,92,246,.05)" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/>
    </svg>
  </div>

  <div class="login-stars" v-if="!prefersReducedMotion">
    <div v-for="(star, i) in stars" :key="i"
      :class="['login-star', { lg: star.large }]"
      :style="{ left: star.left + '%', top: star.top + '%', animation: 'avTwinkle ' + star.duration + 's ease-in-out ' + star.delay + 's infinite' }">
    </div>
  </div>

  <!-- LAYER 1: Card (z-index: 1) -->
  <div class="login-shell">
    <motion.div ...>
      <div class="brand">
        <div class="login-brand-icon-wrap">
          <div class="login-halo-outer" v-if="!prefersReducedMotion"></div>
          <div class="login-halo-inner" v-if="!prefersReducedMotion"></div>
          <div class="login-brand-icon">
            <svg><!-- shield checkmark --></svg>
          </div>
        </div>
        <span class="brand-name">ArcVault</span>
      </div>
    </motion.div>
    <!-- ... rest of card/form/footer unchanged ... -->
  </div>
</div>
```

### 4.2 Script Additions Required

```js
// Starfield data (25 stars, deterministic positions)
const stars = [
  { left: 12, top: 8, delay: 1.4, duration: 3.2, large: false },
  // ... define all 25
]

// Beam data (12 beams converging on logo)
const beams = [
  { angle: 0, width: '' },
  { angle: 30, width: 'thin' },
  // ... define all 12
]
const logoCenter = { left: '50%', top: '26%' }
```

### 4.3 Per-Animation Element Details

| Animation | Elements | Inline Styles | v-if reduced-motion? |
|-----------|----------|---------------|----------------------|
| Aurora | 4 divs | None | Yes (entire divs) |
| Watermark | 1 div + 1 SVG | None needed (CSS handles positioning) | Yes (container) |
| Stars | 1 container + 25 divs | left%, top%, animation (delay+duration) | Yes (container) |
| Beams | 1 container + 12 wraps + 12 beams | left%, top%, rotate(Xdeg) on wraps | Yes (container) |
| Halos | 3 divs (wrap + outer + inner + brand-icon) | None | Yes (outer + inner only; icon pulse stays) |

## 5. Conflict & Integration Analysis

### 5.1 Z-Index Compatibility
- OrbitField canvas: `position: fixed; z-index: 0; pointer-events: none` → OK
- CSS effects: Need `position: absolute/fixed` with `z-index: 0` to be between canvas and card
- **Decision**: Use `position: absolute; inset: 0; pointer-events: none` on containers, no explicit z-index needed since DOM order places them after OrbitField (which is fixed) and before `.login-shell` (which has z-index: 1)
- **Result**: NO CONFLICT. DOM order naturally layers: fixed canvas → absolute effects (just above) → relative shell with z-index: 1

### 5.2 Scoped CSS vs Global CSS Conflict (Brand Icon)
- **Current**: Login.vue scoped `.brand-icon` (line 235-245) — 36x36px, border-radius 8px, accent-dim bg
- **Global**: `login-animation.css` `.login-brand-icon` (line 131-139) — 44x44px, border-radius 10px, rgba(139,92,246,.18) bg
- **Problem**: Scoped `.brand-icon` will NOT match the global `.login-brand-icon` class. They're different selectors.
- **Resolution**: Replace the scoped `.brand-icon` div with the global `.login-brand-icon-wrap > .login-halo-outer + .login-halo-inner + .login-brand-icon` structure. Keep the layout values (36px sizing) from scoped CSS by adjusting the global class or overriding in scoped.
- **Recommendation**: Use `.login-brand-icon` (global, provides animation) and keep the 36x36 sizing via scoped CSS targeting `:deep()` or by adjusting the global class size. The slight size difference (36px vs 44px) means the halo rings (inset: -18px, -10px) need recalculation for 36px icon.
- **Conflict risk**: LOW. Only the `.brand-icon` → `.login-brand-icon-wrap` refactor touches shared properties.

### 5.3 motion-v Spring Animation Compatibility
- motion-v animates opacity and translateY on brand, card, and form elements (staggered entrance)
- CSS `.login-shell` provides container entrance (avShellIn)
- **No property conflicts**: CSS animates the container, motion-v animates children
- **Timing note**: motion-v springs run once on mount; CSS loops are infinite — no conflict
- **Result**: NO CONFLICT

### 5.4 Reduced-Motion Strategy
- Login.vue already reads `prefersReducedMotion` at line 162 for warp behavior
- **Decision**: Use `v-if="!prefersReducedMotion"` on decorative CSS animation elements (aurora, watermark, stars, beams, halo rings)
- Icon pulse (`avIconPulse` on `.login-brand-icon`) is subtle enough to keep on reduced-motion
- `avShellIn` can stay — it's a one-time entrance, not decorative looping
- OrbitField already respects reduced-motion (static frame at line 704)
- **Strategy**: SCRIPT-CONTROLLED (v-if), not CSS media query, for consistency with existing code pattern

### 5.5 Performance Risk Assessment

| Element | Count | Animated Property | GPU-Composited? | Cost |
|---------|-------|-------------------|-----------------|------|
| Aurora blobs | 4 | transform, opacity | Yes (will-change: transform) | Medium (blur filter) |
| Watermark | 1 | transform, opacity | Yes | Low |
| Stars | 25 | opacity | Yes | Low |
| Beams | 12 | transform (translateX) | Yes | Low |
| Halos | 2 | transform (rotate), box-shadow | Yes | Low (box-shadow moderate) |
| OrbitField | 1 | Canvas render loop | Yes | High (rAF + 50+ draw calls) |
| **Total** | **~45** | | | **Medium-High** |

**Recommendations:**
1. `will-change: transform` already on `.login-aurora` — good
2. Aurora blur filters (60px, 45px, 50px) are expensive — consider reducing on low-tier via media query
3. Star count 25 is reasonable
4. Beam count 12 is reasonable
5. If performance issues arise, add CSS media query to disable aurora blurs:
   ```css
   @media (prefers-reduced-motion: reduce) {
     .login-aurora { filter: blur(20px); } /* reduce blur cost */
   }
   ```
6. **Risk**: MEDIUM — recommend verifying with CPU throttling in TASK 7

## 6. Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Reduced-motion control | `v-if` in template | Consistent with existing `prefersReducedMotion` pattern |
| Brand icon halo approach | Keep scoped layout + global animation | Avoid CSS specificity conflicts |
| Star generation | Fixed array in `<script setup>` with `v-for` | Deterministic, easy to tune |
| Beam convergence | Place logo center at `left: 50%; top: 26%` | Needs browser verification, easy to adjust |
| Z-index strategy | Rely on DOM order (no explicit z-index on effects) | Cleaner, avoids stacking context complexity |
| CSS effect containers | `position: absolute; inset: 0; pointer-events: none` | Match OrbitField pattern, safe layering |
