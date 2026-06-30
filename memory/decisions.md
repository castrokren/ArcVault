# Design Decisions — Login Animation Restoration

## Decision: CSS Animations Layer at z-index: 0 Alongside OrbitField Canvas

**Date:** 2026-06-29
**Context:** Login.vue needed decorative CSS animations (aurora, watermark, stars, beams, halos) that would render behind the login card but on top of the OrbitField canvas background.
**Decision:** Use DOM order for z-index layering (no explicit z-index on decorative containers). OrbitField canvas at z-index: 0, decorative CSS elements after OrbitField in DOM (same stacking context), login card at z-index: 1.
**Rationale:**
- Avoids z-index stacking context nesting issues
- CSS animation containers don't need z-index values
- DOM order is sufficient: canvas < decorative divs < card
- All decorative divs use position: absolute, OrbitField uses position: fixed
- The .login-page wrapper (position: relative) does not create a new stacking context (no z-index set), so all children share the root stacking context
**Trade-offs:**
- Less explicit than z-index values
- Relies on DOM order which could be accidentally changed
- Verified correct during TASK-07 QA

## Decision: script-controlled v-if for Reduced Motion (not CSS media query)

**Date:** 2026-06-29
**Context:** Need to hide decorative animations when user prefers reduced motion.
**Decision:** Use `const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches` evaluated once at setup, with `v-if="!prefersReducedMotion"` on each decorative element.
**Rationale:**
- CSS-only approach (`@media (prefers-reduced-motion: reduce)` turning off animations) would still leave DOM elements present
- v-if completely removes decorative elements from the DOM, reducing render load
- The icon pulse animation is intentionally kept on reduced-motion (not decorative, it's branding)
**Trade-offs:**
- Static value: doesn't react to live OS changes (unlikely during a login session)
- Requires test mocking of window.matchMedia

## Decision: Fixed Array for Star Data (not random generation)

**Date:** 2026-06-29
**Context:** 25 stars needed with unique positions and animation delays for the starfield effect.
**Decision:** Pre-compute a fixed array of 25 star objects with explicit left%, top%, delay, duration, and large flags.
**Rationale:**
- Deterministic: same stars every render (no SSR hydration mismatch)
- Testable: can assert positions and count
- Easy to tune: adjust individual star values
**Trade-offs:**
- Not visually random (all users see same star pattern)

## Decision: Beam Convergence at left:50%, top:26%

**Date:** 2026-06-29
**Context:** 12 data-comet beams need to converge on the brand icon center.
**Decision:** Estimated brand icon position at left:50%, top:26% of the viewport. Beams use transform-origin: 0 0 on .login-beam-wrap (positioned at logoCenter) with rotate(Xdeg) rotation.
**Rationale:**
- Percentage-based positioning responsive to viewport
- transform-origin: 0 0 ensures beams pivot from the logo center
- 12 beams at 30-degree intervals (0, 30, 60, ..., 330) provide full 360° coverage
**Open question:** Exact percentage needs browser verification — adjust logoCenter object if beams misalign.

## Decision: Parallax Using Inline Style (not CSS animation)

**Date:** 2026-06-29
**Context:** Login card needs subtle parallax shift on pointer movement.
**Decision:** Use `shellStyle` computed property returning inline `transform: translate3d(...)` based on parallax refs updated by pointermove and deviceorientation events.
**Rationale:**
- Inline style avoids CSS animation interference
- Reactive via computed property
- Fallback: reduced-motion returns empty object (no transform)
**Bug discovered & fixed:** CSS animation `avShellIn` used `animation-fill-mode: both` which persisted the final keyframe state and overrode the inline transform. Fixed by removing `both` from the animation shorthand.

## Decision: Gradient Card Border Using mask-composite

**Date:** 2026-06-29
**Context:** Login card needs a subtle gradient glow border effect.
**Decision:** Use ::before pseudo-element with border-radius inset, linear-gradient background, and mask-composite: exclude to punch out the center.
**Rationale:**
- No extra DOM elements needed
- Gradient border effect without border-image (which doesn't support border-radius)
**Cross-browser note:** Requires unprefixed `mask` property for Firefox (bug fixed during TASK-07 QA); webkit browsers use `-webkit-mask`.
