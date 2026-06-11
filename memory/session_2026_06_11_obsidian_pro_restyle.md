# Obsidian Pro Restyle — Implemented June 11, 2026 (Session 23)

- Full dashboard restyle shipped and deployed; visual-only (Go/api.ts/composables/router/schemas/types untouched)
- Fonts: @fontsource packages imported in main.js (NOT manual woff2 files); Google CDN gone
- Shared classes live in style.css: .card, .stat-card, .skeleton/.skeleton-line/.skeleton-block, pill .badge with pulsing dots, .table sticky headers, --glow-accent focus, modal-pop keyframe
- Sparkline.vue: props points/color/width/height/fill; feed from already-fetched data only
- Deploy script now TLS-aware (https://localhost, cert skip, /health retry loop) — required for every deploy
- LESSON: never use Edit/Write tools on this mount — bash heredoc/python only (null-byte corruption)
- DEBT: JS tests stale (11 pre-existing failures) + vitest/jsdom unpinned — fix before trusting the JS gate
