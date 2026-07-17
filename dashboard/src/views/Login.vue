<template>
  <div class="login-page">
    <!-- Layer 0: Ember glow atmosphere -->
    <div class="login-aurora login-aurora-1"></div>
    <div class="login-aurora login-aurora-2"></div>

    <div class="login-watermark">
      <svg width="560" height="560" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M10 2L17 6V10C17 13.87 13.94 17.5 10 18.5C6.06 17.5 3 13.87 3 10V6L10 2Z" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round"/>
        <path d="M7 10L9.5 12.5L13 8" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
    </div>

    <!-- Vault dial — counter-rotating machined tick rings -->
    <div v-if="!prefersReducedMotion" class="arc-rings">
      <div class="arc arc-1"></div>
      <div class="arc arc-2"></div>
      <div class="arc arc-3"></div>
    </div>

    <!-- Layer 1: Login card -->
    <div class="login-shell">
      <div class="login-card">
        <h1 class="card-title">Sign in</h1>
        <p class="card-sub">Backup orchestrator console</p>

        <div v-if="errorMsg" class="error">{{ errorMsg }}</div>

        <form @submit.prevent="handleSubmit" class="login-form">
          <div class="field">
            <label for="username">Username</label>
            <input id="username" v-model="username" type="text" autocomplete="username" required placeholder="Username" />
          </div>

          <div class="field">
            <label for="password">Password</label>
            <input id="password" v-model="password" type="password" autocomplete="current-password" required placeholder="Password" />
          </div>

          <label class="remember-me">
            <input type="checkbox" v-model="remember" />
            <span class="check-label">Remember me</span>
          </label>

          <button type="submit" class="btn btn-primary submit-btn" :disabled="isSubmitting">
            {{ isSubmitting ? 'Connecting…' : 'Sign in' }}
          </button>
        </form>
      </div>
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

  const result = await auth.login(username.value, password.value, remember.value)

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

  router.push('/')
  isSubmitting.value = false
}
</script>

<style scoped>
.login-page {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: radial-gradient(120% 90% at 50% 8%, #1c1610 0%, #0d0a07 55%, #080605 100%);
  overflow: hidden;
  padding: 2rem;
}

.login-card {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 380px;
  padding: 2rem;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: var(--radius-card, 6px);
  background: rgba(30, 25, 18, 0.45);
  backdrop-filter: blur(8px);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.25);
}

.card-title {
  font-size: 1.75rem;
  font-weight: 700;
  color: #fff;
  margin: 0 0 0.25rem 0;
  font-family: var(--font-display, 'Space Grotesk', sans-serif);
}

.card-sub {
  font-family: var(--font-mono, monospace);
  font-size: 0.72rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.55);
  margin: 0 0 1.5rem 0;
  font-weight: 400;
}

.error {
  padding: 0.75rem;
  margin-bottom: 1rem;
  border-radius: 0.5rem;
  background: rgba(239, 68, 68, 0.1);
  color: #fca5a5;
  font-size: 0.875rem;
  border: 1px solid rgba(239, 68, 68, 0.3);
}

.login-form { display: flex; flex-direction: column; gap: 1rem; }
.field { display: flex; flex-direction: column; gap: 0.5rem; }

.field label {
  font-size: 0.875rem;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.8);
}

.field input {
  padding: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: var(--radius-ctrl, 4px);
  background: rgba(255, 255, 255, 0.05);
  color: #fff;
  font-size: 1rem;
  transition: background-color 0.2s, border-color 0.2s;
}

.field input:focus {
  outline: none;
  border-color: rgba(224, 160, 92, 0.55);
  background: rgba(255, 255, 255, 0.08);
}

.field input::placeholder { color: rgba(255, 255, 255, 0.4); }

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
  width: 1rem; height: 1rem; cursor: pointer;
  accent-color: #e0a05c;
}

.btn {
  padding: 0.75rem; border: none; border-radius: var(--radius-ctrl, 4px);
  font-size: 1rem; font-weight: 500; cursor: pointer;
  transition: background-color 0.2s;
}

.btn-primary {
  background: #e0a05c;
  color: #1c1610;
  font-weight: 600;
}

.btn-primary:hover:not(:disabled) { opacity: 0.9; }
.btn-primary:active:not(:disabled) { opacity: 0.8; }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }

.submit-btn { margin-top: 0.5rem; width: 100%; }

.arc-rings {
  position: absolute; inset: 0;
  display: flex; align-items: center; justify-content: center;
  pointer-events: none; z-index: 0;
}

.arc { position: absolute; border-radius: 50%; will-change: transform; }

/* Vault dial. Three machined rings, no glow:
   outer tick ring · copper sweep hairline · inner verdigris dial marks */
.arc-1 {
  width: 660px; height: 660px;
  background: repeating-conic-gradient(
    rgba(224, 160, 92, 0.22) 0deg 0.6deg,
    transparent 0.6deg 6deg
  );
  -webkit-mask: radial-gradient(circle, transparent 48.5%, #000 49%, #000 51.5%, transparent 52%);
  mask: radial-gradient(circle, transparent 48.5%, #000 49%, #000 51.5%, transparent 52%);
  animation: av-arc-spin 90s linear infinite;
}

.arc-2 {
  width: 480px; height: 480px;
  background-image: conic-gradient(from 120deg, transparent 0deg, rgba(224, 160, 92, 0.65) 45deg, transparent 80deg);
  background-color: rgba(224, 160, 92, 0.08);
  -webkit-mask: radial-gradient(circle, transparent 49%, #000 49.5%, #000 50.5%, transparent 51%);
  mask: radial-gradient(circle, transparent 49%, #000 49.5%, #000 50.5%, transparent 51%);
  animation: av-arc-spin 60s linear infinite reverse;
}

.arc-3 {
  width: 320px; height: 320px;
  background: repeating-conic-gradient(
    rgba(127, 212, 178, 0.22) 0deg 1.2deg,
    transparent 1.2deg 15deg
  );
  -webkit-mask: radial-gradient(circle, transparent 46%, #000 46.5%, #000 53%, transparent 53.5%);
  mask: radial-gradient(circle, transparent 46%, #000 46.5%, #000 53%, transparent 53.5%);
  animation: av-arc-spin 45s linear infinite;
}

@keyframes av-arc-spin { to { transform: rotate(360deg); } }

@media (prefers-reduced-motion: reduce) { .arc { animation: none !important; } }
</style>
