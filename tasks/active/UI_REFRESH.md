# UI Refresh Task

**Status:** In Progress — Color scheme done, login page next  
**Last session:** Session 29, Jun 30, 2026

## What We're Doing
Dashboard UI refresh while keeping all functionality. Color scheme landed; next target is the login page.

## Completed This Sprint
- ✅ Jobs.vue — stripped 350+ lines hardcoded CSS → token-based scoped styles (~60 lines)
- ✅ style.css — full dark theme rewrite (navy/charcoal base + cyan/teal accents)
- ✅ charts.css — created; token-driven bar charts, sparklines, donut gauges
- ✅ History.vue — rebuilt stat cards (Total Runs bar, Completed sparkline, Success Rate donut)
- ✅ vite.config.js — proxy `/api` → `https://localhost` (dev server now works with backend)

## Current Color Scheme (Dark Theme)
| Token | Value | Role |
|---|---|---|
| `--bg-base` | `#141824` | Page background |
| `--bg-card` | `#1c2236` | Cards |
| `--bg-elevated` | `#222a42` | Elevated surfaces, modals |
| `--accent` | `#00d4e8` | Cyan — primary data/interaction |
| `--accent-2` | `#00e58a` | Green — secondary charts/badges |

## Next Session: Login Page
Design a new login page to match the dashboard's navy/cyan aesthetic.
- Reference: the existing `Login.vue` has an orbit animation scene (may want to retheme it)
- Match surface colors to new design system tokens
- Keep all auth functionality intact (JWT, redirect after login)
- File: `dashboard/src/views/Login.vue`
- Related: `dashboard/src/login-animation.css`

## Design System Files
- **Tokens**: `dashboard/src/style.css`
- **Chart helpers**: `dashboard/src/charts.css`
- **Fonts**: Space Grotesk (display), Inter (body), JetBrains Mono (mono)

---
**Infrastructure is running. Dev server: `npm run dev` in `dashboard/` (proxy to https://localhost).**
