// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import Login from './Login.vue'

// ── Mock external dependencies ────────────────────────────────
// Hoisted so the spies are shared with the test bodies. Building them inside the
// factories (`push: vi.fn()`) hands out a fresh spy on every call, which cannot
// be asserted against.
const mocks = vi.hoisted(() => ({
  login: vi.fn(),
  push: vi.fn(),
}))

// The path must match what Login.vue imports ('../composables/useAuth.js' from
// src/views/). Vitest matches mocks by RESOLVED module, so a wrong relative path
// is a silent no-op that leaves the real composable in place.
//
// login() must also return the shape handleSubmit() reads -- it does
// `if (!result.success)` immediately, so a mock resolving to undefined throws
// TypeError. Vue swallows that rejection, so it does NOT fail a test by itself;
// the Submit path test below asserts router.push instead, which is only reached
// when handleSubmit runs to completion.
vi.mock('../composables/useAuth.js', () => ({
  useAuth: vi.fn(() => ({
    login: mocks.login,
    logout: vi.fn(),
    getToken: vi.fn(),
  })),
}))

vi.mock('vue-router', () => ({
  useRouter: vi.fn(() => ({
    push: mocks.push,
  })),
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

// handleSubmit awaits login() before touching the router, so a single $nextTick
// lands before the continuation. Yield to the microtask queue first.
async function flush() {
  await Promise.resolve()
  await Promise.resolve()
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

// ── Submit path ────────────────────────────────────────────────
// Exists to keep the useAuth mock honest. Every other test in this file only
// inspects rendered DOM, so the mock was never exercised -- which is how a mock
// pointing at an unresolvable path ('../../composables/useAuth.js') survived
// unnoticed while the real composable ran instead. This test fails if the mock
// stops being wired up OR if its login() stops returning the shape
// handleSubmit() reads (`result.success`).
describe('Submit path', () => {
  beforeEach(() => {
    mocks.login.mockReset()
    mocks.push.mockReset()
    window.matchMedia = mockMatchMedia(false)
  })

  it('passes the entered credentials to the mocked useAuth.login', async () => {
    mocks.login.mockResolvedValue({ success: true, mustChangePassword: false })
    const wrapper = mountLogin()

    await wrapper.find('#username').setValue('operator1')
    await wrapper.find('#password').setValue('Passw0rd!')
    await wrapper.find('form').trigger('submit')
    await flush()

    // Fails if the mock is not wired to the module Login.vue imports.
    // remember defaults to true (Login.vue:71) — "remember me is always on per
    // UX requirement", per the comment in useAuth.saveToken.
    expect(mocks.login).toHaveBeenCalledWith('operator1', 'Passw0rd!', true)
  })

  it('navigates on success — proves handleSubmit ran past result.success', async () => {
    mocks.login.mockResolvedValue({ success: true, mustChangePassword: false })
    const wrapper = mountLogin()

    await wrapper.find('#username').setValue('operator1')
    await wrapper.find('#password').setValue('Passw0rd!')
    await wrapper.find('form').trigger('submit')
    await flush()

    // router.push is the last statement of the success branch, so it is only
    // reached if `result.success` was readable. A mock resolving to undefined
    // throws before this and leaves push uncalled.
    expect(mocks.push).toHaveBeenCalledWith('/')
  })

  it('surfaces the error message on failure instead of navigating', async () => {
    mocks.login.mockResolvedValue({ success: false, error: 'invalid username or password' })
    const wrapper = mountLogin()

    await wrapper.find('#username').setValue('operator1')
    await wrapper.find('#password').setValue('wrong')
    await wrapper.find('form').trigger('submit')
    await flush()

    expect(mocks.push).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('invalid username or password')
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
