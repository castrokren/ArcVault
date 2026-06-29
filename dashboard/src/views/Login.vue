<template>
  <div class="login-page">
    <!-- Background: OrbitField canvas (fixed, z-index:0) -->
    <OrbitField ref="orbitField" motion="bold" />

    <!-- Centered glass card layer (z-index:1) -->
    <div class="login-shell">
      <!-- Brand — stagger entrance 1 -->
      <motion.div
        :initial="{ opacity: 0, y: 20 }"
        :animate="{ opacity: 1, y: 0 }"
        :transition="{ type: 'spring', stiffness: 120, damping: 14 }"
      >
        <div class="brand">
          <div class="brand-icon">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M12 3L20 7.5V12C20 16.42 16.47 20.5 12 21.5C7.53 20.5 4 16.42 4 12V7.5L12 3Z"
                stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"
              />
              <path
                d="M8.5 12L11.5 15L15.5 9.5"
                stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"
              />
            </svg>
          </div>
          <span class="brand-name">ArcVault</span>
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

      <p class="login-footer">ArcVault \u2014 Distributed backup orchestration</p>
    </div>
  </div>

  <ChangePasswordModal
    :isOpen="showChangePw"
    @close="showChangePw = false"
    @success="showChangePw = false; router.push('/')"
  />
</template>

<script setup>
import { ref, computed } from 'vue'
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

  // Success path — warp (or fade) then redirect
  if (prefersReducedMotion) {
    await new Promise(r => setTimeout(r, 200))      // quick fade ~200ms, no warp
  } else {
    await orbitField.value?.warp()                  // await ~1.25s warp dive
  }
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
  background: var(--bg-base);
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
}

/* ── Brand ───────────────────────────────────────────────── */
.brand {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.brand-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: var(--accent-dim);
  border: 1px solid var(--accent-border);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--accent);
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

/* ── Glass card ──────────────────────────────────────────── */
.login-card {
  width: 100%;
  /* ponytail: translucent bg for glass effect over OrbitField canvas */
  background: color-mix(in srgb, var(--bg-card) 78%, transparent);
  backdrop-filter: blur(24px) saturate(1.25);
  -webkit-backdrop-filter: blur(24px) saturate(1.25);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-card);
  padding: 1.75rem;
  box-shadow: var(--shadow-lg), var(--edge-highlight), 0 0 40px var(--accent-dim);
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

/* ── Footer ──────────────────────────────────────────────── */
.login-footer {
  font-family: var(--font-body);
  font-size: 0.75rem;
  color: var(--text-muted);
  text-align: center;
  margin: 0;
}
</style>
