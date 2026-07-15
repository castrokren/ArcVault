// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import UpdateModal from './UpdateModal.vue'

vi.mock('../api', () => ({
  getToken: vi.fn(() => 'fake-token'),
  saveToken: vi.fn(),
  applyCoordinatorUpdate: vi.fn().mockResolvedValue({})
}))

vi.mock('../composables/useAuth.js', () => ({
  useAuth: vi.fn(() => ({
    refreshToken: vi.fn().mockResolvedValue(true)
  }))
}))

const mockUpdateStore = { current: 'v1.0.0', latest: 'v2.0.0', available: true, releaseUrl: '' }

function mountModal() {
  return mount(UpdateModal, {
    props: { isOpen: true, lastEvent: null },
    global: {
      provide: { updateStore: mockUpdateStore }
    }
  })
}

it('emits updated on done step', async () => {
  const wrapper = mountModal()
  await wrapper.setProps({
    lastEvent: { type: 'update_progress', payload: { step: 'done', pct: 100, message: '' } }
  })
  expect(wrapper.emitted('updated')).toHaveLength(1)
  expect(wrapper.text()).toContain('Update Complete')
})

it('emits updated on done_manual step', async () => {
  const wrapper = mountModal()
  await wrapper.setProps({
    lastEvent: { type: 'update_progress', payload: { step: 'done_manual', pct: 100, message: '' } }
  })
  expect(wrapper.emitted('updated')).toHaveLength(1)
  expect(wrapper.text()).toContain('Binary Updated')
})

it('does not emit updated on error step', async () => {
  const wrapper = mountModal()
  await wrapper.setProps({
    lastEvent: { type: 'update_progress', payload: { step: 'error', message: 'boom' } }
  })
  expect(wrapper.emitted('updated')).toBeUndefined()
})
