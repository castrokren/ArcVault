# TASK 1: Analysis & Design

**Estimate:** 45 min  
**Owner:** @maya (Analysis) → @david (Architecture)  
**Status:** Not Started  
**Dependencies:** None  
**Blocks:** TASK 2-6 (all implementation)

## Description
Audit login-animation.css, cross-reference with Login.vue, plan the template structure for re-adding missing DOM elements.

## Sub-Tasks

### 1.1 Audit CSS Classes & Keyframes (20 min)
- Read login-animation.css fully (180 lines)
- Document ALL class selectors (20 total)
- Document ALL @keyframes (8 total)
- Map each class → which keyframes it uses
- Map each keyframe → which classes reference it
- Identify which classes are currently live (`.login-shell` only)
- Identify which are orphaned (19 classes)

### 1.2 Template Reconstruction Plan (15 min)
For each orphaned class group, determine:
- What HTML elements are needed
- What inline styles are required
- Where in the template they should go (z-index ordering)
- How they interact with existing elements

**Aurora:** 4 divs, no children, no inline styles
**Watermark:** 1 container div + 1 SVG child (need shield path)
**Stars:** 1 container div + 25 star children, each with: left%, top%, animation-delay
**Beams:** 1 container div + 12 beam-wrap divs + 12 beam children, each with: left%, top%, rotate(Xdeg)
**Halos:** Replace `.brand-icon` with `.login-brand-icon-wrap` + `.login-halo-outer` + `.login-halo-inner` + `.login-brand-icon`

### 1.3 Conflict & Integration Analysis (10 min)
- Check z-index compatibility with OrbitField (z-index:0)
- Check scoped CSS vs global CSS conflicts for brand-icon
- Check reduced-motion handling strategy
- Check performance risk (40 animated elements + canvas)

## Deliverable
`tasks/active/restore-login-anim/analysis-results.md` — Full audit with template structure

## Acceptance Criteria
- [ ] All 20 classes and 8 keyframes documented
- [ ] Live vs dead classes clearly identified
- [ ] Template structure defined with z-index ordering
- [ ] Reduced-motion strategy decided
- [ ] Performance risk assessed and mitigated
