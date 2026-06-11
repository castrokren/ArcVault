# Obsidian Pro Frontend Restyle — Implementation Plan

**Spec:** docs/superpowers/specs/2026-06-11-obsidian-pro-frontend-restyle-design.md

## Goal
Apply the approved Obsidian Pro visual system to the entire Vue dashboard — tokens, components, flair — with zero behavior or backend changes.

## Tasks
- [ ] 1. Self-host fonts: `npm i @fontsource/space-grotesk @fontsource/inter @fontsource/jetbrains-mono`, import weights in `main.js`, remove Google Fonts links from `index.html` → Verify: `npm run build` passes; `grep -r fonts.googleapis dist/` empty
- [ ] 2. Rewrite `src/style.css`: Obsidian Pro tokens (dark + re-derived light), depth/glow tokens, ambient background tints, shared classes (cards, tables, badges, buttons, forms, modals, skeleton shimmer), `prefers-reduced-motion` guards → Verify: dev server renders all views legibly in both themes
- [ ] 3. Restyle `App.vue`: translucent nav, sliding active-route underline, mono version badge, `<Transition>` fade-slide on router-view → Verify: nav underline follows route; transition ~80ms
- [ ] 4. Create `components/Sparkline.vue` (inline SVG, props: `points`, `color`, no deps); wire into stat cards on views that already hold run history data → Verify: sparkline renders from existing fetched data; no new network calls in devtools
- [ ] 5. Rebuild `Login.vue` scene: SVG orbit rings + counter-rotating accent arcs + travel dots, card entrance fade → Verify: login renders, animation freezes under reduced-motion, form submit unchanged
- [ ] 6. Sweep 11 views (Agents, Jobs, History, Alerts, Templates, Groups, Users, Federation, FederationHealth, Login leftovers, admin/Credentials): adopt shared classes, remove local style overrides, swap loading text for skeletons, mono columns for IDs/cron/paths → Verify: each view diff touches only `<style>` blocks + class attrs
- [ ] 7. Sweep 9 components (Pagination, ScheduleBuilder, SyncFlagsBuilder, JobTimeline, AgentRunChart, UpdateBanner, UpdateModal, AgentUpdateModal, ChangePasswordModal): badges with pulsing dots, modal glow/enter transitions, focus glow on inputs → Verify: modals open/close, builders still emit same values

## Done When
- [ ] `npm run build` passes
- [ ] `SyncFlagsBuilder.test.js` + `Jobs.integration.test.js` pass
- [ ] `go test ./...` baseline holds (110 tests, 108 pass + 2 skip on Windows)
- [ ] Every view visually verified in dark and light themes
- [ ] No changes to `api.ts`, `schemas/`, `types/`, `composables/`, `router/`, or any Go file

## Notes
- Deploy is Kren-run via `scripts/rebuild-and-restart.ps1`; this plan stops at verified `dist/`-buildable code
- Git commits via PowerShell on host (sandbox git avoided per workflow memory)
- `views/Jobs.vue.main` is a stray backup file — leave untouched
