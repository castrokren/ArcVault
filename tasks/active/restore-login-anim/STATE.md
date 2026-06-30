# STATE — Restore Login Page CSS Animation Layers

## Goal
Restore the 5 missing CSS animation effects (aurora, watermark, stars, beams, halos) to Login.vue by re-adding the DOM elements that use the classes defined in login-animation.css.

## Invariants / decisions
- CSS animations layer at z-index: 0 alongside OrbitField canvas
- login card stays at z-index: 1 (on top)
- reduced-motion hides all decorative (!) animations via v-if
- .login-shell entrance animation stays (already working, keep as-is)
- 6 animation systems total: aurora, watermark, stars, beams, halos, shell
- Individual task files are self-contained — each can be worked in separate sessions
- Reduced-motion strategy: script-controlled v-if (not CSS media query)
- Star generation: fixed array in script with v-for
- Beam convergence: estimated at left:50%, top:26% (needs browser verification)
- Z-index: rely on DOM order, no explicit z-index on CSS effect containers
- Brand icon: keep scoped layout + global animation, merge carefully

## Done
- TASK-01-analysis — Audit CSS, plan template structure, assess conflicts
- TASK-02-aurora — Add 4 aurora blob divs after OrbitField, before login-shell
- TASK-03-watermark — Add watermark container + shield SVG
- TASK-04-starfield — Add stars container + 25 star divs with inline styles
- TASK-05-beams — Add beams container + 12 beam pairs with rotation angles
- TASK-06-halos — Refactor brand icon structure to include halo rings
- TASK-10-visual-polish-css — Background gradient, vignette, card glow, input focus, button shine
- TASK-11-brand-refresh — Icon gradient, colored text, subtitle
- TASK-12-parallax-warp — Pointer parallax, warp animation
- TASK-07-integration-testing — Visual QA, reduced-motion, build verification, cross-browser CSS audit; 3 bugs found & fixed
- TASK-08-automated-tests — 26 Vitest tests covering all 6 animation systems + reduced-motion; all 89 tests pass
- TASK-09-docs-handoff — Code review, CONTEXT.md updated, decisions.md created, completion summary archived

## In-progress
- (none — epic complete)

## Next
- (none — epic complete)

## Open questions
- Beam convergence point: exact percentage for logo center (need browser inspection)
- Star count: 25 seems right, adjust if visually cluttered
- Nebula vs Aurora: Should we replace aurora blobs (4 small) with nebula blobs (2 large) from standalone HTML?

## File map
- STATE.md — This file (epic tracker)
- analysis-results.md — TASK 1 deliverable (full audit & plan)
- TASK-01-analysis.md — Audit & design (original spec)
- TASK-02-aurora.md — Aurora blobs
- TASK-03-watermark.md — Watermark SVG
- TASK-04-starfield.md — Starfield
- TASK-05-beams.md — Data-comet beams
- TASK-06-halos.md — Brand icon halos
- TASK-10-visual-polish-css.md — Visual polish (CSS improvements)
- TASK-11-brand-refresh.md — Brand icon gradient & styled text
- TASK-12-parallax-warp.md — Parallax & warp animations
- TASK-07-integration-testing.md — Visual QA, performance, cross-browser
- TASK-08-automated-tests.md — Vitest tests
- TASK-09-docs-handoff.md — Review, docs, completion
