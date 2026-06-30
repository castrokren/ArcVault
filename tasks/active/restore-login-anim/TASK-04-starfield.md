# TASK 4: Starfield Implementation

**Estimate:** 35 min  
**Owner:** @sofia  
**Status:** Not Started  
**Dependencies:** TASK-01-analysis (complete)  
**Blocks:** TASK-07-integration-testing

## Description
Add a twinkling starfield to Login.vue. 25 stars spread across the viewport with staggered twinkle animations.

## Implementation

### 4.1 Add Stars Array to Script
Add to `<script setup>`:
```js
const stars = [
  { left: 5, top: 8, delay: 0.1, duration: 3.2, large: false },
  { left: 15, top: 20, delay: 0.8, duration: 2.8, large: true },
  { left: 25, top: 5, delay: 1.5, duration: 4.0, large: false },
  { left: 35, top: 15, delay: 2.2, duration: 2.5, large: false },
  { left: 45, top: 30, delay: 0.3, duration: 3.5, large: true },
  { left: 55, top: 10, delay: 1.9, duration: 3.0, large: false },
  { left: 65, top: 25, delay: 0.6, duration: 2.7, large: false },
  { left: 75, top: 3, delay: 2.8, duration: 3.8, large: true },
  { left: 85, top: 18, delay: 1.2, duration: 2.9, large: false },
  { left: 92, top: 35, delay: 0.9, duration: 3.3, large: false },
  { left: 10, top: 45, delay: 1.7, duration: 2.6, large: false },
  { left: 20, top: 60, delay: 0.4, duration: 3.7, large: true },
  { left: 30, top: 50, delay: 2.5, duration: 2.8, large: false },
  { left: 40, top: 70, delay: 0.7, duration: 4.1, large: false },
  { left: 50, top: 55, delay: 1.1, duration: 3.1, large: true },
  { left: 60, top: 75, delay: 2.0, duration: 2.4, large: false },
  { left: 70, top: 65, delay: 0.2, duration: 3.6, large: false },
  { left: 80, top: 80, delay: 1.8, duration: 2.9, large: true },
  { left: 90, top: 58, delay: 2.3, duration: 3.4, large: false },
  { left: 3, top: 72, delay: 1.4, duration: 3.0, large: false },
  { left: 48, top: 42, delay: 0.5, duration: 2.5, large: false },
  { left: 72, top: 48, delay: 2.7, duration: 3.9, large: true },
  { left: 18, top: 35, delay: 1.6, duration: 2.7, large: false },
  { left: 82, top: 42, delay: 0.0, duration: 3.2, large: false },
  { left: 38, top: 85, delay: 2.1, duration: 2.8, large: true },
]
```

### 4.2 Add Stars Template
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

**File:** `dashboard/src/views/Login.vue` (template after watermark, before beams)

### 4.3 Verify Rendering
- 25 stars visible
- Staggered twinkle (not all in sync)
- Mix of 2px and 3px stars
- Color: `#d8ccff`

## Acceptance Criteria
- [ ] 25 star divs render from v-for
- [ ] Hidden when prefersReducedMotion
- [ ] Each star has unique position and delay
- [ ] Twinkle animations run with stagger
