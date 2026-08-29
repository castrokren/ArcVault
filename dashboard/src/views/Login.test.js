// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import Login from './Login.vue'

// ── Mock external dependencies ────────────────────────────────
vi.mock('../../composables/useAuth.js', () => ({
  useAuth: vi.fn(() => ({
    login: vi.fn(),
    logout: vi.fn(),
    getToken: vi.fn(),
  })),
}))

vi.mock('vue-router', () => ({
  useRouter: vi.fn(() => ({
    push: vi.fn(),
  })),
}))

vi.mock('motion-v', () => ({
  motion: new Proxy(
    {},
    {
      get: () => ({
        props: ['initial', 'animate', 'transition', 'whileHover', 'whilePress'],
        template: '<div><slot /></div>',
      }),
    },
  ),
}))

// ── Factory: matchMedia mock ──────────────────────────────────
function mockMatchMedia(matches) {
  return vi.fn().mockImplementation((query) => ({
    matches,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

// ── Shared mount helper ────────────────────────────────────────
function mountLogin() {
  return mount(Login, {
    global: {
      stubs: {
        OrbitField: { template: '<canvas class="orbit-field-stub" />' },
        ChangePasswordModal: { template: '<div class="change-pw-stub" />' },
      },
    },
  })
}

// ── Arc Rings ───────────────────────────────────────────────────
describe('Arc rings', () => {
  beforeEach(() => {
    window.matchMedia = mockMatchMedia(false)
  })

  it('renders 3 arc rings when motion not reduced', () => {
    const wrapper = mountLogin()
    const arcs = wrapper.findAll('.arc')
    expect(arcs).toHaveLength(3)
  })

  it('renders arc container', () => {
    const wrapper = mountLogin()
    expect(wrapper.find('.arc-rings').exists()).toBe(true)
  })

  it('hides all arcs when prefers-reduced-motion', async () => {
    window.matchMedia = mockMatchMedia(true)
    const wrapper = mountLogin()
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.arc').exists()).toBe(false)
    expect(wrapper.find('.arc-rings').exists()).toBe(false)
  })

  it('assigns distinct classes to each arc', () => {
    window.matchMedia = mockMatchMedia(false)
    const wrapper = mountLogin()
    expect(wrapper.find('.arc-1').exists()).toBe(true)
    expect(wrapper.find('.arc-2').exists()).toBe(true)
    expect(wrapper.find('.arc-3').exists()).toBe(true)
  })

  it('each arc is a div element', () => {
    window.matchMedia = mockMatchMedia(false)
    const wrapper = mountLogin()
    const arcs = wrapper.findAll('.arc')
    arcs.forEach((arc) => {
      expect(arc.element.tagName).toBe('DIV')
    })
  })
})

// ── Reduced-motion aggregate ───────────────────────────────────
describe('Reduced motion (aggregate)', () => {
  it('hides decorative arc rings when prefers-reduced-motion', async () => {
    window.matchMedia = mockMatchMedia(true)
    const wrapper = mountLogin()
    await wrapper.vm.$nextTick()
    // Arcs hidden
    expect(wrapper.find('.arc').exists()).toBe(false)
    expect(wrapper.find('.arc-rings').exists()).toBe(false)
    // Core UI still renders
    expect(wrapper.find('.change-pw-stub').exists()).toBe(true)
  })
})
