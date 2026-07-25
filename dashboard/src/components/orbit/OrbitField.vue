<template>
  <canvas
    ref="canvasEl"
    class="orbit-canvas"
    aria-hidden="true"
  ></canvas>
</template>

<script setup lang="ts">
// ── Imports ─────────────────────────────────────────────────
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { warpProgress, perfTier } from './orbitMath'
import type { PerfTier } from './orbitMath'

// ── Types ───────────────────────────────────────────────────
interface PlanetTypeDef {
  light: string; mid: string; dark: string; atmo: string
}
interface Pulse {
  t: number; spd: number
}
interface Moon {
  ang: number
}
interface OrbitNode {
  type: string; size: number; ring: boolean
  ang: number; moon: Moon | null; pulses: Pulse[]
}
interface OrbitDef {
  r: number; sp: number; tilt: number; nodes: OrbitNode[]
}
interface Star {
  x: number; y: number; z: number
  r: number; par: number; tws: number; tp: number
}
interface Comet {
  x: number; y: number; vx: number; vy: number
}
interface WarpState {
  active: boolean; start: number; onDone: (() => void) | null
  flashed: boolean; flashTime: number
}
interface NodeRender {
  o: OrbitDef; nd: OrbitNode; x: number; y: number; depth: number
}

// ── Planet type definitions ─────────────────────────────────
const TYPES: Record<string, PlanetTypeDef> = {
  ice:   { light: '#c4a9ff', mid: '#7a52d6', dark: '#1e1340', atmo: '125,77,255' },
  blue:  { light: '#aecaff', mid: '#3f6bd0', dark: '#0a1c46', atmo: '91,107,255' },
  violet:{ light: '#ddc2ff', mid: '#8b5cf6', dark: '#281748', atmo: '139,92,246' },
  rocky: { light: '#f1d9a8', mid: '#b5894c', dark: '#352510', atmo: '214,180,120' },
  mars:  { light: '#ffa078', mid: '#c2573a', dark: '#341310', atmo: '255,120,90' },
  pale:  { light: '#d8f1ff', mid: '#5fa3c4', dark: '#12303f', atmo: '150,210,235' },
}

// ── Props ───────────────────────────────────────────────────
const props = withDefaults(defineProps<{
  motion?: 'subtle' | 'bold' | 'maximal'
}>(), { motion: 'bold' })

const MOTION_MAP: Record<string, { amp: number; spd: number }> = {
  subtle:  { amp: 0.5,  spd: 0.62 },
  bold:    { amp: 1,    spd: 1    },
  maximal: { amp: 1.7,  spd: 1.6  },
}

// Non-reactive amp/spd for hot path (no reactive overhead in rAF)
let _amp = 1
let _spd = 1
watch(() => props.motion, (m) => {
  const s = MOTION_MAP[m]
  _amp = s.amp
  _spd = s.spd
}, { immediate: true })

// ── Canvas ref & setup ─────────────────────────────────────
const canvasEl = ref<HTMLCanvasElement | null>(null)
let ctx: CanvasRenderingContext2D | null = null
let w = 0
let h = 0
let dpr = 1
let reducedMotion = false
let hidden = false
let ro: ResizeObserver | null = null

// ── Palette (read once at mount) ──────────────────────────
let accentColor = '#7d4dff'
let accentRgb = '125,77,255'

function readPalette() {
  const style = getComputedStyle(document.documentElement)
  const a = style.getPropertyValue('--accent').trim()
  if (a) {
    accentColor = a
    // Extract RGB from hex
    const hex = a.replace('#', '')
    if (hex.length === 6) {
      accentRgb = `${parseInt(hex.substring(0, 2), 16)},${parseInt(hex.substring(2, 4), 16)},${parseInt(hex.substring(4, 6), 16)}`
    }
  }
}

// ── System state ──────────────────────────────────────────
let orbits: OrbitDef[] = []
let stars: Star[] = []
let comet: Comet | null = null
let cometT = 0
let time = 0
let running = false
let rafId: number | null = null
let boost = 1
let zoom = 1

// ── Parallax ──────────────────────────────────────────────
const par = { x: 0, y: 0, tx: 0, ty: 0 }

// ── Warp ──────────────────────────────────────────────────
const warpState: WarpState = { active: false, start: 0, onDone: null, flashed: false, flashTime: 0 }

// ── Perf tracking ────────────────────────────────────────
let frameTimes: number[] = []
let currentTier: PerfTier = 'high'
let lastTs = 0

