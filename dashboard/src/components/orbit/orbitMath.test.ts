import { describe, it, expect } from 'vitest'
import { easeInQuad, planetPos, warpProgress, perfTier } from './orbitMath'

describe('easeInQuad', () => {
  it('returns 0 for t=0', () => {
    expect(easeInQuad(0)).toBe(0)
  })

  it('returns 1 for t=1', () => {
    expect(easeInQuad(1)).toBe(1)
  })

  it('returns 0.25 for t=0.5', () => {
    expect(easeInQuad(0.5)).toBe(0.25)
  })

  it('is monotonic increasing', () => {
    const values = [0, 0.1, 0.3, 0.5, 0.7, 0.9, 1]
    for (let i = 1; i < values.length; i++) {
      expect(easeInQuad(values[i])).toBeGreaterThan(easeInQuad(values[i - 1]))
    }
  })
})

describe('planetPos', () => {
  const orbit = { r: 100, tilt: 0.5, speed: 1, phase: 0 }

  it('returns {x: r, y: 0, depth: 0} at θ=0', () => {
    const pos = planetPos(orbit, 0)
    expect(pos.x).toBeCloseTo(100, 10)
    expect(pos.y).toBeCloseTo(0, 10)
    expect(pos.depth).toBeCloseTo(0, 10)
  })

  it('returns {x: 0, y: r*tilt, depth: 1} at θ=π/2', () => {
    const pos = planetPos(orbit, Math.PI / 2)
    expect(pos.x).toBeCloseTo(0, 10)
    expect(pos.y).toBeCloseTo(100 * 0.5, 10)
    expect(pos.depth).toBeCloseTo(1, 10)
  })

  it('returns {x: -r, y: 0, depth: 0} at θ=π', () => {
    const pos = planetPos(orbit, Math.PI)
    expect(pos.x).toBeCloseTo(-100, 10)
    expect(pos.y).toBeCloseTo(0, 10)
    expect(pos.depth).toBeCloseTo(0, 10)
  })

  it('returns {x: 0, y: -r*tilt, depth: -1} at θ=3π/2', () => {
    const pos = planetPos(orbit, (3 * Math.PI) / 2)
    expect(pos.x).toBeCloseTo(0, 10)
    expect(pos.y).toBeCloseTo(-100 * 0.5, 10)
    expect(pos.depth).toBeCloseTo(-1, 10)
  })

  it('tilt scales Y but not X', () => {
    const orbitNarrow = { r: 100, tilt: 0.2, speed: 1, phase: 0 }
    const posNarrow = planetPos(orbitNarrow, Math.PI / 2)
    const posWide = planetPos({ ...orbitNarrow, tilt: 0.8 }, Math.PI / 2)
    // X unchanged
    expect(posNarrow.x).toBeCloseTo(posWide.x, 10)
    // Y scales with tilt
    expect(posNarrow.y).toBeCloseTo(100 * 0.2, 10)
    expect(posWide.y).toBeCloseTo(100 * 0.8, 10)
  })

  it('depth sign flips across the core (θ crosses π)', () => {
    const before = planetPos(orbit, Math.PI - 0.01)
    const after = planetPos(orbit, Math.PI + 0.01)
    expect(before.depth).toBeGreaterThan(0)
    expect(after.depth).toBeLessThan(0)
  })

  it('uses speed and phase correctly', () => {
    const custom = { r: 50, tilt: 1, speed: 2, phase: Math.PI }
    const pos = planetPos(custom, 0)
    // θ = 0*2 + π = π
    expect(pos.x).toBeCloseTo(-50, 10)
    expect(pos.y).toBeCloseTo(0, 10)
  })
})

