import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import SyncFlagsBuilder from './SyncFlagsBuilder.vue'

describe('SyncFlagsBuilder.vue', () => {
  let wrapper

  beforeEach(() => {
    wrapper = mount(SyncFlagsBuilder, {
      props: {
        modelValue: {
          mirror: false,
          max_age: null,
          min_age: null,
          max_size: null,
          exclude_files: [],
          exclude_dirs: []
        }
      }
    })
  })

  describe('Component Rendering', () => {
    it('renders collapsed by default', () => {
      expect(wrapper.find('.advanced-content').exists()).toBe(false)
      expect(wrapper.find('.advanced-header').exists()).toBe(true)
    })

    it('expands when header is clicked', async () => {
      await wrapper.find('.advanced-header').trigger('click')
      expect(wrapper.find('.advanced-content').exists()).toBe(true)
    })

    it('renders all three sections (Filtering, Behavior, Exclusions)', async () => {
      await wrapper.find('.advanced-header').trigger('click')
      expect(wrapper.find('.filtering-section').exists()).toBe(true)
      expect(wrapper.find('.behavior-section').exists()).toBe(true)
      expect(wrapper.find('.exclusions-section').exists()).toBe(true)
    })
  })

  describe('Form Fields', () => {
    beforeEach(async () => {
      await wrapper.find('.advanced-header').trigger('click')
    })

    it('has Max Age input field', () => {
      expect(wrapper.find('#max-age').exists()).toBe(true)
    })

    it('has Min Age input field', () => {
      expect(wrapper.find('#min-age').exists()).toBe(true)
    })

    it('has Max Size input field', () => {
      expect(wrapper.find('#max-size').exists()).toBe(true)
    })

    it('has Mirror checkbox', () => {
      expect(wrapper.find('#mirror').exists()).toBe(true)
    })

    it('has Exclude Files textarea', () => {
      expect(wrapper.find('#exclude-files').exists()).toBe(true)
    })

    it('has Exclude Directories textarea', () => {
      expect(wrapper.find('#exclude-dirs').exists()).toBe(true)
    })
  })

  describe('v-model Binding', () => {
    beforeEach(async () => {
      await wrapper.find('.advanced-header').trigger('click')
    })

    it('updates max_age when input changes', async () => {
      const input = wrapper.find('#max-age')
      await input.setValue(30)
      await wrapper.vm.$nextTick()
      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      const emitted = wrapper.emitted('update:modelValue')[0][0]
      expect(emitted.max_age).toBe(30)
    })

    it('updates mirror flag when checkbox changes', async () => {
      const checkbox = wrapper.find('#mirror')
      await checkbox.trigger('input')
      await wrapper.vm.$nextTick()
      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      const emitted = wrapper.emitted('update:modelValue')[0][0]
      expect(emitted.mirror).toBe(true)
    })

    it('parses exclude patterns correctly', async () => {
      const textarea = wrapper.find('#exclude-files')
      await textarea.setValue('*.tmp\n*.log\n*.bak')
      await textarea.trigger('input')
      await wrapper.vm.$nextTick()
      const emitted = wrapper.emitted('update:modelValue')[0][0]
      expect(emitted.exclude_files).toEqual(['*.tmp', '*.log', '*.bak'])
    })

    it('ignores empty lines in exclude patterns', async () => {
      const textarea = wrapper.find('#exclude-files')
      await textarea.setValue('*.tmp\n\n*.log\n')
      await textarea.trigger('input')
      await wrapper.vm.$nextTick()
      const emitted = wrapper.emitted('update:modelValue')[0][0]
      expect(emitted.exclude_files).toEqual(['*.tmp', '*.log'])
    })

    it('trims whitespace from exclude patterns', async () => {
      const textarea = wrapper.find('#exclude-files')
      await textarea.setValue('  *.tmp  \n  *.log  ')
      await textarea.trigger('input')
      await wrapper.vm.$nextTick()
      const emitted = wrapper.emitted('update:modelValue')[0][0]
      expect(emitted.exclude_files).toEqual(['*.tmp', '*.log'])
    })
  })

  describe('Validation', () => {
    beforeEach(async () => {
      await wrapper.find('.advanced-header').trigger('click')
    })

    it('shows error when min_age > max_age', async () => {
      const minInput = wrapper.find('#min-age')
      const maxInput = wrapper.find('#max-age')

      await maxInput.setValue(10)
      await maxInput.trigger('input')
      await wrapper.vm.$nextTick()

      await minInput.setValue(20)
      await minInput.trigger('input')
      await wrapper.vm.$nextTick()

      expect(wrapper.vm.validationErrors.minMaxAge).toBe('Min Age must be ≤ Max Age')
      expect(wrapper.text()).toContain('Min Age must be ≤ Max Age')
    })

    it('clears error when min_age becomes <= max_age', async () => {
      const minInput = wrapper.find('#min-age')
      const maxInput = wrapper.find('#max-age')

      // Set invalid state
      await maxInput.setValue(10)
      await maxInput.trigger('input')
      await wrapper.vm.$nextTick()
      await minInput.setValue(20)
      await minInput.trigger('input')
      await wrapper.vm.$nextTick()

      expect(wrapper.vm.validationErrors.minMaxAge).toBeTruthy()

      // Fix to valid state
      await maxInput.setValue(30)
      await maxInput.trigger('input')
      await wrapper.vm.$nextTick()

      expect(wrapper.vm.validationErrors.minMaxAge).toBe('')
    })

    it('allows valid min/max age combinations', async () => {
      const minInput = wrapper.find('#min-age')
      const maxInput = wrapper.find('#max-age')

      await minInput.setValue(5)
      await minInput.trigger('input')
      await wrapper.vm.$nextTick()

      await maxInput.setValue(10)
      await maxInput.trigger('input')
      await wrapper.vm.$nextTick()

      expect(wrapper.vm.validationErrors.minMaxAge).toBe('')
    })
  })

  describe('Command Preview', () => {
    beforeEach(async () => {
      await wrapper.find('.advanced-header').trigger('click')
    })

    it('generates robocopy command with mirror flag', async () => {
      const checkbox = wrapper.find('#mirror')
      await checkbox.trigger('input')
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('/MIR')
    })

    it('generates robocopy command with max age', async () => {
      const input = wrapper.find('#max-age')
      await input.setValue(30)
      await input.trigger('input')
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('/MAXAGE:30')
    })

    it('generates robocopy command with exclude patterns', async () => {
      const textarea = wrapper.find('#exclude-files')
      await textarea.setValue('*.tmp\n*.log')
      await textarea.trigger('input')
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('/XF *.tmp *.log')
    })

    it('generates rsync command with delete flag', async () => {
      const checkbox = wrapper.find('#mirror')
      await checkbox.trigger('input')
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('--delete')
    })

    it('converts max_age to seconds for rsync', async () => {
      const input = wrapper.find('#max-age')
      await input.setValue(1)
      await input.trigger('input')
      await wrapper.vm.$nextTick()
      // 1 day = 86400 seconds
      expect(wrapper.text()).toContain('--max-age=86400')
    })

    it('converts max_size to bytes for rsync', async () => {
      const input = wrapper.find('#max-size')
      await input.setValue(1)
      await input.trigger('input')
      await wrapper.vm.$nextTick()
      // 1 MB = 1048576 bytes
      expect(wrapper.text()).toContain('--maxsize=1048576')
    })

    it('generates rsync command with exclude patterns', async () => {
      const textarea = wrapper.find('#exclude-files')
      await textarea.setValue('*.tmp')
      await textarea.trigger('input')
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain("--exclude='*.tmp'")
    })
  })

  describe('External v-model updates', () => {
    it('syncs when parent updates modelValue', async () => {
      const newValue = {
        mirror: true,
        max_age: 15,
        min_age: 5,
        max_size: 1024,
        exclude_files: ['*.tmp'],
        exclude_dirs: ['.git']
      }

      await wrapper.setProps({ modelValue: newValue })
      await wrapper.vm.$nextTick()

      expect(wrapper.vm.flags).toEqual(newValue)
    })
  })

  describe('Empty state', () => {
    it('renders command preview even with empty flags', async () => {
      await wrapper.find('.advanced-header').trigger('click')
      expect(wrapper.find('.command-preview').exists()).toBe(true)
      expect(wrapper.text()).toContain('robocopy')
      expect(wrapper.text()).toContain('rsync')
    })
  })
})
