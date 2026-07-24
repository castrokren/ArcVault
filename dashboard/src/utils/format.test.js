import { describe, it, expect } from 'vitest'
import { versionBehind } from './format.js'

describe('versionBehind', () => {
  it('flags an agent strictly behind the baseline', () => {
    expect(versionBehind('v0.5.1', 'v0.6.0')).toBe(true)
    expect(versionBehind('0.5.1', '0.6.0')).toBe(true)   // leading v optional
    expect(versionBehind('v0.6.0', 'v0.6.1')).toBe(true) // patch drift
  })

  it('does not flag equal or newer versions', () => {
    expect(versionBehind('v0.6.0', 'v0.6.0')).toBe(false)
    expect(versionBehind('v0.7.0', 'v0.6.0')).toBe(false) // newer than baseline
    expect(versionBehind('v0.6.0-rc1', 'v0.6.0')).toBe(false) // pre-release == release core
  })

  it('never flags on missing data (fail safe)', () => {
    expect(versionBehind('', 'v0.6.0')).toBe(false)
    expect(versionBehind('v0.5.0', '')).toBe(false)
    expect(versionBehind(undefined, undefined)).toBe(false)
  })
})
