<template>
  <div class="login-page">

    <!-- Atmospheric background -->
    <div class="bg-orb bg-orb--a" aria-hidden="true"></div>
    <div class="bg-orb bg-orb--b" aria-hidden="true"></div>
    <div class="bg-grid" aria-hidden="true"></div>

    <div class="login-shell">

      <!-- Brand mark -->
      <div class="brand">
        <div class="brand-icon">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M12 3L20 7.5V12C20 16.42 16.47 20.5 12 21.5C7.53 20.5 4 16.42 4 12V7.5L12 3Z"
              stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/>
            <path d="M8.5 12L11.5 15L15.5 9.5"
              stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </div>
        <span class="brand-name">ArcVault</span>
      </div>

      <!-- Card -->
      <div class="login-card">

        <!-- Change password flow -->
        <template v-if="showChangePassword">
          <div class="card-header">
            <h1 class="card-title">Set new password</h1>
            <p class="card-sub">You must set a new password before continuing.</p>
          </div>

          <div class="notice">
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><circle cx="7" cy="7" r="5.5" stroke="currentColor" stroke-width="1.2"/><path d="M7 6v3" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/><circle cx="7" cy="4.5" r="0.6" fill="currentColor"/></svg>
            Your account requires a password change.
          </div>

          <form @submit.prevent="handleChangePassword" class="login-form">
            <div class="field">
              <label for="new-password">New password</label>
              <div class="input-wrap">
                <svg class="input-icon" width="14" height="14" viewBox="0 0 14 14" fill="none"><rect x="3" y="6" width="8" height="6" rx="1" stroke="currentColor" stroke-width="1.2"/><path d="M5 6V4.5a2 2 0 0 1 4 0V6" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/></svg>
                <input
                  id="new-password"
                  v-model="newPassword"
                  type="password"
                  placeholder="At least 8 characters"
                  :disabled="cpLoading"
                  autocomplete="new-password"
                />
              </div>
            </div>
            <div class="field">
              <label for="confirm-password">Confirm password</label>
              <div class="input-wrap">
                <svg class="input-icon" width="14" height="14" viewBox="0 0 14 14" fill="none"><rect x="3" y="6" width="8" height="6" rx="1" stroke="currentColor" stroke-width="1.2"/><path d="M5 6V4.5a2 2 0 0 1 4 0V6" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/></svg>
                <input
                  id="confirm-password"
                  v-model="confirmPassword"
                  type="password"
                  placeholder="Repeat new password"
                  :disabled="cpLoading"
                  autocomplete="new-password"
                />
              </div>
            </div>
            <div v-if="cpError" class="form-error">
              <svg width="13" height="13" viewBox="0 0 13 13" fill="none"><circle cx="6.5" cy="6.5" r="5" stroke="currentColor" stroke-width="1.2"/><path d="M6.5 4v3" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/><circle cx="6.5" cy="9" r="0.6" fill="currentColor"/></svg>
              {{ cpError }}
            </div>
            <button
              type="submit"
              class="submit-btn"
              :disabled="cpLoading || !newPassword || !confirmPassword"
            >
              <span v-if="cpLoading" class="spinner"></span>
              <svg v-else width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M2.5 7.5L5.5 10.5L11.5 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
              {{ cpLoading ? 'Saving…' : 'Set password' }}
            </button>
          </form>
        </template>

        <!-- Normal login -->
        <template v-else>
          <div class="card-header">
            <h1 class="card-title">Sign in</h1>
            <p class="card-sub">Backup orchestrator console</p>
          </div>

          <form @submit.prevent="handleLogin" class="login-form">
            <div class="field">
              <label for="username">Username</label>
              <div class="input-wrap">
                <svg class="input-icon" width="14" height="14" viewBox="0 0 14 14" fill="none"><circle cx="7" cy="5" r="2.5" stroke="currentColor" stroke-width="1.2"/><path d="M2 12c0-2.76 2.24-5 5-5s5 2.24 5 5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/></svg>
                <input
                  id="username"
                  v-model="username"
                  type="text"
                  placeholder="your username"
                  autocomplete="username"
                  :disabled="loading"
                />
              </div>
            </div>
            <div class="field">
              <label for="password">Password</label>
              <div class="input-wrap">
                <svg class="input-icon" width="14" height="14" viewBox="0 0 14 14" fill="none"><rect x="3" y="6" width="8" height="6" rx="1" stroke="currentColor" stroke-width="1.2"/><path d="M5 6V4.5a2 2 0 0 1 4 0V6" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/><circle cx="7" cy="9" r="0.75" fill="currentColor"/></svg>
                <input
                  id="password"
                  v-model="password"
                  type="password"
                  placeholder="••••••••"
                  autocomplete="current-password"
                  :disabled="loading"
                />
              </div>
            </div>
            <div v-if="error" class="form-error">
              <svg width="13" height="13" viewBox="0 0 13 13" fill="none"><circle cx="6.5" cy="6.5" r="5" stroke="currentColor" stroke-width="1.2"/><path d="M6.5 4v3" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/><circle cx="6.5" cy="9" r="0.6" fill="currentColor"/></svg>
              {{ error }}
            </div>
            <button
              type="submit"
              class="submit-btn"
              :disabled="loading || !username || !password"
            >
              <span v-if="loading" class="spinner"></span>
              <svg v-else width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M5 3l5 4-5 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
              {{ loading ? 'Signing in…' : 'Sign in' }}
            </button>
          </form>
        </template>
      </div>

      <p class="login-footer">ArcVault — Distributed backup orchestration</p>
    </div>
  </div>