// ── Canvas utility ───────────────────────────────────────
function fitCanvas() {
  const canvas = canvasEl.value
  if (!canvas) return
  const maxDpr = currentTier === 'high' ? 2 : currentTier === 'med' ? 1.5 : 1
  dpr = Math.min(window.devicePixelRatio || 1, maxDpr)
  w = canvas.clientWidth
  h = canvas.clientHeight
  canvas.width = w * dpr
  canvas.height = h * dpr
  ctx!.setTransform(dpr, 0, 0, dpr, 0, 0)
}

// ── Build system (orbits + stars) ────────────────────────
function build() {
  const defs: Array<{
    r: number; sp: number; tilt: number
    items: Array<[string, number] | [string, number, boolean]>
  }> = [
    { r: 120, sp: 0.50, tilt: 0.30, items: [['ice', 6], ['pale', 5], ['blue', 6]] },
    { r: 204, sp: 0.34, tilt: 0.54, items: [['blue', 7], ['rocky', 8], ['ice', 6], ['mars', 6]] },
    { r: 292, sp: 0.21, tilt: 0.38, items: [['violet', 15, true], ['blue', 7], ['ice', 6], ['pale', 5], ['mars', 7]] },
    { r: 378, sp: 0.14, tilt: 0.62, items: [['blue', 9], ['ice', 7], ['violet', 9]] },
  ]

  orbits = defs.map((o) => ({
    r: o.r,
    sp: o.sp,
    tilt: o.tilt,
    nodes: o.items.map((it, i) => {
      const ring = it.length > 2 && (it[2] as boolean)
      const np = ring ? 2 : (o.r < 210 ? 2 : 1)
      return {
        type: it[0],
        size: it[1],
        ring,
        ang: Math.random() * Math.PI * 2 + i * 1.4,
        moon: ring ? { ang: Math.random() * 6 } : null,
        pulses: Array.from({ length: np }, () => ({ t: Math.random(), spd: 0.7 + Math.random() * 0.9 })),
      } as OrbitNode
    }),
  }))

  // Stars
  const starCount = Math.round(w * h / 5200 * (0.7 + _amp * 0.4))
  const tierMul = currentTier === 'high' ? 1 : currentTier === 'med' ? 0.5 : 0.25
  const n = Math.round(starCount * tierMul)

  stars = Array.from({ length: n }, () => {
    const z = Math.random()
    return {
      x: Math.random() * w,
      y: Math.random() * h,
      z,
      r: z * 1.5 + 0.3,
      par: 4 + z * 16,
      tws: 0.5 + Math.random() * 2,
      tp: Math.random() * 6,
    }
  })

  comet = null
  cometT = 0
}

// ═══════════════════════════════════════════════════════════
//  DRAWING FUNCTIONS
// ═══════════════════════════════════════════════════════════

function drawPlanet(x: number, y: number, r: number, nd: OrbitNode, cx: number, cy: number) {
  const T = TYPES[nd.type]

  // Direction from planet center toward core (lit side)
  let lx = cx - x
  let ly = cy - y
  const L = Math.hypot(lx, ly) || 1
  lx /= L
  ly /= L
  const hx = x + lx * r * 0.5
  const hy = y + ly * r * 0.5

  // Atmosphere halo
  const atmo = ctx!.createRadialGradient(x, y, r * 0.55, x, y, r * 1.8)
  atmo.addColorStop(0, `rgba(${T.atmo},0.32)`)
  atmo.addColorStop(1, `rgba(${T.atmo},0)`)
  ctx!.fillStyle = atmo
  ctx!.beginPath()
  ctx!.arc(x, y, r * 1.8, 0, Math.PI * 2)
  ctx!.fill()

  // Planet body (lit toward core)
  const g = ctx!.createRadialGradient(hx, hy, r * 0.1, x, y, r)
  g.addColorStop(0, T.light)
  g.addColorStop(0.55, T.mid)
  g.addColorStop(1, T.dark)
  ctx!.fillStyle = g
  ctx!.beginPath()
  ctx!.arc(x, y, r, 0, Math.PI * 2)
  ctx!.fill()

  // Ringed gas giant banding
  if (nd.ring) {
    ctx!.save()
    ctx!.beginPath()
    ctx!.arc(x, y, r, 0, Math.PI * 2)
    ctx!.clip()
    for (let i = -3; i <= 3; i++) {
      ctx!.globalAlpha = 0.10
      ctx!.fillStyle = i % 2 ? T.light : T.dark
      ctx!.beginPath()
      ctx!.ellipse(x, y + i * r * 0.27, r * 1.1, r * 0.11, 0, 0, Math.PI * 2)
      ctx!.fill()
    }
    ctx!.globalAlpha = 1
    ctx!.restore()
  }

  // Specular highlight on lit side
  const sp = ctx!.createRadialGradient(hx, hy, 0, hx, hy, r * 0.55)
  sp.addColorStop(0, 'rgba(255,255,255,0.45)')
  sp.addColorStop(1, 'rgba(255,255,255,0)')
  ctx!.fillStyle = sp
  ctx!.beginPath()
  ctx!.arc(hx, hy, r * 0.55, 0, Math.PI * 2)
  ctx!.fill()

  // Atmospheric rim stroke
  ctx!.strokeStyle = `rgba(${T.atmo},0.4)`
  ctx!.lineWidth = 1
  ctx!.beginPath()
  ctx!.arc(x, y, r, 0, Math.PI * 2)
  ctx!.stroke()
}

