<template>
  <div class="login-container">
    <div class="login-box">
      <h1>ArcVault</h1>
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

        <div class="form-group checkbox">
          <label>
            <input
              v-model="rememberMe"
              type="checkbox"
              :disabled="loading"
            />
            Remember me
          </label>
        </div>

        <div v-if="error" class="error-message">
          {{ error }}
        </div>

        <button
          type="submit"
          class="btn-login"
          :disabled="loading || !username || !password"
        >
          {{ loading ? 'Signing in...' : 'Sign In' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '../composables/useAuth.js'

const router = useRouter()
const auth = useAuth()

const username = ref('')
const password = ref('')
const rememberMe = ref(false)
const loading = ref(false)
const error = ref('')

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

  const result = await auth.login(username.value, password.value, rememberMe.value)

  if (result.success) {
    // Redirect to agents page
    router.push('/agents')
  } else {
    error.value = result.error || 'Login failed. Please try again.'
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100vh;
  background: var(--bg-secondary);
}

.login-box {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 40px;
  width: 100%;
  max-width: 400px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

h1 {
  font-size: 28px;
  font-weight: 600;
  margin: 0 0 8px 0;
  color: var(--text-primary);
}

h2 {
  font-size: 16px;
  font-weight: 400;
  margin: 0 0 24px 0;
  color: var(--text-secondary);
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
  color: var(--text-primary);
}

.form-group input[type='text'],
.form-group input[type='password'] {
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--bg-primary);
  color: var(--text-primary);
  font-size: 14px;
  transition: border-color 0.2s;
}

.form-group input[type='text']:focus,
.form-group input[type='password']:focus {
  outline: none;
  border-color: var(--accent-color);
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
}

.form-group input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.form-group.checkbox {
  flex-direction: row;
  align-items: center;
}

.form-group.checkbox label {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-weight: 400;
  cursor: pointer;
}

.form-group.checkbox input[type='checkbox'] {
  width: 16px;
  height: 16px;
  cursor: pointer;
}

.error-message {
  padding: 12px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 4px;
  color: var(--error-color, #ef4444);
  font-size: 14px;
}

.btn-login {
  padding: 12px 16px;
  background: var(--accent-color, #6366f1);
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
  background: var(--accent-color-hover, #4f46e5);
}

.btn-login:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Dark mode */
[data-theme='dark'] {
  --bg-primary: #1f2937;
  --bg-secondary: #111827;
  --border-color: #374151;
  --text-primary: #f3f4f6;
  --text-secondary: #9ca3af;
  --accent-color: #6366f1;
  --accent-color-hover: #4f46e5;
  --error-color: #ef4444;
}

/* Light mode */
[data-theme='light'] {
  --bg-primary: #ffffff;
  --bg-secondary: #f9fafb;
  --border-color: #e5e7eb;
  --text-primary: #111827;
  --text-secondary: #6b7280;
  --accent-color: #6366f1;
  --accent-color-hover: #4f46e5;
  --error-color: #dc2626;
}
</style>
