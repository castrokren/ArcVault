<template>
  <div class="login-container">
    <div class="login-box">
      <h1>ArcVault</h1>

      <!-- Change password form (shown after first login) -->
      <template v-if="showChangePassword">
        <h2>Change Password</h2>
        <p class="info-message">You must set a new password before continuing.</p>
        <form @submit.prevent="handleChangePassword">
          <div class="form-group">
            <label for="new-password">New Password</label>
            <input
              id="new-password"
              v-model="newPassword"
              type="password"
              placeholder="At least 8 characters"
              :disabled="cpLoading"
            />
          </div>
          <div class="form-group">
            <label for="confirm-password">Confirm Password</label>
            <input
              id="confirm-password"
              v-model="confirmPassword"
              type="password"
              placeholder="Repeat new password"
              :disabled="cpLoading"
            />
          </div>
          <div v-if="cpError" class="error-message">{{ cpError }}</div>
          <button
            type="submit"
            class="btn-login"
            :disabled="cpLoading || !newPassword || !confirmPassword"
          >
            {{ cpLoading ? 'Saving...' : 'Set Password' }}
          </button>
        </form>
      </template>

      <!-- Normal login form -->
      <template v-else>
        <h2>Sign In</h2>
        <form @submit.prevent="handleLogin">
          <div class="form-group">
            <label for="username">Username</label>
            <input
              id="username"
              v-model="username"
              type="text"
              placeholder="Enter your username"
              autocomplete="username"
              @keyup.enter="handleLogin"
              :disabled="loading"
            />
          </div>
          <div class="form-group">
            <label for="password">Password</label>
            <input
              id="password"
              v-model="password"
              type="password"
              placeholder="Enter your password"
              autocomplete="current-password"
              @keyup.enter="handleLogin"
              :disabled="loading"
            />
          </div>
          <div v-if="error" class="error-message">{{ error }}</div>
          <button
            type="submit"
            class="btn-login"
            :disabled="loading || !username || !password"
          >
            {{ loading ? 'Signing in...' : 'Sign In' }}
          </button>
        </form>
      </template>
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
.login-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: calc(100vh - 60px);
  background: var(--bg-secondary, #111827);
}

.login-box {
  background: var(--bg-primary, #1f2937);
  border: 1px solid var(--border-color, #374151);
  border-radius: 8px;
  padding: 40px;
  width: 100%;
  max-width: 400px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}

h1 {
  font-size: 28px;
  font-weight: 600;
  margin: 0 0 8px 0;
  color: var(--text-primary, #f3f4f6);
}

h2 {
  font-size: 16px;
  font-weight: 400;
  margin: 0 0 24px 0;
  color: var(--text-secondary, #9ca3af);
}

form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary, #f3f4f6);
}

.form-group input[type='text'],
.form-group input[type='password'] {
  padding: 10px 12px;
  border: 1px solid var(--border-color, #374151);
  border-radius: 4px;
  background: var(--bg-secondary, #111827);
  color: var(--text-primary, #f3f4f6);
  font-size: 14px;
  transition: border-color 0.2s;
}

.form-group input[type='text']:focus,
.form-group input[type='password']:focus {
  outline: none;
  border-color: #6366f1;
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
}

.form-group input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.info-message {
  padding: 10px 12px;
  background: rgba(99, 102, 241, 0.08);
  border: 1px solid rgba(99, 102, 241, 0.3);
  border-radius: 4px;
  color: var(--text-secondary, #9ca3af);
  font-size: 13px;
  margin-bottom: 8px;
}

.error-message {
  padding: 12px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 4px;
  color: #ef4444;
  font-size: 14px;
}

.btn-login {
  padding: 12px 16px;
  background: #6366f1;
  color: white;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
  margin-top: 8px;
}

.btn-login:hover:not(:disabled) {
  background: #4f46e5;
}

.btn-login:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
