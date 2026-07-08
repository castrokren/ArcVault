// @vitest-environment jsdom
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import Jobs from './Jobs.vue'
import * as api from '../api.js'

// Mock the API
vi.mock('../api.js', () => ({
  getJobs: vi.fn(() => Promise.resolve({ data: [], total: 0, page: 1, pages: 0, limit: 25 })),
  createJob: vi.fn(() => Promise.resolve({ id: 'job-1' })),
  deleteJob: vi.fn(() => Promise.resolve()),
  getFederationJobs: vi.fn(() => Promise.resolve({ jobs: [], stale: false })),
  getAgents: vi.fn(() => Promise.resolve({ agents: [{ id: 'agent-1', name: 'Agent 1' }] })),
  getGroups: vi.fn(() => Promise.resolve({ groups: [] })),
  getToken: vi.fn(() => 'mock-token'),
  getJobRuns: vi.fn(() => Promise.resolve({ data: [] }))
}))

// Mock composition API inject
vi.mock('vue', async () => {
  const actual = await vi.importActual('vue')
  return {
    ...actual,
    inject: () => ({ value: null })
  }
})

describe('Jobs.vue Integration with SyncFlagsBuilder', () => {
  let wrapper

  beforeEach(async () => {
    vi.clearAllMocks()
    wrapper = mount(Jobs, {
      global: {
        stubs: {
          Pagination: true,
          ScheduleBuilder: true,
          SyncFlagsBuilder: true
        }
      }
    })
    await wrapper.vm.$nextTick()
  })

  describe('Form initialization', () => {
    it('initializes sync_flags in form', () => {
      expect(wrapper.vm.form.sync_flags).toBeDefined()
    })

    it('sync_flags defaults to empty object', () => {
      expect(wrapper.vm.form.sync_flags).toEqual({})
    })
  })

  describe('Job creation with sync_flags', () => {
    it('includes sync_flags in API payload when set', async () => {
      wrapper.vm.form.dispatchMode = 'agent'
      wrapper.vm.form.agent_id = 'agent-1'
      wrapper.vm.form.name = 'test-job'
      wrapper.vm.form.source_path = '/src'
      wrapper.vm.form.dest_path = '/dest'
      wrapper.vm.form.sync_flags = {
        mirror: true,
        max_age: 30,
        min_age: null,
        max_size: null,
        exclude_files: ['*.tmp'],
        exclude_dirs: ['.git']
      }

      await wrapper.vm.createJob()

      expect(api.createJob).toHaveBeenCalledWith(
        expect.objectContaining({
          agent_id: 'agent-1',
          name: 'test-job',
          source_path: '/src',
          dest_path: '/dest',
          sync_flags: {
            mirror: true,
            max_age: 30,
            exclude_files: ['*.tmp'],
            exclude_dirs: ['.git']
          }
        })
      )
    })

    it('omits sync_flags from payload when empty', async () => {
      wrapper.vm.form.dispatchMode = 'agent'
      wrapper.vm.form.agent_id = 'agent-1'
      wrapper.vm.form.name = 'test-job'
      wrapper.vm.form.source_path = '/src'
      wrapper.vm.form.dest_path = '/dest'
      wrapper.vm.form.sync_flags = {
        mirror: false,
        max_age: null,
        min_age: null,
        max_size: null,
        exclude_files: [],
        exclude_dirs: []
      }

      await wrapper.vm.createJob()

      const payload = api.createJob.mock.calls[0][0]
      expect(payload.sync_flags).toBeUndefined()
    })

    it('omits sync_flags when object is empty', async () => {
      wrapper.vm.form.dispatchMode = 'agent'
      wrapper.vm.form.agent_id = 'agent-1'
      wrapper.vm.form.name = 'test-job'
      wrapper.vm.form.source_path = '/src'
      wrapper.vm.form.dest_path = '/dest'
      wrapper.vm.form.sync_flags = {}

      await wrapper.vm.createJob()

      const payload = api.createJob.mock.calls[0][0]
      expect(payload.sync_flags).toBeUndefined()
    })

    it('resets sync_flags after successful job creation', async () => {
      wrapper.vm.form.dispatchMode = 'agent'
      wrapper.vm.form.agent_id = 'agent-1'
      wrapper.vm.form.name = 'test-job'
      wrapper.vm.form.source_path = '/src'
      wrapper.vm.form.dest_path = '/dest'
      wrapper.vm.form.sync_flags = { mirror: true }

      await wrapper.vm.createJob()
      await wrapper.vm.$nextTick()

      expect(wrapper.vm.form.sync_flags).toEqual({})
    })

    it('includes sync_flags with only some fields set', async () => {
      wrapper.vm.form.dispatchMode = 'agent'
      wrapper.vm.form.agent_id = 'agent-1'
      wrapper.vm.form.name = 'test-job'
      wrapper.vm.form.source_path = '/src'
      wrapper.vm.form.dest_path = '/dest'
      wrapper.vm.form.sync_flags = {
        mirror: false,
        max_age: 7,
        min_age: null,
        max_size: null,
        exclude_files: [],
        exclude_dirs: []
      }

      await wrapper.vm.createJob()

      const payload = api.createJob.mock.calls[0][0]
      expect(payload.sync_flags).toBeDefined()
      expect(payload.sync_flags.max_age).toBe(7)
    })
  })

  describe('Job creation without sync_flags', () => {
    it('creates job successfully with basic fields only', async () => {
      wrapper.vm.form.dispatchMode = 'agent'
      wrapper.vm.form.agent_id = 'agent-1'
      wrapper.vm.form.name = 'basic-job'
      wrapper.vm.form.source_path = '/src'
      wrapper.vm.form.dest_path = '/dest'

      await wrapper.vm.createJob()

      expect(api.createJob).toHaveBeenCalled()
      const payload = api.createJob.mock.calls[0][0]
      expect(payload.name).toBe('basic-job')
      expect(payload.sync_flags).toBeUndefined()
    })
  })

  describe('Form reset', () => {
    it('resets all form fields including sync_flags after creation', async () => {
      wrapper.vm.form.dispatchMode = 'agent'
      wrapper.vm.form.agent_id = 'agent-1'
      wrapper.vm.form.name = 'test-job'
      wrapper.vm.form.source_path = '/src'
      wrapper.vm.form.dest_path = '/dest'
      wrapper.vm.form.schedule = 'daily'
      wrapper.vm.form.sync_flags = { mirror: true }

      await wrapper.vm.createJob()
      await wrapper.vm.$nextTick()

      expect(wrapper.vm.form.agent_id).toBe('')
      expect(wrapper.vm.form.name).toBe('')
      expect(wrapper.vm.form.source_path).toBe('')
      expect(wrapper.vm.form.dest_path).toBe('')
      expect(wrapper.vm.form.schedule).toBe('')
      expect(wrapper.vm.form.sync_flags).toEqual({})
    })
  })
})
