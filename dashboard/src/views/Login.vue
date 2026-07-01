<template>
  <div class="login-page">
    <div class="login-card">
      <h1 class="card-title">Sign in</h1>
      <p class="card-sub">Backup orchestrator console</p>

      <div v-if="errorMsg" class="error">{{ errorMsg }}</div>

      <form @submit.prevent="handleSubmit" class="login-form">
        <!-- Username field -->
        <div class="field">
          <label for="username">Username</label>
          <input
            id="username"
            v-model="username"
            type="text"
            autocomplete="username"
            required
            placeholder="Username"
          />
        </div>

        <!-- Password field -->
        <div class="field">
          <label for="password">Password</label>
          <input
            id="password"
            v-model="password"
            type="password"
            autocomplete="current-password"
            required
            placeholder="Password"
          />
        </div>

        <!-- Remember me -->
        <label class="remember-me">
          <input type="checkbox" v-model="remember" />
          <span class="check-label">Remember me</span>
        </label>

        <!-- Submit button -->
        <button
          type="submit"
          class="btn btn-primary submit-btn"
          :disabled="isSubmitting"
        >
          {{ isSubmitting ? 'Connecting…' : 'Sign in' }}
        </button>
      </form>
    </div>

    <!-- Decorative arc rings (respects prefers-reduced-motion) -->
    <div v-if="!prefersReducedMotion" class="arc-rings">
      <div class="arc arc-1"></div>
      <div class="arc arc-2"></div>
      <div class="arc arc-3"></div>
    </div>

    <ChangePasswordModal
      :isOpen="showChangePw"
      @close="showChangePw = false"
      @success="showChangePw = false; router.push('/')"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '../composables/useAuth.js'
import ChangePasswordModal from '../components/ChangePasswordModal.vue'

const router = useRouter()
const auth = useAuth()

const username = ref('')
const password = ref('')
const remember = ref(true)
const isSubmitting = ref(false)
const errorMsg = ref(null)
const showChangePw = ref(false)

// ponytail: matchMedia listener for reduced motion (drives v-if on arcs)
const prefersReducedMotion = ref(false)
let mql = null

onMounted(() => {
  mql = window.matchMedia('(prefers-reduced-motion: reduce)')
  prefersReducedMotion.value = mql.matches
  mql.addEventListener('change', onMotionChange)
})

onBeforeUnmount(() => {
  if (mql) mql.removeEventListener('change', onMotionChange)
})

function onMotionChange(e) {
  prefersReducedMotion.value = e.matches
}

async function handleSubmit() {
  if (isSubmitting.value) return
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
    return
  }

  if (result.mustChangePassword) {
    showChangePw.value = true
    isSubmitting.value = false
    return
  }

  // Success — redirect
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

/* ── Card ───────────────────────────────────────────────── */
.login-card {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 380px;
  padding: 2rem;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 1.5rem;
  background: rgba(15, 23, 42, 0.4);
  backdrop-filter: blur(8px);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}

.card-title {
  font-size: 1.75rem;
  font-weight: 700;
  color: #fff;
  margin: 0 0 0.25rem 0;
  font-family: var(--font-display, 'Space Grotesk', sans-serif);
}

.card-sub {
  font-size: 0.875rem;
  color: rgba(255, 255, 255, 0.6);
  margin: 0 0 1.5rem 0;
  font-weight: 400;
}

/* ── Error ───────────────────────────────────────────────– */
.error {
  padding: 0.75rem;
  margin-bottom: 1rem;
  border-radius: 0.5rem;
  background: rgba(239, 68, 68, 0.1);
  color: #fca5a5;
  font-size: 0.875rem;
  border: 1px solid rgba(239, 68, 68, 0.3);
}

/* ── Form ───────────────────────────────────────────────– */
.login-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.field label {
  font-size: 0.875rem;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.8);
}

.field input {
  padding: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 0.5rem;
  background: rgba(255, 255, 255, 0.05);
  color: #fff;
  font-size: 1rem;
  transition: background-color 0.2s, border-color 0.2s;
}

.field input:focus {
  outline: none;
  border-color: rgba(157, 114, 255, 0.5);
  background: rgba(255, 255, 255, 0.08);
}

.field input::placeholder {
  color: rgba(255, 255, 255, 0.4);
}

/* ── Remember me ────────────────────────────────────────– */
.remember-me {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  color: rgba(255, 255, 255, 0.7);
  cursor: pointer;
  user-select: none;
}

.remember-me input[type="checkbox"] {
  width: 1rem;
  height: 1rem;
  cursor: pointer;
  accent-color: #9d72ff;
}

/* ── Button ────────────────────────────────────────────– */
.btn {
  padding: 0.75rem;
  border: none;
  border-radius: 0.5rem;
  font-size: 1rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s;
}

.btn-primary {
  background: linear-gradient(135deg, #9d72ff 0%, #7c5cdb 100%);
  color: #fff;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-primary:active:not(:disabled) {
  opacity: 0.8;
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.submit-btn {
  margin-top: 0.5rem;
  width: 100%;
}

/* ═══════════════════════════════════════════════════════════
   Arc Rings — decorative rotating arc segments
   Three concentric arcs (purple, teal, pink) that sweep
   around the login card at different speeds. Created with
   conic-gradient (colored segment) + radial-gradient mask
   (thin ring shape) — pure CSS, no canvas.
   ═══════════════════════════════════════════════════════════ */

.arc-rings {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
  z-index: 0;          /* behind the login card (z-index: 1) */
}

.arc {
  position: absolute;
  border-radius: 50%;
  will-change: transform;
}

/* Arc 1 — large, purple, slow sweep (90°) */
.arc-1 {
  width: 580px;
  height: 580px;
  background: conic-gradient(from 0deg, transparent 0deg, #9d72ff 50deg, transparent 90deg);
  -webkit-mask: radial-gradient(circle, transparent 48%, #000 49%, #000 52%, transparent 53%);
  mask: radial-gradient(circle, transparent 48%, #000 49%, #000 52%, transparent 53%);
  animation: av-arc-spin 22s linear infinite;
  filter: drop-shadow(0 0 10px rgba(157, 114, 255, 0.5));
}

/* Arc 2 — medium, cyan, reverse sweep (120°) */
.arc-2 {
  width: 420px;
  height: 420px;
  background: conic-gradient(from 120deg, transparent 0deg, #6ee7ff 60deg, transparent 130deg);
  -webkit-mask: radial-gradient(circle, transparent 47%, #000 48%, #000 53%, transparent 54%);
  mask: radial-gradient(circle, transparent 47%, #000 48%, #000 53%, transparent 54%);
  animation: av-arc-spin 16s linear infinite reverse;
  filter: drop-shadow(0 0 8px rgba(110, 231, 255, 0.45));
}

/* Arc 3 — small, pink, fastest sweep (70°) */
.arc-3 {
  width: 280px;
  height: 280px;
  background: conic-gradient(from 240deg, transparent 0deg, #ff6eb4 35deg, transparent 75deg);
  -webkit-mask: radial-gradient(circle, transparent 45%, #000 46%, #000 55%, transparent 56%);
  mask: radial-gradient(circle, transparent 45%, #000 46%, #000 55%, transparent 56%);
  animation: av-arc-spin 12s linear infinite;
  filter: drop-shadow(0 0 8px rgba(255, 110, 180, 0.4));
}

@keyframes av-arc-spin {
  to { transform: rotate(360deg); }
}

/* Safety net for users who prefer reduced motion */
@media (prefers-reduced-motion: reduce) {
  .arc {
    animation: none !important;
  }
}
</style>
