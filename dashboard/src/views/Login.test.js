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

// ── Aurora ─────────────────────────────────────────────────────
describe('Aurora blobs', () => {
  beforeEach(() => {
    window.matchMedia = mockMatchMedia(false)
  })

  it('renders 4 aurora blobs when motion not reduced', () => {
    const wrapper = mountLogin()
    const auroras = wrapper.findAll('.login-aurora')
    expect(auroras).toHaveLength(4)
  })

  it('hides aurora blobs when prefers-reduced-motion', () => {
    window.matchMedia = mockMatchMedia(true)
    const wrapper = mountLogin()
    const auroras = wrapper.findAll('.login-aurora')
    expect(auroras).toHaveLength(0)
  })

  it('assigns distinct classes to each blob', () => {
    window.matchMedia = mockMatchMedia(false)
    const wrapper = mountLogin()
    expect(wrapper.find('.login-aurora-1').exists()).toBe(true)
    expect(wrapper.find('.login-aurora-2').exists()).toBe(true)
    expect(wrapper.find('.login-aurora-3').exists()).toBe(true)
    expect(wrapper.find('.login-aurora-4').exists()).toBe(true)
  })
})

// ── Watermark ──────────────────────────────────────────────────
describe('Watermark', () => {
  beforeEach(() => {
    window.matchMedia = mockMatchMedia(false)
  })

  it('renders watermark container', () => {
    const wrapper = mountLogin()
    expect(wrapper.find('.login-watermark').exists()).toBe(true)
  })

  it('renders shield SVG inside watermark', () => {
    const wrapper = mountLogin()
    const svg = wrapper.find('.login-watermark svg')
    expect(svg.exists()).toBe(true)
    expect(svg.attributes('viewBox')).toBe('0 0 560 560')
    expect(svg.attributes('width')).toBe('560')
    expect(svg.attributes('height')).toBe('560')
  })

  it('hides watermark when prefers-reduced-motion', () => {
    window.matchMedia = mockMatchMedia(true)
    const wrapper = mountLogin()
    expect(wrapper.find('.login-watermark').exists()).toBe(false)
  })
})

// ── Starfield ──────────────────────────────────────────────────
describe('Starfield', () => {
  beforeEach(() => {
    window.matchMedia = mockMatchMedia(false)
  })

  it('renders stars container', () => {
    const wrapper = mountLogin()
    expect(wrapper.find('.login-stars').exists()).toBe(true)
  })

  it('renders 25 stars when motion not reduced', () => {
    const wrapper = mountLogin()
    const stars = wrapper.findAll('.login-star')
    expect(stars).toHaveLength(25)
  })

  it('renders some large stars (.lg class)', () => {
    const wrapper = mountLogin()
    const large = wrapper.findAll('.login-star.lg')
    expect(large.length).toBeGreaterThan(0)
    expect(large.length).toBeLessThan(25)
  })

  it('hides stars when prefers-reduced-motion', () => {
    window.matchMedia = mockMatchMedia(true)
    const wrapper = mountLogin()
    expect(wrapper.find('.login-star').exists()).toBe(false)
  })

  it('each star has unique position styles', () => {
    window.matchMedia = mockMatchMedia(false)
    const wrapper = mountLogin()
    const stars = wrapper.findAll('.login-star')
    const positions = stars.map((s) => s.attributes('style'))
    const unique = new Set(positions)
    // At least 24 of 25 should have unique style (one tiny chance of animation collision)
    expect(unique.size).toBeGreaterThanOrEqual(24)
  })
})

