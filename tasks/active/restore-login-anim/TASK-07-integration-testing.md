# TASK 7: Integration & Manual Testing

**Estimate:** 45 min  
**Owner:** @aisha  
**Status:** Not Started  
**Dependencies:** TASK-02, TASK-03, TASK-04, TASK-05, TASK-06 (all complete)  
**Blocks:** TASK-08-automated-tests

## Description
Full visual QA, reduced-motion testing, performance profiling, and cross-browser verification of all restored login page animations.

## Sub-Tasks

### 7.1 Visual QA (15 min)
- All 6 animation systems rendering simultaneously
- Z-index layering correct: canvas → aurora → watermark → stars → beams → halos → shell → card
- No visual glitches, jank, or overlap conflicts
- Animation timing feels natural (slow aurora drift, fast beams)

### 7.2 Reduced-Motion Testing (10 min)
- Enable `prefers-reduced-motion: reduce`
- Verify: aurora, watermark, stars, beams, halos all hidden
- Verify: OrbitField shows static frame (already handled)
- Verify: motion-v spring animations disabled (already handled)
- Verify: icon pulse still active (subtle effect, should stay)

### 7.3 Performance Testing (10 min)
- Chrome DevTools Performance tab, record 10s
- Target: 60fps sustained on high-tier
- CPU 4x throttling → no freeze, UI responsive
- Memory: no leaks on navigation away/back

### 7.4 Cross-Browser Testing (10 min)
- Chrome: primary target
- Firefox: check backdrop-filter, blur, animation support
- Edge: Chromium-based, should match Chrome

## Acceptance Criteria
- [ ] All effects render correctly
- [ ] Reduced-motion works
- [ ] 60fps on high tier
- [ ] No memory leaks
- [ ] Works in Chrome, Firefox, Edge