function ringArc(x: number, y: number, r: number, a0: number, a1: number, alpha: number, tilt: number) {
  ctx!.save()
  ctx!.translate(x, y)
  ctx!.scale(1, tilt)
  ctx!.strokeStyle = `rgba(224,206,255,${alpha})`
  ctx!.lineWidth = r * 0.42
  ctx!.beginPath()
  ctx!.arc(0, 0, r * 1.85, a0, a1)
  ctx!.stroke()
  ctx!.restore()
}

function drawCore(cx: number, cy: number) {
  const flick = (Math.sin(time * 2) * 0.5 + 0.5) * 0.7 + (Math.sin(time * 5.3) * 0.5 + 0.5) * 0.3
  const R = 82 + flick * 26 * _amp

  // 4 rotating flares
  ctx!.save()
  ctx!.translate(cx, cy)
  ctx!.rotate(time * 0.15)
  for (let i = 0; i < 4; i++) {
    ctx!.rotate(Math.PI / 2)
    const fg = ctx!.createLinearGradient(0, 0, R * 2.3, 0)
    fg.addColorStop(0, `rgba(${accentRgb},${(0.16 * _amp).toFixed(3)})`)
    fg.addColorStop(1, `rgba(${accentRgb},0)`)
    ctx!.fillStyle = fg
    ctx!.beginPath()
    ctx!.moveTo(0, 0)
    ctx!.lineTo(R * 2.3, -R * 0.15)
    ctx!.lineTo(R * 2.3, R * 0.15)
    ctx!.closePath()
    ctx!.fill()
  }
  ctx!.restore()

  // Radial glow: white → purple → transparent
  const cg = ctx!.createRadialGradient(cx, cy, 0, cx, cy, R)
  cg.addColorStop(0, 'rgba(234,228,255,.95)')
  cg.addColorStop(0.22, `rgba(${accentRgb},.72)`)
  cg.addColorStop(0.55, `rgba(${accentRgb},.22)`)
  cg.addColorStop(1, `rgba(${accentRgb},0)`)
  ctx!.fillStyle = cg
  ctx!.beginPath()
  ctx!.arc(cx, cy, R, 0, Math.PI * 2)
  ctx!.fill()

  // Accretion ring (squashed ellipse)
  ctx!.save()
  ctx!.translate(cx, cy)
  ctx!.scale(1, 0.32)
  ctx!.strokeStyle = `rgba(190,170,255,${(0.22 * _amp).toFixed(3)})`
  ctx!.lineWidth = 2
  ctx!.beginPath()
  ctx!.arc(0, 0, 30, 0, Math.PI * 2)
  ctx!.stroke()
  ctx!.restore()

  // Center hot dot
  ctx!.fillStyle = '#f6f2ff'
  ctx!.beginPath()
  ctx!.arc(cx, cy, 9, 0, Math.PI * 2)
  ctx!.fill()
}

