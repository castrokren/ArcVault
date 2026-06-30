# TASK 12: Parallax & Warp Polish

**Estimate:** 35 min  
**Owner:** @sofia  
**Status:** Not Started  
**Dependencies:** TASK-11 (complete)  
**Blocks:** TASK-07 (manual testing), TASK-08 (automated tests)

## Description
Add pointer-based parallax card shift and warp animation (card shrinks + fades) to match the standalone HTML reference.

## Implementation

### 12.1 Parallax Card Shift (20 min)

**Script changes:**
Add pointer tracking after `prefersReducedMotion`:

```js
// Parallax pointer tracking
const parallax = { x: 0, y: 0 }

function updateParallax(clientX, clientY) {
  parallax.x = (clientX / window.innerWidth - 0.5) * 2
  parallax.y = (clientY / window.innerHeight - 0.5) * 2
}

if (!prefersReducedMotion) {
  window.addEventListener('pointermove', (e) => updateParallax(e.clientX, e.clientY))
  window.addEventListener('deviceorientation', (e) => {
    if (e.gamma == null) return
    parallax.x = Math.max(-1, Math.min(1, e.gamma / 28))
    parallax.y = Math.max(-1, Math.min(1, (e.beta - 45) / 28))
  })
}
```

**Template changes:**
Add computed transform to `.login-shell`:

```vue
<div class="login-shell" :style="shellStyle">
```

```js
const shellStyle = computed(() => {
  if (prefersReducedMotion) return {}
  return {
    transform: `translate3d(${parallax.x * -15}px, ${parallax.y * -15}px, 0)`
  }
})
```

**Scoped CSS:**
```css
.login-shell {
  transition: transform 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
}
```

### 12.2 Warp Card Animation (15 min)

**Script changes:**
Add warp state tracking:

```js
const isWarping = ref(false)

async function handleSubmit() {
  // ... existing code up to success path ...
  
  // Success path — warp (or fade) then redirect
  isWarping.value = true
  if (prefersReducedMotion) {
    await new Promise(r => setTimeout(r, 200))
  } else {
    await orbitField.value?.warp()
  }
  isWarping.value = false
  router.push('/')
  isSubmitting.value = false
}
```

**Template changes:**
Add warp class binding to `.login-shell`:

```vue
<div class="login-shell" :class="{ warping: isWarping }" :style="shellStyle">
```

**Scoped CSS:**
```css
.login-shell.warping {
  transform: scale(0.8) translateY(-8px) !important;
  opacity: 0;
  transition: transform 1.05s cubic-bezier(0.6, 0, 0.25, 1), opacity 0.9s ease;
}
```

## Acceptance Criteria
- [ ] Card shifts on pointer move (parallax)
- [ ] Card shifts on device orientation (mobile)
- [ ] Parallax disabled on reduced-motion
- [ ] Card shrinks + fades during warp
- [ ] Warp animation smooth (1.05s cubic-bezier)
- [ ] Card returns to normal after warp completes
