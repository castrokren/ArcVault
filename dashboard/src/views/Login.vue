<template>
  <div class="login-page">
    <!-- Background: OrbitField canvas (fixed, z-index:0) -->
    <OrbitField ref="orbitField" motion="bold" />

    <!-- Vignette overlay (z-index: 0) -->
    <div class="login-vignette"></div>

    <!-- Aurora gradient blobs (z-index: 0, decorative) -->
    <div class="login-aurora login-aurora-1" v-if="!prefersReducedMotion"></div>
    <div class="login-aurora login-aurora-2" v-if="!prefersReducedMotion"></div>
    <div class="login-aurora login-aurora-3" v-if="!prefersReducedMotion"></div>
    <div class="login-aurora login-aurora-4" v-if="!prefersReducedMotion"></div>

    <!-- Shield watermark (z-index: 0, decorative) -->
    <div class="login-watermark" v-if="!prefersReducedMotion">
      <svg viewBox="0 0 560 560" width="560" height="560" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path
          d="M280 70L490 175V280C490 392 406 476 280 504C154 476 70 392 70 280V175L280 70Z"
          stroke="currentColor"
          stroke-width="2"
          opacity="0.05"
        />
        <path
          d="M210 280L252 322L350 238"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          opacity="0.05"
        />
      </svg>
    </div>

    <!-- Starfield (z-index: 0, decorative) -->
    <div class="login-stars" v-if="!prefersReducedMotion">
      <div
        v-for="(star, i) in stars"
        :key="i"
        :class="['login-star', { lg: star.large }]"
        :style="{
          left: `${star.left}%`,
          top: `${star.top}%`,
          animation: `avTwinkle ${star.duration}s ease-in-out ${star.delay}s infinite`
        }"
      ></div>
    </div>

    <!-- Data-comet beams (z-index: 0, decorative) -->
    <div class="login-beams" v-if="!prefersReducedMotion">
      <div
        v-for="(beam, i) in beams"
        :key="i"
        class="login-beam-wrap"
        :style="{
          left: logoCenter.left,
          top: logoCenter.top,
          transform: `rotate(${beam.angle}deg)`
        }"
      >
        <div :class="['login-beam', beam.width]"></div>
      </div>
    </div>

    <!-- Centered glass card layer (z-index:1) -->
    <div class="login-shell" :class="{ warping: isWarping }" :style="shellStyle">
      <!-- Brand — stagger entrance 1 -->
      <motion.div
        :initial="{ opacity: 0, y: 20 }"
        :animate="{ opacity: 1, y: 0 }"
        :transition="{ type: 'spring', stiffness: 120, damping: 14 }"
      >
        <div class="brand">
          <div class="login-brand-icon-wrap">
            <div class="login-halo-outer" v-if="!prefersReducedMotion"></div>
            <div class="login-halo-inner" v-if="!prefersReducedMotion"></div>
            <div class="login-brand-icon">
              <svg class="mark" viewBox="0 0 48 48" fill="none">
                <defs>
                  <linearGradient id="brandGrad" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0" stop-color="#9d72ff"/>
                    <stop offset="1" stop-color="#5f33d6"/>
                  </linearGradient>
                </defs>
                <path d="M24 5 L41 14 V34 L24 43 L7 34 V14 Z" stroke="url(#brandGrad)" stroke-width="2" opacity=".55"/>
                <path d="M14 30 A12 12 0 0 1 34 30" stroke="url(#brandGrad)" stroke-width="2.6" stroke-linecap="round"/>
                <path d="M18 30 A6.5 6.5 0 0 1 30 30" stroke="url(#brandGrad)" stroke-width="2.6" stroke-linecap="round" opacity=".7"/>
                <circle cx="24" cy="30" r="2.4" fill="#9d72ff"/>
              </svg>
            </div>
          </div>
          <div class="brand-text">
            <span class="brand-name">Arc<b>Vault</b></span>
            <span class="brand-sub">Backup orchestrator</span>
          </div>
        </div>
      </motion.div>

      <!-- Card with error shake wrapper -->
      <motion.div
        :animate="cardAnim"
        :transition="{ type: 'spring', stiffness: 500, damping: 8 }"
      >
        <div class="login-card">
          <h1 class="card-title">Sign in</h1>
          <p class="card-sub">Backup orchestrator console</p>

          <div v-if="errorMsg" class="error">{{ errorMsg }}</div>

          <form @submit.prevent="handleSubmit" class="login-form">
            <!-- Username field — stagger entrance 2 -->
            <motion.div
              :initial="{ opacity: 0, y: 16 }"
              :animate="{ opacity: 1, y: 0 }"
              :transition="{ type: 'spring', stiffness: 100, damping: 20, delay: 0.08 }"
              class="field"
            >
              <label for="username">Username</label>
              <div class="input-wrap">
                <svg
                  class="input-icon" width="14" height="14" viewBox="0 0 14 14" fill="none"
                  aria-hidden="true"
                >
                  <circle cx="7" cy="5" r="2.5" stroke="currentColor" stroke-width="1.2" />
                  <path d="M2 12c0-2.76 2.24-5 5-5s5 2.24 5 5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
                </svg>
                <input
                  id="username"
                  v-model="username"
                  type="text"
                  autocomplete="username"
                  required
                  placeholder="Username"
                />
              </div>
            </motion.div>

            <!-- Password field — stagger entrance 3 -->
            <motion.div
              :initial="{ opacity: 0, y: 16 }"
              :animate="{ opacity: 1, y: 0 }"
              :transition="{ type: 'spring', stiffness: 100, damping: 20, delay: 0.14 }"
              class="field"
            >
              <label for="password">Password</label>
              <div class="input-wrap">
                <svg
                  class="input-icon" width="14" height="14" viewBox="0 0 14 14" fill="none"
                  aria-hidden="true"
                >
                  <rect x="3" y="6" width="8" height="6" rx="1" stroke="currentColor" stroke-width="1.2" />
                  <path d="M5 6V4.5a2 2 0 0 1 4 0V6" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
                  <circle cx="7" cy="9" r="0.75" fill="currentColor" />
                </svg>
                <input
                  id="password"
                  v-model="password"
                  type="password"
                  autocomplete="current-password"
                  required
                  placeholder="Password"
                />
              </div>
            </motion.div>

            <!-- Remember me — stagger entrance 4 -->
            <motion.div
              :initial="{ opacity: 0, y: 16 }"
              :animate="{ opacity: 1, y: 0 }"
              :transition="{ type: 'spring', stiffness: 100, damping: 20, delay: 0.20 }"
            >
              <label class="remember-me">
                <input type="checkbox" v-model="remember" />
                <span class="check-label">Remember me</span>
              </label>
            </motion.div>

            <!-- Submit button — stagger entrance 5 + gestures -->
            <motion.div
              :initial="{ opacity: 0, y: 16 }"
              :animate="{ opacity: 1, y: 0 }"
              :transition="{ type: 'spring', stiffness: 100, damping: 20, delay: 0.26 }"
            >
              <motion.button
                type="submit"
                class="btn btn-primary submit-btn"
                :disabled="isSubmitting"
                :whileHover="{ scale: 1.02 }"
                :whilePress="{ scale: 0.97 }"
                :transition="{ type: 'spring', stiffness: 400, damping: 10 }"
              >
                {{ isSubmitting ? 'Connecting\u2026' : 'Sign in' }}
              </motion.button>
            </motion.div>
          </form>
        </div>
      </motion.div>

      <p class="login-footer">ArcVault — Distributed backup orchestration</p>
    </div>

  <ChangePasswordModal
    :isOpen="showChangePw"
    @close="showChangePw = false"
    @success="showChangePw = false; router.push('/')"
  />