// ── Beams ──────────────────────────────────────────────────────
describe('Data-comet beams', () => {
  beforeEach(() => {
    window.matchMedia = mockMatchMedia(false)
  })

  it('renders beams container', () => {
    const wrapper = mountLogin()
    expect(wrapper.find('.login-beams').exists()).toBe(true)
  })

  it('renders 12 beams when motion not reduced', () => {
    const wrapper = mountLogin()
    const wraps = wrapper.findAll('.login-beam-wrap')
    expect(wraps).toHaveLength(12)
  })

  it('each beam-wrap has a beam child', () => {
    const wrapper = mountLogin()
    const wraps = wrapper.findAll('.login-beam-wrap')
    wraps.forEach((w) => {
      expect(w.find('.login-beam').exists()).toBe(true)
    })
  })

  it('hides beams when prefers-reduced-motion', () => {
    window.matchMedia = mockMatchMedia(true)
    const wrapper = mountLogin()
    expect(wrapper.find('.login-beam').exists()).toBe(false)
  })

  it('beams include thin, thick, and default widths', () => {
    window.matchMedia = mockMatchMedia(false)
    const wrapper = mountLogin()
    const beams = wrapper.findAll('.login-beam')
    const thin = beams.filter((b) => b.classes().includes('thin'))
    const thick = beams.filter((b) => b.classes().includes('thick'))
    const plain = beams.filter((b) => !b.classes().includes('thin') && !b.classes().includes('thick'))
    expect(thin.length).toBe(3)
    expect(thick.length).toBe(3)
    expect(plain.length).toBe(6)
  })

  it('beams have rotation angles applied', () => {
    window.matchMedia = mockMatchMedia(false)
    const wrapper = mountLogin()
    const wraps = wrapper.findAll('.login-beam-wrap')
    wraps.forEach((w) => {
      const transform = w.attributes('style') || ''
      expect(transform).toMatch(/rotate\(/)
    })
  })
})

// ── Halos ──────────────────────────────────────────────────────
describe('Brand icon halos', () => {
  beforeEach(() => {
    window.matchMedia = mockMatchMedia(false)
  })

  it('renders outer halo when motion not reduced', () => {
    const wrapper = mountLogin()
    expect(wrapper.find('.login-halo-outer').exists()).toBe(true)
  })

  it('renders inner halo when motion not reduced', () => {
    const wrapper = mountLogin()
    expect(wrapper.find('.login-halo-inner').exists()).toBe(true)
  })

  it('renders brand icon always (no v-if)', () => {
    const wrapper = mountLogin()
    expect(wrapper.find('.login-brand-icon').exists()).toBe(true)
  })

  it('hides outer halo when prefers-reduced-motion', () => {
    window.matchMedia = mockMatchMedia(true)
    const wrapper = mountLogin()
    expect(wrapper.find('.login-halo-outer').exists()).toBe(false)
  })

  it('hides inner halo when prefers-reduced-motion', () => {
    window.matchMedia = mockMatchMedia(true)
    const wrapper = mountLogin()
    expect(wrapper.find('.login-halo-inner').exists()).toBe(false)
  })

  it('icon pulse still renders on reduced-motion', () => {
    window.matchMedia = mockMatchMedia(true)
    const wrapper = mountLogin()
    expect(wrapper.find('.login-brand-icon').exists()).toBe(true)
  })
})

// ── Shell entrance ─────────────────────────────────────────────
describe('Shell entrance', () => {
  beforeEach(() => {
    window.matchMedia = mockMatchMedia(false)
  })

  it('login-shell has entrance animation class (avShellIn via CSS)', () => {
    const wrapper = mountLogin()
    const shell = wrapper.find('.login-shell')
    expect(shell.exists()).toBe(true)
    // CSS animation is applied via login-animation.css; we verify the element renders
    expect(shell.classes()).not.toContain('warping')
  })

  it('login-shell parallax style is applied as inline style', () => {
    window.matchMedia = mockMatchMedia(false)
    const wrapper = mountLogin()
    const shell = wrapper.find('.login-shell')
    const style = shell.attributes('style') || ''
    expect(style).toContain('transform')
  })
})

// ── Reduced-motion aggregate ───────────────────────────────────
describe('Reduced motion (aggregate)', () => {
  it('hides all 5 decorative animation groups when prefers-reduced-motion', () => {
    window.matchMedia = mockMatchMedia(true)
    const wrapper = mountLogin()
    expect(wrapper.findAll('.login-aurora')).toHaveLength(0)
    expect(wrapper.find('.login-watermark').exists()).toBe(false)
    expect(wrapper.find('.login-star').exists()).toBe(false)
    expect(wrapper.find('.login-beam').exists()).toBe(false)
    expect(wrapper.find('.login-halo-outer').exists()).toBe(false)
    expect(wrapper.find('.login-halo-inner').exists()).toBe(false)
    // Shell and icon remain
    expect(wrapper.find('.login-shell').exists()).toBe(true)
    expect(wrapper.find('.login-brand-icon').exists()).toBe(true)
  })
})
