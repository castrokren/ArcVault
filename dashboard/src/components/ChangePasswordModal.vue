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

  // Check password strength
  if (pwd.length < 8) {
    passwordStrength.value = 'weak'
    strengthLabel.value = 'Weak (min 8 characters)'
  } else if (/[A-Z]/.test(pwd) && /[0-9]/.test(pwd)) {
    passwordStrength.value = 'strong'
    strengthLabel.value = 'Strong'
  } else if (/[A-Z]/.test(pwd) || /[0-9]/.test(pwd)) {
    passwordStrength.value = 'medium'
    strengthLabel.value = 'Medium'
  } else {
    passwordStrength.value = 'weak'
    strengthLabel.value = 'Weak'
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
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-dialog {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  width: 100%;
  max-width: 450px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.3);
  max-height: 90vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.modal-close {
  background: none;
  border: none;
  font-size: 24px;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: background 0.2s, color 0.2s;
}

.modal-close:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.modal-body {
  padding: 20px;
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

.form-group input[type='password'] {
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--bg-primary);
  color: var(--text-primary);
  font-size: 14px;
  transition: border-color 0.2s;
}

.form-group input[type='password']:focus {
  outline: none;
  border-color: var(--accent-color);
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
}

.form-group input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.strength-indicator {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 12px;
}

.strength-bar {
  height: 4px;
  background: var(--border-color);
  border-radius: 2px;
  overflow: hidden;
}

.strength-indicator.weak .strength-bar {
  background: #ef4444;
}

.strength-indicator.medium .strength-bar {
  background: #f59e0b;
}

.strength-indicator.strong .strength-bar {
  background: #10b981;
}

.strength-text {
  color: var(--text-secondary);
  font-size: 12px;
}

.error-message {
  padding: 12px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 4px;
  color: #ef4444;
  font-size: 14px;
}

.success-message {
  padding: 12px;
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.3);
  border-radius: 4px;
  color: #10b981;
  font-size: 14px;
}

.modal-footer {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
}

.btn {
  padding: 10px 16px;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-primary {
  background: var(--accent-color);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: var(--accent-color-hover);
}

.btn-secondary {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--border-color);
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Dark mode */
[data-theme='dark'] {
  --bg-primary: #1f2937;
  --bg-secondary: #374151;
  --border-color: #4b5563;
  --text-primary: #f3f4f6;
  --text-secondary: #9ca3af;
  --accent-color: #6366f1;
  --accent-color-hover: #4f46e5;
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
}
</style>