function drawComet() {
  cometT += _spd

  if (!comet && cometT > 360 && Math.random() < 0.012) {
    comet = {
      x: -60,
      y: Math.random() * h * 0.55,
      vx: 2.6 + Math.random() * 2,
      vy: 0.5 + Math.random() * 0.9,
    }
    cometT = 0
  }

  if (comet) {
    comet.x += comet.vx * _spd * 2
    comet.y += comet.vy * _spd * 2

    // Gradient tail
    const tx = comet.x - comet.vx * 18
    const ty = comet.y - comet.vy * 18
    const g = ctx!.createLinearGradient(comet.x, comet.y, tx, ty)
    g.addColorStop(0, 'rgba(205,245,255,.9)')
    g.addColorStop(1, 'rgba(205,245,255,0)')
    ctx!.strokeStyle = g
    ctx!.lineWidth = 2
    ctx!.beginPath()
    ctx!.moveTo(comet.x, comet.y)
    ctx!.lineTo(tx, ty)
    ctx!.stroke()

    // Comet head
    ctx!.fillStyle = '#eafaff'
    ctx!.beginPath()
    ctx!.arc(comet.x, comet.y, 2, 0, Math.PI * 2)
    ctx!.fill()

    // Remove when off-screen
    if (comet.x > w + 80 || comet.y > h + 80) comet = null
  }
}

function drawNode(it: NodeRender) {
  const { o, nd, x, y, depth } = it
  const front = (depth + 1) / 2
  const r = nd.size * (0.82 + 0.18 * front)

  ctx!.globalAlpha = 0.55 + 0.45 * front

  // Back half of ring (behind planet)
  if (nd.ring) ringArc(x, y, r, Math.PI, Math.PI * 2, 0.45 * _amp, o.tilt)

  // Planet body
  drawPlanet(x, y, r, nd, w / 2, h / 2)

  // Front half of ring + moon
  if (nd.ring) {
    ringArc(x, y, r, 0, Math.PI, 0.7 * _amp, o.tilt)
    if (nd.moon) {
      nd.moon.ang += 0.045 * _spd
      const mx = x + Math.cos(nd.moon.ang) * r * 2.7
      const my = y + Math.sin(nd.moon.ang) * r * 2.7 * o.tilt
      ctx!.fillStyle = '#d2def0'
      ctx!.beginPath()
      ctx!.arc(mx, my, 2.3, 0, Math.PI * 2)
      ctx!.fill()
    }
  }

  ctx!.globalAlpha = 1
}

// ═══════════════════════════════════════════════════════════
//  MAIN ANIMATION LOOP
// ═══════════════════════════════════════════════════════════

function renderStaticFrame() {
  // Render a single rest-pose frame without animation
  if (!ctx || w === 0 || h === 0) return

  const cx = w / 2
  const cy = h / 2
  const a = _amp

  ctx.clearRect(0, 0, w, h)

  // Stars (static, no twinkle)
  for (const st of stars) {
    ctx.globalAlpha = 0.25 + 0.6 * st.z
    ctx.fillStyle = st.z > 0.7 ? '#dfeefc' : '#9fb2cc'
    ctx.beginPath()
    ctx.arc(st.x, st.y, st.r, 0, Math.PI * 2)
    ctx.fill()
  }
  ctx.globalAlpha = 1

  // Orbit ellipses
  for (const o of orbits) {
    ctx.strokeStyle = `rgba(125,77,255,${0.10 * a})`
    ctx.lineWidth = 1
    ctx.beginPath()
    ctx.ellipse(cx, cy, o.r, o.r * o.tilt, 0, 0, Math.PI * 2)
    ctx.stroke()
  }

  // Core
  drawCore(cx, cy)
}

