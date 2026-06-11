# Obsidian Pro — Dashboard Visual Modernization (Design Spec)

**Date:** 2026-06-11
**Author:** Kren Castro + Claude
**Status:** Approved by Kren (sections 1–3 approved in brainstorming session)
**Scope:** Visual-only restyle of the Vue 3 dashboard. Zero backend changes. Zero behavior changes.

## Goal

Replace the Phase 18 visual system with a refined dark direction ("Obsidian Pro"):
deeper blacks, a teal + violet accent pair, larger display typography, floating
panels with depth/glow, micro-interactions, sparkline stat cards, and an animated
login scene. The light theme is re-derived from the same tokens so the existing
theme toggle keeps working.

## Decisions Made

| Decision | Choice |
|---|---|
| Driver | Visual style only (no layout/IA changes) |
| Direction | A — Obsidian Pro (vs. Ops Terminal, Slate Daylight) |
| Depth | Full visual pass: every view and component |
| Flair | Glow/depth, micro-interactions, animated login, stat sparklines |
| Typography | Space Grotesk (display) + Inter (body) + JetBrains Mono, **self-hosted woff2** (Google CDN removed — offline networks currently fall back to system fonts) |
| Approach | 1 — Token-first cascade (vs. base-component extraction, parallel theme) |

## Section 1 — Visual Language (Tokens)

All tokens remain CSS custom properties in `dashboard/src/style.css`, same
mechanism as today (`:root` / `[data-theme="light"]`).

### Surfaces (dark default)

| Token | Value |
|---|---|
| `--bg-base` | `#07070d` |
| `--bg-surface` | `#0d0d17` |
| `--bg-card` | `#13131f` |
| `--bg-elevated` | `#1a1a2a` |
| `--bg-input` | `#0b0b14` |

Borders: `#1c1c2c` (subtle), `#232338` (default), `#2e2e48` (strong).
Text: `#f0f2ff` (primary), `#aab2cc` (secondary), `#6b7394` (muted) — raised
contrast vs. current `#7b87a2`/`#404b62`.

### Accents & semantics

- `--accent: #00e5b8` (primary actions, brand, active states)
- `--accent-2: #8b86ff` (violet companion: secondary highlights, chart series 2, login orbit)
- Success `#4ade80`, error `#ff5c7a`, warning `#fbbf24`, info/running `#38bdf8`
- Each semantic keeps a `--bg-*` dim variant at ~12% alpha

### Depth & glow

- 3-stop shadow scale (sm/md/lg), darker and tighter than current
- Cards get a 1px top-edge highlight: `inset 0 1px 0 rgba(255,255,255,0.04)`
- `--glow-accent: 0 0 0 1px var(--accent-border), 0 0 24px var(--accent-dim)` for focused/active elements
- Page background: two fixed ambient radial tints (teal top-left, violet bottom-right, ~4% opacity) on `body::before` / `body::after` — `pointer-events: none`, `position: fixed`
- Radius: cards 8px → 10px, inputs/buttons 6px → 8px

### Typography

- `--font-display: 'Space Grotesk'` — page titles, stat numerals, nav brand
- `--font-body: 'Inter'` — body, tables, forms, labels
- `--font-mono: 'JetBrains Mono'` — tokens, cron, paths, logs, versions
- Six woff2 files bundled in `dashboard/src/assets/fonts/` (~400KB), declared
  via `@font-face` in `style.css` with `font-display: swap`. Google Fonts
  `<link>` tags removed from `index.html`.

### Light theme

Re-derived from the same token names: warm off-white base `#f5f6fa`, white
cards, accent darkened to `#00a887` / `#5d56d6` for contrast, same shadow/glow
structure at reduced strength. Toggle mechanism unchanged.

## Section 2 — Components & Flair

- **Nav:** slimmer translucent bar (`backdrop-filter: blur`), bottom hairline,
  accent underline that slides to the active route, version badge in mono.
- **Stat cards:** Space Grotesk numerals; inline sparkline of recent run history
  via new `components/Sparkline.vue` (~60-line inline-SVG component, no
  dependencies, fed from data the views already fetch — no new API calls).
- **Tables:** sticky headers, hover row wash, no zebra; mono font for IDs,
  cron expressions, and paths.
- **Badges:** pill style with status dot; `running`/`online` dots pulse via CSS
  keyframe.
- **Buttons:** solid-accent primary (dark text on teal), ghost secondary,
  danger variant; 150ms hover lift (`translateY(-1px)` + shadow step).
- **Modals:** elevated surface, glow border, scale+fade enter transition.
- **Forms:** consistent focus glow (`--glow-accent`), unified input heights.
- **Loading:** skeleton shimmer placeholders replace "Loading…" text in views.
- **View transitions:** Vue built-in `<Transition>` fade-slide, ~80ms.
- **Login scene:** concentric orbit rings (SVG) behind the centered card; two
  accent arcs + travel dots rotate slowly (60–90s, opposite directions); card
  entrance fade. Pure CSS/SVG — no canvas, no libraries.
- **Motion safety:** all animation (pulse, orbits, shimmer, transitions)
  disabled under `prefers-reduced-motion: reduce`.

## Section 3 — Scope & Safety

### Files touched

- `dashboard/index.html` — remove Google Fonts links
- `dashboard/src/style.css` — token + shared-class rewrite, @font-face
- `dashboard/src/assets/fonts/` — new, 6 woff2 files
- `dashboard/src/components/Sparkline.vue` — new
- `dashboard/src/App.vue`, all 11 views (incl. `admin/Credentials.vue`), all 9
  existing components — style blocks and class names only

### Explicitly untouched

`api.ts`, `schemas/`, `types/`, `composables/`, `router/`, all fetch/WebSocket
logic, props/emits/v-models, form field names, anything in Go, the API
contract, and the `dist/` → `coordinator/static/dist` → `go:embed` pipeline.

### Verification

1. `npm run build` passes
2. Existing dashboard tests pass (`SyncFlagsBuilder.test.js`, `Jobs.integration.test.js`)
3. `go test ./...` baseline holds (core rule 1)
4. Visual smoke of every view in both themes via `npm run dev`
5. Deployment remains manual via `scripts/rebuild-and-restart.ps1` (Kren-run)

### Second-order effects considered

Font self-hosting changes `index.html` asset references; Vite fingerprints
fonts into `dist/` automatically, so the Go embed picks them up like any other
static asset. No installer, service, or token implications.
