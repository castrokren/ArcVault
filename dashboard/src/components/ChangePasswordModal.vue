<template>
  <div v-if="isOpen" class="modal-overlay" @click.self="handleClose">
    <div class="modal-dialog">
      <div class="modal-header">
        <h2>Change Password</h2>
        <button class="modal-close" @click="handleClose" aria-label="Close">×</button>
      </div>

      <form @submit.prevent="handleChangePassword" class="modal-body">
        <div class="form-group">
          <label for="current-password">Current Password</label>
          <input
            id="current-password"
            v-model="currentPassword"
            type="password"
            placeholder="Enter your current password"
            :disabled="loading"
            autocomplete="current-password"
          />
        </div>

        <div class="form-group">
          <label for="new-password">New Password</label>
          <input
            id="new-password"
            v-model="newPassword"
            type="password"
            placeholder="Enter a new password"
            :disabled="loading"
            autocomplete="new-password"
            @input="updateStrength"
          />
          <div v-if="newPassword" class="strength-indicator" :class="passwordStrength">
            <div class="strength-bar"></div>
            <span class="strength-text">{{ strengthLabel }}</span>
          </div>
        </div>

        <div class="form-group">
          <label for="confirm-password">Confirm New Password</label>
          <input
            id="confirm-password"
            v-model="confirmPassword"
            type="password"
            placeholder="Confirm your new password"
            :disabled="loading"
            autocomplete="new-password"
          />
        </div>

        <div v-if="error" class="error-message">
          {{ error }}
        </div>

        <div v-if="success" class="success-message">
          Password changed successfully!
        </div>

        <div class="modal-footer">
          <button
            type="button"
            class="btn btn-secondary"
            @click="handleClose"
            :disabled="loading"
          >
            Cancel
          </button>
          <button
            type="submit"
            class="btn btn-primary"
            :disabled="loading || !canSubmit"
          >
            {{ loading ? 'Changing...' : 'Change Password' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useAuth } from '../composables/useAuth.js'

const props = defineProps({
  isOpen: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['close', 'success'])

const auth = useAuth()
const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const error = ref('')
const success = ref(false)
const passwordStrength = ref('weak')
const strengthLabel = ref('Weak')

const canSubmit = computed(() => {
  return currentPassword.value && newPassword.value && confirmPassword.value && newPassword.value === confirmPassword.value
})

function updateStrength() {
  const pwd = newPassword.value
  const hasUpper = /[A-Z]/.test(pwd)
  const hasLower = /[a-z]/.test(pwd)
  const hasDigit = /[0-9]/.test(pwd)
  const hasSpecial = /[^A-Za-z0-9]/.test(pwd)
  const classes = [hasUpper, hasLower, hasDigit, hasSpecial].filter(Boolean).length

  if (pwd.length < 8) {
    passwordStrength.value = 'weak'
    strengthLabel.value = 'Weak — too short'
  } else if (classes < 4) {
    passwordStrength.value = 'weak'
    strengthLabel.value = 'Weak — need uppercase, lowercase, digit & special character'
  } else if (classes >= 3 && pwd.length >= 10) {
    passwordStrength.value = 'strong'
    strengthLabel.value = 'Strong'
  } else {
    passwordStrength.value = 'medium'
    strengthLabel.value = 'Medium'
  }
}

async function handleChangePassword() {
  error.value = ''
  success.value = false

  // Validate
  if (!currentPassword.value || !newPassword.value || !confirmPassword.value) {
    error.value = 'All fields are required'
    return
  }

  if (newPassword.value !== confirmPassword.value) {
    error.value = 'New passwords do not match'
    return
  }

  if (newPassword.value.length < 8) {
    error.value = 'Password must be at least 8 characters'
    return
  }

  if (passwordStrength.value === 'weak') {
    error.value = 'Password is too weak. Include uppercase, lowercase, digit, and special character.'
    return
  }

  loading.value = true

  const result = await auth.changePassword(currentPassword.value, newPassword.value)

  if (result.success) {
    success.value = true
    // Reset form
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    passwordStrength.value = 'weak'

    // Close after 2 seconds
    setTimeout(() => {
      handleClose()
      emit('success')
    }, 2000)
  } else {
    error.value = result.error || 'Failed to change password'
    loading.value = false
  }
}

function handleClose() {
  if (loading.value) return

  currentPassword.value = ''
  newPassword.value = ''
  confirmPassword.value = ''
  error.value = ''
  success.value = false
  passwordStrength.value = 'weak'

  emit('close')
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-dialog {
  background: var(--bg-elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-card);
  animation: modal-pop 0.18s ease-out;
  width: 100%;
  max-width: 450px;
  box-shadow: var(--shadow-lg);
  max-height: 90vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1.1rem 1.5rem;
  border-bottom: 1px solid var(--border-subtle);
}

.modal-header h2 {
  margin: 0;
  font-family: var(--font-display);
  font-size: 1rem;
  font-weight: 700;
  color: var(--text-primary);
}

.modal-close {
  background: none;
  border: none;
  font-size: 1.4rem;
  color: var(--text-muted);
  cursor: pointer;
  padding: 0;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: background 0.15s, color 0.15s;
  line-height: 1;
}

.modal-close:hover {
  background: var(--bg-elevated);
  color: var(--text-primary);
}

.modal-body {
  padding: 1.25rem 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  font-family: var(--font-body);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.form-group label {
  font-family: var(--font-body);
  font-size: 0.82rem;
  font-weight: 500;
  color: var(--text-secondary);
}

.form-group input[type='password'] {
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--border-default);
  border-radius: 5px;
  background: var(--bg-input);
  color: var(--text-primary);
  font-family: var(--font-body);
  font-size: 0.88rem;
  transition: border-color 0.15s;
}

.form-group input[type='password']:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: var(--glow-accent);
}

.form-group input:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.strength-indicator {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.strength-bar {
  height: 3px;
  background: var(--border-default);
  border-radius: 2px;
  overflow: hidden;
}

.strength-indicator.weak   .strength-bar { background: var(--color-error); }
.strength-indicator.medium .strength-bar { background: var(--color-warning); }
.strength-indicator.strong .strength-bar { background: var(--color-success); }

.strength-text {
  font-family: var(--font-body);
  font-size: 0.75rem;
  color: var(--text-muted);
}

.error-message {
  padding: 0.65rem 0.85rem;
  background: var(--bg-error);
  border: 1px solid rgba(255, 92, 122, 0.3);
  border-radius: 5px;
  color: var(--color-error);
  font-family: var(--font-body);
  font-size: 0.85rem;
}

.success-message {
  padding: 0.65rem 0.85rem;
  background: var(--bg-success);
  border: 1px solid rgba(34, 201, 147, 0.3);
  border-radius: 5px;
  color: var(--color-success);
  font-family: var(--font-body);
  font-size: 0.85rem;
}

.modal-footer {
  display: flex;
  gap: 0.65rem;
  justify-content: flex-end;
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--border-subtle);
}

.btn {
  display: inline-flex;
  align-items: center;
  padding: 0.45rem 1rem;
  border: none;
  border-radius: 5px;
  font-family: var(--font-body);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: filter 0.15s, background 0.15s;
}

.btn-primary {
  background: var(--accent);
  color: var(--bg-base);
}
.btn-primary:hover:not(:disabled) { filter: brightness(1.1); }

.btn-secondary {
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
}
.btn-secondary:hover:not(:disabled) {
  background: var(--bg-elevated);
  color: var(--text-primary);
  border-color: var(--border-strong);
}

.btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
</style>