</template>

<script setup>
import { ref, inject } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '../composables/useAuth.js'

const router = useRouter()
const auth = useAuth()
const connectWs = inject('connectWs', () => {})

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

// Change-password state
const showChangePassword = ref(false)
const newPassword = ref('')
const confirmPassword = ref('')
const cpLoading = ref(false)
const cpError = ref('')

// If already authenticated, redirect
if (auth.isAuthenticated.value) {
  router.push('/agents')
}

async function handleLogin() {
  if (!username.value || !password.value) {
    error.value = 'Please enter username and password'
    return
  }

  loading.value = true
  error.value = ''

  const result = await auth.login(username.value, password.value, true)

  if (result.success) {
    connectWs()
    if (result.mustChangePassword) {
      showChangePassword.value = true
      loading.value = false
    } else {
      router.push('/agents')
    }
  } else {
    error.value = result.error || 'Login failed. Please try again.'
    loading.value = false
  }
}

async function handleChangePassword() {
  cpError.value = ''

  if (newPassword.value.length < 8) {
    cpError.value = 'Password must be at least 8 characters.'
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    cpError.value = 'Passwords do not match.'
    return
  }

  cpLoading.value = true
  const result = await auth.changePassword('changeme', newPassword.value)
  cpLoading.value = false

  if (result.success) {
    const reLogin = await auth.login(username.value, newPassword.value, true)
    if (reLogin.success) {
      connectWs()
      router.push('/agents')
    } else {
      cpError.value = 'Password changed — please log in again.'
      showChangePassword.value = false
      auth.clearAuth()
    }
  } else {
    cpError.value = result.error || 'Failed to change password.'
  }
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

/* ── Background atmosphere ───────────────────────────────── */
.bg-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  pointer-events: none;
}

.bg-orb--a {
  width: 500px;
  height: 500px;
  background: radial-gradient(circle, rgba(0, 212, 170, 0.12) 0%, transparent 70%);
  top: -100px;
  left: -100px;
  animation: orb-drift-a 12s ease-in-out infinite alternate;
}

.bg-orb--b {
  width: 400px;
  height: 400px;
  background: radial-gradient(circle, rgba(91, 141, 238, 0.10) 0%, transparent 70%);
  bottom: -80px;
  right: -60px;
  animation: orb-drift-b 15s ease-in-out infinite alternate;
}