</div>
</template>

<script setup>
import { ref, computed, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { motion } from 'motion-v'
import { useAuth } from '../composables/useAuth.js'
import OrbitField from '../components/orbit/OrbitField.vue'
import ChangePasswordModal from '../components/ChangePasswordModal.vue'

const router = useRouter()
const auth = useAuth()

const username = ref('')
const password = ref('')
const remember = ref(true)
const isSubmitting = ref(false)
const errorMsg = ref(null)
const showChangePw = ref(false)

const orbitField = ref(null)

const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches

// Starfield data — 25 stars with staggered twinkle
const stars = [
  { left: 5, top: 8, delay: 0.1, duration: 3.2, large: false },
  { left: 15, top: 20, delay: 0.8, duration: 2.8, large: true },
  { left: 25, top: 5, delay: 1.5, duration: 4.0, large: false },
  { left: 35, top: 15, delay: 2.2, duration: 2.5, large: false },
  { left: 45, top: 30, delay: 0.3, duration: 3.5, large: true },
  { left: 55, top: 10, delay: 1.9, duration: 3.0, large: false },
  { left: 65, top: 25, delay: 0.6, duration: 2.7, large: false },
  { left: 75, top: 3, delay: 2.8, duration: 3.8, large: true },
  { left: 85, top: 18, delay: 1.2, duration: 2.9, large: false },
  { left: 92, top: 35, delay: 0.9, duration: 3.3, large: false },
  { left: 10, top: 45, delay: 1.7, duration: 2.6, large: false },
  { left: 20, top: 60, delay: 0.4, duration: 3.7, large: true },
  { left: 30, top: 50, delay: 2.5, duration: 2.8, large: false },
  { left: 40, top: 70, delay: 0.7, duration: 4.1, large: false },
  { left: 50, top: 55, delay: 1.1, duration: 3.1, large: true },
  { left: 60, top: 75, delay: 2.0, duration: 2.4, large: false },
  { left: 70, top: 65, delay: 0.2, duration: 3.6, large: false },
  { left: 80, top: 80, delay: 1.8, duration: 2.9, large: true },
  { left: 90, top: 58, delay: 2.3, duration: 3.4, large: false },
  { left: 3, top: 72, delay: 1.4, duration: 3.0, large: false },
  { left: 48, top: 42, delay: 0.5, duration: 2.5, large: false },
  { left: 72, top: 48, delay: 2.7, duration: 3.9, large: true },
  { left: 18, top: 35, delay: 1.6, duration: 2.7, large: false },
  { left: 82, top: 42, delay: 0.0, duration: 3.2, large: false },
  { left: 38, top: 85, delay: 2.1, duration: 2.8, large: true },
]

// Data-comet beams — 12 beams converging on brand icon from all angles
const beams = [
  { angle: 0, width: '' },
  { angle: 30, width: 'thin' },
  { angle: 60, width: '' },
  { angle: 90, width: 'thick' },
  { angle: 120, width: '' },
  { angle: 150, width: 'thin' },
  { angle: 180, width: '' },
  { angle: 210, width: 'thick' },
  { angle: 240, width: '' },
  { angle: 270, width: 'thin' },
  { angle: 300, width: '' },
  { angle: 330, width: 'thick' },
]

// ponytail: exact logo center needs browser verification — adjust these if beams misalign
const logoCenter = { left: '50%', top: '26%' }

// Parallax pointer tracking — card shifts slightly on mouse move
const parallax = reactive({ x: 0, y: 0 })

if (!prefersReducedMotion) {
  const updateParallax = (clientX, clientY) => {
    parallax.x = (clientX / window.innerWidth - 0.5) * 2
    parallax.y = (clientY / window.innerHeight - 0.5) * 2
  }
  window.addEventListener('pointermove', (e) => updateParallax(e.clientX, e.clientY))
  window.addEventListener('deviceorientation', (e) => {
    if (e.gamma == null) return
    parallax.x = Math.max(-1, Math.min(1, e.gamma / 28))
    parallax.y = Math.max(-1, Math.min(1, (e.beta - 45) / 28))
  })
}

// Card transform for parallax shift
const shellStyle = computed(() => {
  if (prefersReducedMotion) return {}
  return {
    transform: `translate3d(${parallax.x * -15}px, ${parallax.y * -15}px, 0)`
  }
})

const isWarping = ref(false)

// ponytail: shake anim on error — keyframes replay if errorMsg transitions null→string
const cardAnim = computed(() => ({
  x: errorMsg.value ? [0, -8, 8, -6, 6, 0] : 0,
}))

async function handleSubmit() {
  if (isSubmitting.value) return                    // double-submit guard
  isSubmitting.value = true
  errorMsg.value = ''

  const result = await auth.login(
    username.value,
    password.value,
    remember.value,
  )

  if (!result.success) {
    errorMsg.value = result.error
    isSubmitting.value = false
    return                                          // shake fires automatically via cardAnim
  }

  if (result.mustChangePassword) {
    showChangePw.value = true
    isSubmitting.value = false
    return                                          // skip warp, open modal
  }

  // Success path — minimal fade+scale then redirect
  // ponytail: full warp freezes browser; use lightweight fade+scale (css-only, no canvas)
  isWarping.value = true
  await new Promise(r => setTimeout(r, 300))        // ~300ms lightweight transition
  isWarping.value = false
  router.push('/')
  isSubmitting.value = false
}
</script>

<style scoped>
/* ── Page shell ──────────────────────────────────────────── */
.login-page {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: radial-gradient(120% 90% at 50% 8%, #0a1322 0%, #05070d 55%, #03040a 100%);
  overflow: hidden;
  padding: 2rem;
}

/* ── Content layer ──────────────────────────────────────── */
.login-shell {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1.5rem;
  width: 100%;
  max-width: 380px;
  transition: transform 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.login-shell.warping {
  /* ponytail: minimal fade+scale animation (~300ms), no expensive canvas effects */
  transform: scale(0.95) !important;
  opacity: 0;
  transition: transform 0.3s ease, opacity 0.3s ease;
}

/* ── Vignette overlay ───────────────────────────────────── */
.login-vignette {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  background: radial-gradient(circle at 50% 42%, transparent 36%, rgba(3,4,10,.82) 100%);
}

/* ── Brand ───────────────────────────────────────────────── */
.brand {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.brand-icon svg {
  width: 20px;
  height: 20px;
}

.brand-name {
  font-family: var(--font-display);
  font-weight: 800;
  font-size: 1.35rem;
  letter-spacing: 0.04em;
  color: var(--text-primary);
}

.mark {
  width: 42px;
  height: 42px;
  flex: none;
  filter: drop-shadow(0 0 16px var(--accent-dim));
}

.brand-name b {
  color: var(--accent);
}

.brand-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.brand-sub {
  display: block;
  font-size: 10px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-weight: 500;
}

/* ── Glass card ──────────────────────────────────────────── */
.login-card {
  position: relative;
  width: 100%;
  /* ponytail: reduced blur (8px) for perf, solid bg fallback for glass appearance */
  background: rgba(20, 24, 48, 0.92);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-card);
  padding: 1.75rem;
  box-shadow: var(--shadow-lg), var(--edge-highlight), 0 0 40px var(--accent-dim), 0 50px 130px -30px rgba(0,0,0,.9);
}

.login-card::before {
  content: "";
  position: absolute;
  inset: 0;
  border-radius: var(--radius-card);
  padding: 1px;
  background: linear-gradient(155deg, rgba(125,77,255,.55), rgba(139,92,246,.25) 35%, transparent 65%);
   mask: linear-gradient(#000 0 0) content-box, linear-gradient(#000 0 0);
   -webkit-mask: linear-gradient(#000 0 0) content-box, linear-gradient(#000 0 0);
   -webkit-mask-composite: xor;
  mask-composite: exclude;
  pointer-events: none;
}

.card-title {
  font-family: var(--font-display);
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.01em;
  margin-bottom: 0.3rem;
}

.card-sub {
  font-family: var(--font-body);
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin: 0 0 0.5rem;
}

/* ── Form ────────────────────────────────────────────────── */
.login-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  margin-top: 1rem;
}

.field {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.field label {
  font-family: var(--font-body);
  font-size: 0.82rem;
  font-weight: 500;
  color: var(--text-secondary);
}

.field::after {
  content: "";
  position: absolute;
  left: 15px;
  right: 15px;
  bottom: 1px;
  height: 2px;
  border-radius: 2px;
  background: linear-gradient(90deg, var(--accent), transparent);
  transform: scaleX(0);
  transform-origin: left;
  transition: transform 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
  opacity: 0;
}

.field:focus-within::after {
  transform: scaleX(1);
  opacity: 0.9;
}

.input-wrap {
  position: relative;
}

.input-icon {
  position: absolute;
  left: 0.7rem;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-muted);
  pointer-events: none;
}

.input-wrap input {
  width: 100%;
  padding: 0.6rem 0.85rem 0.6rem 2.2rem;
  border-radius: var(--radius-ctrl);
  border: 1px solid var(--border-default);
  background: var(--bg-input);
  color: var(--text-primary);
  font-family: var(--font-body);
  font-size: 0.9rem;
  transition: border-color 0.15s, box-shadow 0.15s;
  box-sizing: border-box;
}

.input-wrap input::placeholder {
  color: var(--text-muted);
}

.input-wrap input:focus-visible {
  outline: none;
  border-color: var(--accent);
  box-shadow: var(--glow-accent);
}

.input-wrap input:focus:not(:focus-visible) {
  outline: none;
  border-color: var(--border-strong);
}

.input-wrap input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ── Remember me ─────────────────────────────────────────── */
.remember-me {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  font-family: var(--font-body);
  font-size: 0.85rem;
  color: var(--text-secondary);
  user-select: none;
}

.remember-me input[type="checkbox"] {
  width: 15px;
  height: 15px;
  accent-color: var(--accent);
  cursor: pointer;
}

.check-label {
  color: var(--text-secondary);
  font-family: var(--font-body);
  font-size: 0.85rem;
}

/* ── Submit button ───────────────────────────────────────── */
.submit-btn {
  position: relative;
  overflow: hidden;
  width: 100%;
  padding: 0.65rem 1rem;
  margin-top: 0.25rem;
  font-family: var(--font-display);
  font-size: 0.9rem;
  font-weight: 700;
  letter-spacing: 0.02em;
  cursor: pointer;
}

.submit-btn:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.submit-btn:disabled {
  cursor: not-allowed;
}

.submit-btn::after {
  content: "";
  position: absolute;
  top: 0;
  left: -60%;
  width: 40%;
  height: 100%;
  background: linear-gradient(100deg, transparent, rgba(255,255,255,.55), transparent);
  transform: skewX(-18deg);
  transition: left 0.6s ease;
}

.submit-btn:hover::after {
  left: 130%;
}

/* ── Footer ──────────────────────────────────────────────── */
.login-footer {
  font-family: var(--font-body);
  font-size: 0.75rem;
  color: var(--text-muted);
  text-align: center;
  margin: 0;
}
</style>
