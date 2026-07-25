// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import App from './App.vue'

const { checkUpdateMock, getVersionMock, wsState } = vi.hoisted(() => ({
  checkUpdateMock: vi.fn().mockResolvedValue({
    current: 'v0.6.0', latest: 'v0.6.0', update_available: false, release_url: '', asset_url: ''
  }),
  getVersionMock: vi.fn().mockResolvedValue({
    version: 'v0.6.0', build_time: '2026-07-15T00:00:00Z', os: 'linux', arch: 'amd64', go_version: 'go1.25', uptime: '10h'
  }),
  wsState: { lastEvent: null }
}))

vi.mock('./api', () => ({
  getToken: vi.fn(() => 'fake-token'),
  saveToken: vi.fn(),
  applyCoordinatorUpdate: vi.fn(),
  checkUpdate: checkUpdateMock,
  getVersion: getVersionMock
}))

vi.mock('./composables/useWebSocket.js', async () => {
  const { ref } = await vi.importActual('vue')
  wsState.lastEvent = ref(null)
  return {
    useWebSocket: vi.fn(() => ({
      connected: { value: true },
      lastEvent: wsState.lastEvent,
      connect: vi.fn(),
      disconnect: vi.fn()
    }))
  }
})

vi.mock('./composables/useAuth.js', () => ({
  useAuth: vi.fn(() => ({
    isAuthenticated: { value: true },
    currentUser: { value: { username: 'admin', role: 'admin' } },
    hasRole: vi.fn(() => true),
    refreshToken: vi.fn(),
    logout: vi.fn()
  }))
}))

vi.mock('vue-router', () => ({
  useRouter: vi.fn(() => ({ push: vi.fn() }))
}))

it('re-fetches version and update state after successful update', async () => {
  const wrapper = mount(App, {
    global: {
      stubs: {
        UpdateBanner: { template: '<div class="banner-stub" />' },
        ChangePasswordModal: { template: '<div class="cp-stub" />' },
        'router-view': { template: '<div><slot /></div>' },
        'router-link': { template: '<a><slot /></a>' }
      }
    }
  })

  await nextTick()

  // Initial onMounted calls
  expect(checkUpdateMock).toHaveBeenCalledTimes(1)
  expect(getVersionMock).toHaveBeenCalledTimes(1)

  // Simulate WebSocket sending done event → UpdateModal emits 'updated' → App re-fetches
  wsState.lastEvent.value = { type: 'update_progress', payload: { step: 'done', pct: 100, message: '' } }
  await nextTick()

  expect(checkUpdateMock).toHaveBeenCalledTimes(2)
  expect(getVersionMock).toHaveBeenCalledTimes(2)
})