function frame(ts: number) {
  if (!running || !ctx || hidden) {
    if (running && !hidden) {
      rafId = requestAnimationFrame(frame)
    } else {
      rafId = null
    }
    return
  }

  // ── Perf tracking ──
  if (lastTs > 0 && ts - lastTs > 100) {
    // Resume from pause — skip this frame's dt
    lastTs = ts
    rafId = requestAnimationFrame(frame)
    return
  }
  if (lastTs > 0) {
    const dt = ts - lastTs
    frameTimes.push(dt)
    if (frameTimes.length > 30) frameTimes.shift()

    if (frameTimes.length >= 30) {
      const avg = frameTimes.reduce((a, b) => a + b, 0) / frameTimes.length
      const newTier = perfTier(avg, currentTier)
      if (newTier !== currentTier) {
        currentTier = newTier
        if (currentTier === 'low') {
          // Degraded to low — render final static frame, stop loop
          renderStaticFrame()
          running = false
          rafId = null
          return
        }
        // Re-fit with new DPR cap and rebuild
        fitCanvas()
        build()
      }
    }
  }
  lastTs = ts

  // ── Frame vars ──
  const a = _amp
  const s = _spd
  const cx = w / 2
  const cy = h / 2

  // Advance time (fixed-step, frame-rate dependent per reference)
  time += 0.01 * s

  // Parallax lerp
  par.x += (par.tx - par.x) * 0.06
  par.y += (par.ty - par.y) * 0.06

  // ── Warp ──
  if (warpState.active) {
    const elapsed = (performance.now() - warpState.start) / 1000
    const wp = warpProgress(elapsed)
    boost = wp.boost
    zoom = wp.zoom
    if (wp.flash && !warpState.flashed) {
      warpState.flashed = true
      warpState.flashTime = performance.now()
    }
    if (wp.done) {
      warpState.active = false
      boost = 1
      zoom = 1
      const doneCb = warpState.onDone
      warpState.onDone = null
      doneCb?.()
    }
  }

  // ── Clear ──
  ctx.clearRect(0, 0, w, h)

  // ── Stars (parallax + twinkle + warp streaks) ──
  const streak = warpState.active && boost > 1.3
  for (const st of stars) {
    const tw = 0.5 + 0.5 * Math.sin(time * st.tws + st.tp)
    const px = st.x - par.x * st.par
    const py = st.y - par.y * st.par

    if (streak) {
      const dx = px - cx
      const dy = py - cy
      const len = Math.min(1, (boost - 1) / 11)
      ctx.strokeStyle = `rgba(205,235,255,${(0.55 * tw).toFixed(3)})`
      ctx.lineWidth = st.r
      ctx.beginPath()
      ctx.moveTo(px, py)
      ctx.lineTo(px + dx * len * 1.5, py + dy * len * 1.5)
      ctx.stroke()
    } else {
      ctx.globalAlpha = (0.25 + 0.6 * st.z) * tw
      ctx.fillStyle = st.z > 0.7 ? '#dfeefc' : '#9fb2cc'
      ctx.beginPath()
      ctx.arc(px, py, st.r, 0, Math.PI * 2)
      ctx.fill()
    }
  }
  ctx.globalAlpha = 1

  // ── Comet (high tier only) ──
  if (currentTier === 'high') drawComet()

  // ── System group ──
  ctx.save()

  // Parallax shift on system
  ctx.translate(par.x * 16, par.y * 16)

  // Warp zoom about center
  ctx.translate(cx, cy)
  ctx.scale(zoom, zoom)
  ctx.translate(-cx, -cy)

  // Orbit ellipses
  for (const o of orbits) {
    ctx.strokeStyle = `rgba(125,77,255,${(0.10 * a).toFixed(3)})`
    ctx.lineWidth = 1
    ctx.beginPath()
    ctx.ellipse(cx, cy, o.r, o.r * o.tilt, 0, 0, Math.PI * 2)
    ctx.stroke()
  }

  // Build render list with depth
  const list: NodeRender[] = []
  for (const o of orbits) {
    for (const nd of o.nodes) {
      nd.ang += o.sp * 0.01 * s * boost
      const x = cx + Math.cos(nd.ang) * o.r
      const y = cy + Math.sin(nd.ang) * o.r * o.tilt
      const depth = Math.sin(nd.ang)
      list.push({ o, nd, x, y, depth })
    }
  }

  // Faint static links: core → planets
  for (const it of list) {
    ctx.strokeStyle = `rgba(125,77,255,${(0.06 * a).toFixed(3)})`
    ctx.lineWidth = 1
    ctx.beginPath()
    ctx.moveTo(cx, cy)
    ctx.lineTo(it.x, it.y)
    ctx.stroke()
  }

  // Depth sort: back (negative depth) → core → front (positive depth)
  const back = list.filter(p => p.depth < 0).sort((a, b) => a.depth - b.depth)
  const front = list.filter(p => p.depth >= 0).sort((a, b) => a.depth - b.depth)

  for (const it of back) drawNode(it)
  drawCore(cx, cy)
  for (const it of front) drawNode(it)

  // Data pulses (core → planet along link path)
  for (const it of list) {
    const { nd, x, y } = it
    for (const ph of nd.pulses) {
      ph.t += ph.spd * 0.004 * s
      if (ph.t > 1) ph.t -= 1 + Math.random() * 0.35
      if (ph.t < 0) continue
      const px = cx + (x - cx) * ph.t
      const py = cy + (y - cy) * ph.t
      const fade = Math.sin(ph.t * Math.PI)
      ctx.fillStyle = `rgba(196,170,255,${(0.9 * fade * a).toFixed(3)})`
      ctx.shadowBlur = 8
      ctx.shadowColor = `rgba(${accentRgb},.9)`
      ctx.beginPath()
      ctx.arc(px, py, 2, 0, Math.PI * 2)
      ctx.fill()
      ctx.shadowBlur = 0
    }
  }

  ctx.restore()

  // ── Warp flash overlay ──
  if (warpState.flashed && warpState.active) {
    const flashAge = (performance.now() - warpState.flashTime) / 1000
    const flashAlpha = Math.max(0, 0.5 * (1 - Math.min(flashAge / 0.15, 1)))
    ctx.fillStyle = `rgba(244,240,255,${flashAlpha.toFixed(3)})`
    ctx.fillRect(0, 0, w, h)
  }

  // ── Next frame ──
  rafId = requestAnimationFrame(frame)
}

