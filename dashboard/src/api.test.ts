// @vitest-environment node
import { describe, it, expect, vi } from 'vitest'

describe('getVersion', () => {
  it('is exported as a function from api.ts', async () => {
    // This will FAIL on current code because getVersion isn't exported.
    // After the fix (export keyword added), it passes.
    const { getVersion } = await import('./api')
    expect(typeof getVersion).toBe('function')
  })

  it('fetches version from /api/version and validates response shape', async () => {
    const fakeVersion = {
      version: 'v1.2.3',
      build_time: '2026-07-15T00:00:00Z',
      os: 'linux',
      arch: 'amd64',
      go_version: 'go1.25',
      uptime: '10h'
    }
    // ponytail: shim localStorage for getToken() in api.ts
    global.localStorage = { getItem: () => '' } as any
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve(fakeVersion),
      text: () => Promise.resolve(''),
    })

    const { getVersion } = await import('./api')
    const result = await getVersion()
    expect(result.version).toBe('v1.2.3')
    expect(result.build_time).toBe('2026-07-15T00:00:00Z')

    // Cleanup
    vi.restoreAllMocks()
    delete (global as any).localStorage
  })
})