@keyframes orb-drift-a {
  from { transform: translate(0, 0) scale(1); }
  to   { transform: translate(60px, 40px) scale(1.1); }
}

@keyframes orb-drift-b {
  from { transform: translate(0, 0) scale(1); }
  to   { transform: translate(-40px, -60px) scale(0.9); }
}

/* Dot grid overlay */
.bg-grid {
  position: absolute;
  inset: 0;
  background-image: radial-gradient(circle, var(--border-subtle) 1px, transparent 1px);
  background-size: 28px 28px;
  pointer-events: none;
  mask-image: radial-gradient(ellipse 80% 80% at 50% 50%, black 20%, transparent 80%);
  -webkit-mask-image: radial-gradient(ellipse 80% 80% at 50% 50%, black 20%, transparent 80%);
}

/* ── Content ─────────────────────────────────────────────── */
.login-shell {
  position: relative;
  z-index: 10;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1.5rem;
  width: 100%;
  max-width: 380px;
  animation: shell-in 0.4s cubic-bezier(0.22, 1, 0.36, 1) both;
}

@keyframes shell-in {
  from { opacity: 0; transform: translateY(12px); }
  to   { opacity: 1; transform: translateY(0); }
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

/* ── Card ────────────────────────────────────────────────── */
.login-card {
  width: 100%;
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: 12px;
  padding: 1.75rem;
  box-shadow: var(--shadow-lg);
}

.card-header {
  margin-bottom: 1.5rem;
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
  margin: 0;
}

/* ── Notice ──────────────────────────────────────────────── */
.notice {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: var(--accent-dim);
  border: 1px solid var(--accent-border);
  color: var(--accent);
  font-family: var(--font-body);
  font-size: 0.82rem;
  padding: 0.6rem 0.85rem;
  border-radius: 6px;
  margin-bottom: 1.25rem;
}

/* ── Form ────────────────────────────────────────────────── */
.login-form {
  display: flex;
  flex-direction: column;
  gap: 1.1rem;
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
  border-radius: 6px;
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

.input-wrap input:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-dim);
}

.input-wrap input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ── Error ───────────────────────────────────────────────── */
.form-error {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.6rem 0.85rem;
  background: var(--bg-error);
  border: 1px solid rgba(255, 77, 109, 0.3);
  border-radius: 6px;
  color: var(--color-error);
  font-family: var(--font-body);
  font-size: 0.82rem;
}

/* ── Submit button ───────────────────────────────────────── */
.submit-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  width: 100%;
  padding: 0.65rem 1rem;
  margin-top: 0.25rem;
  background: var(--accent);
  color: var(--bg-base);
  border: none;
  border-radius: 6px;
  font-family: var(--font-display);
  font-size: 0.9rem;
  font-weight: 700;
  letter-spacing: 0.02em;
  cursor: pointer;
  transition: filter 0.15s, transform 0.1s;
}

.submit-btn:hover:not(:disabled) {
  filter: brightness(1.08);
  transform: translateY(-1px);
}

.submit-btn:active:not(:disabled) {
  transform: translateY(0);
}

.submit-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
  transform: none;
}

/* ── Spinner ─────────────────────────────────────────────── */
.spinner {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 2px solid rgba(0,0,0,0.2);
  border-top-color: currentColor;
  animation: spin 0.6s linear infinite;
  flex-shrink: 0;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ── Footer ──────────────────────────────────────────────── */
.login-footer {
  font-family: var(--font-body);
  font-size: 0.75rem;
  color: var(--text-muted);
  text-align: center;
  margin: 0;
}

/* ── Light theme adjustments ─────────────────────────────── */
[data-theme="light"] .bg-orb--a {
  background: radial-gradient(circle, rgba(0, 168, 135, 0.08) 0%, transparent 70%);
}
[data-theme="light"] .bg-orb--b {
  background: radial-gradient(circle, rgba(59, 111, 212, 0.06) 0%, transparent 70%);
}
[data-theme="light"] .bg-grid {
  background-image: radial-gradient(circle, var(--border-default) 1px, transparent 1px);
}
</style>
