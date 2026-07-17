# Design — ArcVault Dashboard

A locked design system for this app ("Kiln"). Every page/component change reads
this file before emitting styles. Do not regenerate per page — extend or amend
this file when the system needs to grow. Consistency across views is the goal,
not variety.

## Genre
modern-minimal (ops dashboard — function carries the page, no enrichment)

## Identity
Vault materials: warm charcoal + copper + verdigris in dark mode; parchment +
burnt copper in light mode. Machined corners, hairline borders, flat shadows.
**Banned:** glows, neon, gradient accents, ambient radial blobs, pill-shaped
status badges, italic headings.

## Theme (dark, default)
- `--bg-base`      oklch(0.175 0.008 60)
- `--bg-card`      oklch(0.215 0.010 60)
- `--text-primary` oklch(0.93 0.010 80)
- `--accent`       oklch(0.75 0.13 65)   ← copper
- `--accent-2`     oklch(0.72 0.09 165)  ← verdigris
- Semantic: success oklch(0.74 0.13 155) · error oklch(0.68 0.17 25) ·
  warning oklch(0.80 0.13 85) · info oklch(0.74 0.09 230) · running = copper

## Theme (light)
Parchment paper oklch(0.965 0.006 85), umber ink oklch(0.25 0.015 60),
burnt-copper accent oklch(0.54 0.12 55). Full block in `src/style.css`.

## Typography
- Display: Space Grotesk 700, roman only, tight tracking — headings and stat values
- Body: Inter 400/500/600
- Structural microtype: JetBrains Mono 500 — stat labels, table headers, badges,
  version tag, WS indicator. Uppercase + 0.1em tracking is the mono voice's job.

## Spacing & shape
- rem-based scale as established in `src/style.css`; new styles reuse tokens.
- `--radius-card: 6px` · `--radius-ctrl: 4px` · badges/tags 3px. Pills (999px)
  are reserved for filter chips only.

## Motion
- CSS transitions 0.12–0.2s, `transform`/`opacity`/color only.
- Router view fade-slide 80ms. `prefers-reduced-motion` collapse is global.
- Status dots may pulse (badge-pulse, ws-pulse). Nothing else loops.

## Interaction
- Focus: `--glow-accent` is a crisp 3px `--accent-dim` ring (the token name is
  historical — it must never become a glow again). Buttons get
  `outline: 2px solid var(--accent); outline-offset: 2px` on `:focus-visible`.
- Hover: 1px translateY on buttons, border-color step-up on cards. No glow, no scale.
- Semantic borders derive via `color-mix(in oklab, var(--color-*) N%, transparent)`,
  never hardcoded rgba.

## What views MUST share
- All colors and fonts by token reference — no inline hex/oklch in components.
- The badge voice (square mono tag + status dot), table header voice (mono
  uppercase), button voice (copper fill primary, outlined secondary/ghost).

## Per-page allowances
- Login screen: "vault dial" composition (Login.vue + login-animation.css) —
  counter-rotating machined tick rings (copper outer/sweep, verdigris inner)
  over ember glow, etched copper shield watermark. Always dark; its warm-dark
  values are intentionally hardcoded (#e0a05c copper, #1c1610 charcoal) since
  the page renders before/independent of theme tokens. Slow rotation only
  (45–90s), no glow filters, reduced-motion removes the dial entirely.
- Charts: solid copper marks, verdigris only as the second series.
