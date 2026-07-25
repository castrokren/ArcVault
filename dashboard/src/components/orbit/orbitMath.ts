// Pure math utilities for the orbital field scene
// No DOM, no Vue dependencies — testable pure functions.

export type PerfTier = 'high' | 'med' | 'low'

export interface OrbitParams {
  r: number
  tilt: number
  speed: number
  phase: number
}

export interface PlanetPosition {
  x: number
  y: number
  depth: number
}

export interface WarpState {
  e1: number
  e2: number
  boost: number
  zoom: number
  flash: boolean
  done: boolean
}

/**
 * Ease-in quadratic: t*t
 */
export function easeInQuad(t: number): number {
  return t * t
}

/**
 * Compute planet position in 2.5D orbital space.
 * θ = time * speed + phase
 * x = r * cos(θ)
 * y = r * sin(θ) * tilt
 * depth = sin(θ) → used for front/back occlusion sorting
 */
export function planetPos(orbit: OrbitParams, time: number): PlanetPosition {
  const theta = time * orbit.speed + orbit.phase
  return {
    x: orbit.r * Math.cos(theta),
    y: orbit.r * Math.sin(theta) * orbit.tilt,
    depth: Math.sin(theta),
  }
}

/**
 * Compute warp dive progress given elapsed seconds.
 * e1 drives boost (0→1 over 0.8s), e2 drives zoom (0→1 over 1.25s).
 * flash triggers at 85% of the full duration.
 * done signals full reset at 1.25s.
 */
export function warpProgress(tSec: number): WarpState {
  const e1 = Math.min(tSec / 0.8, 1)
  const e2 = Math.min(tSec / 1.25, 1)
  const flash = tSec >= 0.85 * 1.25
  const done = tSec >= 1.25
  return {
    e1,
    e2,
    boost: 1 + easeInQuad(e1) * 11,
    zoom: 1 + easeInQuad(e2) * 1.5,
    flash,
    done,
  }
}

/**
 * Determine performance tier from rolling average frame time (ms).
 * <18 → high, <33 → med, else low.
 * prev enables hysteresis: a single-frame downgrade is blocked
 * (stays at prev tier if it would otherwise downgrade).
 * Upgrades are always immediate.
 */
export function perfTier(avgFrameMs: number, prev?: PerfTier): PerfTier {
  let raw: PerfTier
  if (avgFrameMs < 18) {
    raw = 'high'
  } else if (avgFrameMs < 33) {
    raw = 'med'
  } else {
    raw = 'low'
  }

  // Hysteresis: block downgrade if prev is better than raw
  if (prev !== undefined) {
    const order: PerfTier[] = ['high', 'med', 'low']
    if (order.indexOf(raw) > order.indexOf(prev)) {
      return prev
    }
  }

  return raw
}
