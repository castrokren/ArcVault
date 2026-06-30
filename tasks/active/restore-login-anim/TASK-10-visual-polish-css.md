# TASK 10: Visual Polish — CSS Improvements

**Estimate:** 30 min  
**Owner:** @sofia  
**Status:** Not Started  
**Dependencies:** TASK-06 (complete)  
**Blocks:** TASK-11

## Description
Improve the visual quality of the login page to match the standalone HTML reference design. Focus on CSS-only improvements: background gradient, vignette overlay, card styling, input focus effects, and button polish.

## Implementation

### 10.1 Background Gradient (5 min)
Replace flat `var(--bg-base)` with deep radial gradient on `.login-page`:

```css
background: radial-gradient(120% 90% at 50% 8%, #0a1322 0%, #05070d 55%, #03040a 100%);
```

### 10.2 Vignette Overlay (5 min)
Add a vignette overlay div in template after OrbitField, before aurora:

```vue
<div class="login-vignette"></div>
```

Scoped CSS:
```css
.login-vignette {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  background: radial-gradient(circle at 50% 42%, transparent 36%, rgba(3,4,10,.82) 100%);
}
```

### 10.3 Card Border Glow (10 min)
Update `.login-card` scoped styles to add gradient border via `::before` pseudo-element:

```css
.login-card::before {
  content: "";
  position: absolute;
  inset: 0;
  border-radius: var(--radius-card);
  padding: 1px;
  background: linear-gradient(155deg, rgba(125,77,255,.55), rgba(139,92,246,.25) 35%, transparent 65%);
  -webkit-mask: linear-gradient(#000 0 0) content-box, linear-gradient(#000 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  pointer-events: none;
}
```

Also strengthen card shadow:
```css
box-shadow: 
  var(--shadow-lg), 
  var(--edge-highlight), 
  0 0 60px var(--accent-dim),
  0 50px 130px -30px rgba(0,0,0,.9);
```

### 10.4 Input Focus Underline (5 min)
Add animated underline effect to `.input-wrap` or `.field`:

```css
.field::after {
  content: "";
  position: absolute;
  left: 15px;
  right: 15px;
  bottom: 1px;
  height: 2px;
  border-radius: 2px;
  background: linear-gradient(90deg, var(--accent), transparent);
  transform: scaleX(0);
  transform-origin: left;
  transition: transform 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
  opacity: 0;
}

.field:focus-within::after {
  transform: scaleX(1);
  opacity: 0.9;
}
```

### 10.5 Button Shine Effect (5 min)
Add shine sweep effect to `.submit-btn`:

```css
.submit-btn::after {
  content: "";
  position: absolute;
  top: 0;
  left: -60%;
  width: 40%;
  height: 100%;
  background: linear-gradient(100deg, transparent, rgba(255,255,255,.55), transparent);
  transform: skewX(-18deg);
  transition: left 0.6s ease;
}

.submit-btn:hover::after {
  left: 130%;
}
```

## Acceptance Criteria
- [ ] Background has depth (radial gradient)
- [ ] Vignette darkens edges
- [ ] Card border glows (gradient edge)
- [ ] Card shadow stronger/more dramatic
- [ ] Input focus shows animated underline
- [ ] Button has shine sweep on hover
