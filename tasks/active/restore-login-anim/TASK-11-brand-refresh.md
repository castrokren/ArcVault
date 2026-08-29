# TASK 11: Brand Refresh — Icon Gradient & Styled Text

**Estimate:** 25 min  
**Owner:** @sofia  
**Status:** Not Started  
**Dependencies:** TASK-10 (complete)  
**Blocks:** TASK-12

## Description
Upgrade the brand section to match the standalone HTML reference: gradient-filled shield icon, colored "Vault" text, and "Coordinator" subtitle.

## Implementation

### 11.1 SVG Icon with Gradient (15 min)
Replace the current shield checkmark SVG with the standalone HTML version that has gradient fill:

```vue
<svg class="mark" viewBox="0 0 48 48" fill="none">
  <defs>
    <linearGradient id="brandGrad" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#9d72ff"/>
      <stop offset="1" stop-color="#5f33d6"/>
    </linearGradient>
  </defs>
  <path d="M24 5 L41 14 V34 L24 43 L7 34 V14 Z" stroke="url(#brandGrad)" stroke-width="2" opacity=".55"/>
  <path d="M14 30 A12 12 0 0 1 34 30" stroke="url(#brandGrad)" stroke-width="2.6" stroke-linecap="round"/>
  <path d="M18 30 A6.5 6.5 0 0 1 30 30" stroke="url(#brandGrad)" stroke-width="2.6" stroke-linecap="round" opacity=".7"/>
  <circle cx="24" cy="30" r="2.4" fill="#9d72ff"/>
</svg>
```

Add scoped CSS for `.mark`:
```css
.mark {
  width: 42px;
  height: 42px;
  flex: none;
  filter: drop-shadow(0 0 16px var(--accent-dim));
}
```

### 11.2 Colored Brand Text (5 min)
Update `.brand-name` to split "Arc" and "Vault":

```vue
<span class="brand-name">Arc<b>Vault</b></span>
```

Scoped CSS:
```css
.brand-name b {
  color: var(--accent);
}
```

### 11.3 Brand Subtitle (5 min)
Add subtitle below brand name:

```vue
<div class="brand-text">
  <span class="brand-name">Arc<b>Vault</b></span>
  <span class="brand-sub">Backup orchestrator</span>
</div>
```

Scoped CSS:
```css
.brand-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.brand-sub {
  display: block;
  font-size: 10px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-weight: 500;
}
```

## Acceptance Criteria
- [ ] Shield icon has gradient fill (purple → indigo)
- [ ] Icon has drop-shadow glow
- [ ] "Vault" text is colored accent
- [ ] Subtitle "Backup orchestrator" displays below brand name
