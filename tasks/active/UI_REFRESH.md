# UI Refresh Task

**Status:** Planning  
**Branch:** feat/login-orbital (can start new branch or continue on this)  
**Date:** Jun 30, 2026

## What We're Doing
Dashboard UI refresh while keeping all functionality. The design system exists; we're improving component styling and visual hierarchy.

## Current State
- ✅ Login simplified to working minimal version (no animations)
- ✅ Dashboard shell, navigation, themes working
- 🔄 Ready for UI improvements

## Design System (Already In Place)
- **Fonts:** Space Grotesk (display), Inter (body), JetBrains Mono (code)
- **Colors:** Dark theme with purple/blue accents, semantic colors (success, error, warning, info)
- **File:** `dashboard/src/style.css` — all tokens defined

## High-Impact Changes (No Breaking Changes)
1. **Stat cards** — better shadows, borders, readability
2. **Tables** — improved rows, hover effects, spacing
3. **Buttons** — refined styling and feedback
4. **Badges** — more prominent with better contrast
5. **Global spacing** — better consistency
6. **Transitions** — subtle polish on interactions

## Key Files to Refresh
- `dashboard/src/views/Agents.vue` (primary user view)
- `dashboard/src/views/Jobs.vue`
- `dashboard/src/views/History.vue`
- `dashboard/src/App.vue` (navigation)
- `dashboard/src/style.css` (global styles)
- Component styles in `dashboard/src/components/`

## Suggested Approach
Start with **stat cards and table styling** (Agents view) — highest ROI for visual impact.

## Next Steps
1. Choose starting component (recommend: stat cards or table)
2. Update component CSS in scoped styles
3. Test on running coordinator
4. Move to other key views
5. Commit with descriptive messages

---
**Ready to pick up in desktop app. All infrastructure running and tested.**
