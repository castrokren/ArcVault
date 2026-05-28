---
name: ArcVault Identity
category: system
priority: critical
last_updated: 2026-05-26
stale_after_days: 180
---

# ArcVault Identity

## Project

**ArcVault** is a cross-platform backup orchestrator built in Go with a Vue 3 dashboard. It is self-hosted, embeds the dashboard in a single binary, and uses lightweight agents that coordinate through a central coordinator.

**Owner:** Kren Castro
**Repository:** https://github.com/castrokren/ArcVault
**Current Release:** v1.0.0 (Phase 17 complete — production ready)

## What ArcVault Solves

RoboBackup was Windows-only with limited monitoring and no remote visibility. ArcVault replaces it with a cross-platform solution (Windows, macOS, Linux) with real-time monitoring, self-hosted deployment, and agent-to-coordinator orchestration.

## Architecture in One Line

Lightweight agents → central Go coordinator → Vue 3 dashboard (embedded in coordinator binary via `//go:embed`)

## AI Role

You are helping Kren build, maintain, and evolve ArcVault. You work across Go backend code (coordinator + agent), the Vue 3 frontend dashboard, testing, deployment, and architectural decisions.

Kren is the sole developer. Work should be efficient, avoid over-engineering, and favor shipping working features over perfect architecture.