describe('warpProgress', () => {
  it('at t=0: boost=1, zoom=1, flash=false, done=false', () => {
    const wp = warpProgress(0)
    expect(wp.e1).toBe(0)
    expect(wp.e2).toBe(0)
    expect(wp.boost).toBe(1)
    expect(wp.zoom).toBe(1)
    expect(wp.flash).toBe(false)
    expect(wp.done).toBe(false)
  })

  it('boost saturates at 12 when t≥0.8', () => {
    const wp = warpProgress(0.8)
    expect(wp.e1).toBe(1)
    expect(wp.boost).toBe(1 + 1 * 11)
    expect(wp.boost).toBe(12)
    // Also verify just past 0.8 is still 12
    const wp2 = warpProgress(1.0)
    expect(wp2.boost).toBe(12)
  })

  it('zoom saturates at 2.5 when t≥1.25', () => {
    const wp = warpProgress(1.25)
    expect(wp.e2).toBe(1)
    expect(wp.zoom).toBe(1 + 1 * 1.5)
    expect(wp.zoom).toBe(2.5)
  })

  it('flash is false before 1.0625', () => {
    expect(warpProgress(1.06).flash).toBe(false)
  })

  it('flash is true at t≥1.0625', () => {
    expect(warpProgress(1.0625).flash).toBe(true)
    expect(warpProgress(1.1).flash).toBe(true)
    expect(warpProgress(1.25).flash).toBe(true)
  })

  it('done is true at t≥1.25', () => {
    expect(warpProgress(1.249).done).toBe(false)
    expect(warpProgress(1.25).done).toBe(true)
    expect(warpProgress(2.0).done).toBe(true)
  })

  it('e1 and e2 are monotonic increasing', () => {
    const times = [0, 0.2, 0.4, 0.6, 0.8, 1.0, 1.25, 1.5]
    for (let i = 1; i < times.length; i++) {
      const a = warpProgress(times[i - 1])
      const b = warpProgress(times[i])
      expect(b.e1).toBeGreaterThanOrEqual(a.e1)
      expect(b.e2).toBeGreaterThanOrEqual(a.e2)
      expect(b.boost).toBeGreaterThanOrEqual(a.boost)
      expect(b.zoom).toBeGreaterThanOrEqual(a.zoom)
    }
  })

  it('boost starts at 1 and reaches 12 at t=0.8', () => {
    expect(warpProgress(0).boost).toBe(1)
    expect(warpProgress(0.4).boost).toBeGreaterThan(1)
    expect(warpProgress(0.8).boost).toBe(12)
  })

  it('zoom starts at 1 and reaches 2.5 at t=1.25', () => {
    expect(warpProgress(0).zoom).toBe(1)
    expect(warpProgress(0.6).zoom).toBeGreaterThan(1)
    expect(warpProgress(1.25).zoom).toBe(2.5)
  })
})

describe('perfTier', () => {
  it('returns high for avgFrameMs < 18', () => {
    expect(perfTier(10)).toBe('high')
    expect(perfTier(17)).toBe('high')
  })

  it('returns med at boundary 18', () => {
    expect(perfTier(18)).toBe('med')
  })

  it('returns med for avgFrameMs < 33', () => {
    expect(perfTier(19)).toBe('med')
    expect(perfTier(32)).toBe('med')
  })

  it('returns low at boundary 33', () => {
    expect(perfTier(33)).toBe('low')
  })

  it('returns low for avgFrameMs >= 33', () => {
    expect(perfTier(40)).toBe('low')
    expect(perfTier(100)).toBe('low')
  })

  it('hysteresis blocks a single-frame downgrade', () => {
    // Should stay at previous tier when it would downgrade
    expect(perfTier(40, 'med')).toBe('med')
    expect(perfTier(40, 'high')).toBe('high')
    expect(perfTier(25, 'high')).toBe('high')
  })

  it('does not block upgrades (immediate upgrade)', () => {
    expect(perfTier(10, 'low')).toBe('high')
    expect(perfTier(20, 'low')).toBe('med')
    expect(perfTier(10, 'med')).toBe('high')
  })

  it('no hysteresis when prev matches current tier', () => {
    expect(perfTier(10, 'high')).toBe('high')
    expect(perfTier(25, 'med')).toBe('med')
    expect(perfTier(50, 'low')).toBe('low')
  })
})
