# TASK 5: Data-Comet Beams Implementation

**Estimate:** 40 min  
**Owner:** @sofia  
**Status:** Not Started  
**Dependencies:** TASK-01-analysis (complete)  
**Blocks:** TASK-07-integration-testing

## Description
Add 12 data-comet beams converging on the ArcVault logo from all angles. Each beam is a gradient line that rushes inward on a 3s loop.

## Implementation

### 5.1 Add Beams Array to Script
```js
const beams = [
  { angle: 0, width: '' },
  { angle: 30, width: 'thin' },
  { angle: 60, width: '' },
  { angle: 90, width: 'thick' },
  { angle: 120, width: '' },
  { angle: 150, width: 'thin' },
  { angle: 180, width: '' },
  { angle: 210, width: 'thick' },
  { angle: 240, width: '' },
  { angle: 270, width: 'thin' },
  { angle: 300, width: '' },
  { angle: 330, width: 'thick' },
]

// ponytail: exact logo center needs browser verification — adjust these if beams misalign
const logoCenter = { left: '50%', top: '26%' }
```

### 5.2 Add Beams Template
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
    <div :class="['login-beam', beam.width]"></div>
  </div>
</div>
```

**File:** `dashboard/src/views/Login.vue` (template after stars, before login-shell)

### 5.3 Verify Rendering
- 12 beams visible
- Converge accurately on brand icon
- Range of widths (thin, normal, thick)
- 3s rush animation smooth

## Acceptance Criteria
- [ ] 12 beam pairs render from v-for
- [ ] Hidden when prefersReducedMotion
- [ ] Beams converge on brand icon center
- [ ] Rotation angles cover full 360° (30° increments)
- [ ] Animation timing correct (3s loop)
- [ ] `ponytail:` comment marks logoCenter for easy adjustment