// ═══════════════════════════════════════════════════════════
//  WARP (exposed)
// ═══════════════════════════════════════════════════════════

function warp(): Promise<void> {
  if (reducedMotion || currentTier === 'low') {
    // ponytail: reduced-motion/low-tier warp = quick fade, no animation
    return new Promise(resolve => setTimeout(resolve, 200))
  }
  return new Promise((resolve) => {
    if (warpState.active) {
      resolve()
      return
    }
    warpState.active = true
    warpState.start = performance.now()
    warpState.onDone = () => {
      resolve()
    }
    warpState.flashed = false
  })
}

// ═══════════════════════════════════════════════════════════
//  EVENT HANDLERS
// ═══════════════════════════════════════════════════════════

function onPointerMove(e: PointerEvent) {
  par.tx = Math.max(-1, Math.min(1, (e.clientX / window.innerWidth - 0.5) * 2))
  par.ty = Math.max(-1, Math.min(1, (e.clientY / window.innerHeight - 0.5) * 2))
}

function onDeviceOrientation(e: DeviceOrientationEvent) {
  if (e.gamma == null) return
  par.tx = Math.max(-1, Math.min(1, e.gamma / 28))
  par.ty = Math.max(-1, Math.min(1, (e.beta! - 45) / 28))
}

function onVisibilityChange() {
  hidden = document.hidden
  if (!hidden && running) {
    lastTs = 0
    if (rafId === null) {
      rafId = requestAnimationFrame(frame)
    }
  }
}

function onResize() {
  fitCanvas()
  if (running || reducedMotion || currentTier === 'low') {
    build()
    if (!running) renderStaticFrame()
  }
}

// ═══════════════════════════════════════════════════════════
//  LIFECYCLE
// ═══════════════════════════════════════════════════════════

onMounted(() => {
  // Reset module-level state from any previous mount
  currentTier = 'high'
  frameTimes = []
  lastTs = 0
  running = false
  rafId = null

  const canvas = canvasEl.value
  if (!canvas) return

  ctx = canvas.getContext('2d')
  if (!ctx) return

  readPalette()
  reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches

  fitCanvas()
  build()

  if (reducedMotion || currentTier === 'low') {
    // Static frame only — no animation loop
    renderStaticFrame()
    // Watch for resize to re-render static frame
    ro = new ResizeObserver(() => {
      fitCanvas()
      build()
      renderStaticFrame()
    })
    ro.observe(canvas)
    return
  }

  // Start animation loop
  running = true
  lastTs = 0
  frameTimes = []
  rafId = requestAnimationFrame(frame)

  // ResizeObserver for dynamic resize
  ro = new ResizeObserver(onResize)
  ro.observe(canvas)

  // Listeners
  if (!reducedMotion && currentTier !== 'low') {
    window.addEventListener('pointermove', onPointerMove, { passive: true })
    window.addEventListener('deviceorientation', onDeviceOrientation)
  }
  document.addEventListener('visibilitychange', onVisibilityChange)
})

onBeforeUnmount(() => {
  running = false
  if (rafId !== null) {
    cancelAnimationFrame(rafId)
    rafId = null
  }
  // Reset perf state so next mount starts fresh
  currentTier = 'high'
  frameTimes = []
  lastTs = 0
  if (ro) {
    ro.disconnect()
    ro = null
  }
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('deviceorientation', onDeviceOrientation)
  document.removeEventListener('visibilitychange', onVisibilityChange)
})

defineExpose({ warp })
</script>

<style scoped>
.orbit-canvas {
  position: fixed;
  inset: 0;
  z-index: 0;
  display: block;
  pointer-events: none;
}
</style>
