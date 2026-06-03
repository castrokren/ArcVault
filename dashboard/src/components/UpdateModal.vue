<template>
  <div v-if="isOpen" class="modal-overlay" @click.self="close">
    <div class="modal">
      <!-- Confirm State -->
      <div v-if="state === 'confirm'" class="modal-content">
        <h2>Update Available</h2>
        <div class="version-info">
          <p><strong>Current:</strong> {{ updateStore.current }}</p>
          <p><strong>New:</strong> {{ updateStore.latest }}</p>
        </div>
        <div class="warning">
          <strong>⚠ The coordinator will restart during the update.</strong>
          <p>Any running jobs will need to be rescheduled.</p>
        </div>
        <div v-if="updateStore.releaseUrl" class="release-link">
          <a :href="updateStore.releaseUrl" target="_blank" rel="noopener">
            View release notes →
          </a>
        </div>
        <div class="modal-actions">
          <button class="btn-cancel" @click="close">Cancel</button>
          <button class="btn-update" @click="startUpdate">Update now</button>
        </div>
      </div>

      <!-- In Progress State -->
      <div v-else-if="state === 'progress'" class="modal-content">
        <h2>Updating Coordinator</h2>
        <div class="steps-container">
          <div
            v-for="step in steps"
            :key="step.name"
            class="step"
            :class="{ active: currentStep === step.name, done: isStepComplete(step.name) }"
          >
            <div class="step-indicator">
              <span v-if="isStepComplete(step.name)" class="check">✓</span>
              <span v-else-if="currentStep === step.name" class="spinner">⟳</span>
              <span v-else class="number">{{ steps.indexOf(step) + 1 }}</span>
            </div>
            <div class="step-content">
              <div class="step-name">{{ step.label }}</div>
              <div v-if="currentStep === step.name && progressMessage" class="step-message">
                {{ progressMessage }}
              </div>
            </div>
          </div>
        </div>

        <!-- Progress Bar (for download step) -->
        <div v-if="currentStep === 'downloading'" class="progress-bar-container">
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: progressPercent + '%' }"></div>
          </div>
        </div>
      </div>

      <!-- Reconnecting State -->
      <div v-else-if="state === 'reconnecting'" class="modal-content">
        <div class="spinner-large">⟳</div>
        <h2>Waiting for Coordinator</h2>
        <p>The coordinator is restarting with the new version...</p>
        <p class="countdown" v-if="reconnectCountdown > 0">
          Trying to reconnect ({{ reconnectCountdown }}s)
        </p>
      </div>

      <!-- Success State (Service Mode) -->
      <div v-else-if="state === 'success'" class="modal-content success">
        <div class="success-icon">✓</div>
        <h2>Update Complete</h2>
        <p>Updated to <strong>{{ updateStore.latest }}</strong> — reconnected successfully!</p>
        <div class="modal-actions">
          <button class="btn-close" @click="close">Close</button>
        </div>
      </div>

      <!-- Success State (Terminal Mode) -->
      <div v-else-if="state === 'success_manual'" class="modal-content success">
        <div class="success-icon">✓</div>
        <h2>Binary Updated</h2>
        <p>The coordinator binary has been updated to <strong>{{ updateStore.latest }}</strong>.</p>
        <p class="warning">Please restart the coordinator manually to complete the update.</p>
        <div class="modal-actions">
          <button class="btn-close" @click="close">Close</button>
        </div>
      </div>

      <!-- Error State -->
      <div v-else-if="state === 'error'" class="modal-content error">
        <div class="error-icon">✕</div>
        <h2>Update Failed</h2>
        <p>The coordinator was not modified.</p>
        <div class="error-detail">
          <pre>{{ errorMessage }}</pre>
        </div>
        <div class="modal-actions">
          <button class="btn-close" @click="close">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, inject, watch, onUnmounted } from 'vue'
import { getToken, applyCoordinatorUpdate, saveToken } from '../api'
import { useAuth } from '../composables/useAuth.js'

const props = defineProps({
  isOpen: Boolean,
  lastEvent: Object
})

const emit = defineEmits(['close'])

const updateStore = inject('updateStore', {
  current: 'v0.2.0',
  latest: 'v0.2.0',
  available: false,
  releaseUrl: ''
})

// State management
const state = ref('confirm') // confirm, progress, reconnecting, success, success_manual, error
const currentStep = ref('resolving')
const progressPercent = ref(0)
const progressMessage = ref('')
const errorMessage = ref('')
const reconnectCountdown = ref(60)

const steps = [
  { name: 'resolving', label: 'Resolving release asset' },
  { name: 'downloading', label: 'Downloading binary' },
  { name: 'verifying', label: 'Verifying binary' },
  { name: 'staging', label: 'Staging binary' },
  { name: 'restarting', label: 'Restarting service' }
]

let reconnectTimer = null
let countdownTimer = null

// Watch for WebSocket events
watch(() => props.lastEvent, (evt) => {
  if (!evt || evt.type !== 'update_progress') return

  const payload = evt.payload
  currentStep.value = payload.step
  progressPercent.value = payload.pct

  if (payload.step === 'done') {
    state.value = 'success'
    clearTimers()
  } else if (payload.step === 'done_manual') {
    state.value = 'success_manual'
    clearTimers()
  } else if (payload.step === 'error') {
    state.value = 'error'
    errorMessage.value = payload.message
    clearTimers()
  } else if (payload.step === 'restarting') {
    // Start reconnection polling
    state.value = 'reconnecting'
    startReconnectPolling()
  } else {
    progressMessage.value = payload.message
  }
})

function startUpdate() {
  state.value = 'progress'
  progressPercent.value = 0
  progressMessage.value = ''
  currentStep.value = 'resolving'

  const token = getToken()
  if (!token) {
    state.value = 'error'
    errorMessage.value = 'Missing authentication token. Please sign in again.'
    return
  }

  const auth = useAuth()

  console.log('Refreshing token before update...')

  // Refresh token to ensure it's valid, then apply update
  auth.refreshToken()
    .then(success => {
      if (!success) {
        throw new Error('Token refresh failed')
      }
      console.log('Token refreshed, applying coordinator update')
      return applyCoordinatorUpdate()
    })
    .then(() => {
      console.log('Coordinator update request accepted')
    })
    .catch(err => {
      console.error('Coordinator update failed:', err)
      state.value = 'error'
      errorMessage.value = `Failed to start update: ${err.message}`
      clearTimers()
    })
}

function startReconnectPolling() {
  reconnectCountdown.value = 60

  countdownTimer = setInterval(() => {
    reconnectCountdown.value--
    if (reconnectCountdown.value <= 0) {
      clearInterval(countdownTimer)
    }
  }, 1000)

  // Poll for WebSocket reconnection
  reconnectTimer = setInterval(() => {
    // The useWebSocket composable automatically reconnects
    // We can check if it's connected by looking at the ws state
    // For now, we'll rely on the 'success' event from the server
  }, 2000)

  // Timeout after 60 seconds
  setTimeout(() => {
    if (state.value === 'reconnecting') {
      state.value = 'error'
      errorMessage.value = 'Coordinator may still be restarting. Try refreshing.'
      clearTimers()
    }
  }, 60000)
}

function isStepComplete(stepName) {
  const stepIndex = steps.findIndex(s => s.name === stepName)
  const currentIndex = steps.findIndex(s => s.name === currentStep.value)
  return stepIndex < currentIndex
}

function close() {
  state.value = 'confirm'
  clearTimers()
  emit('close')
}

function clearTimers() {
  if (reconnectTimer) {
    clearInterval(reconnectTimer)
    reconnectTimer = null
  }
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

onUnmounted(() => {
  clearTimers()
})
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.65);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal {
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: 10px;
  box-shadow: var(--shadow-lg);
  width: 500px;
  max-width: 95vw;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-content {
  padding: 1.75rem 2rem;
}

.modal-content h2 {
  margin: 0 0 1.25rem;
  font-family: var(--font-display);
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--text-primary);
}

.modal-content p {
  margin: 0.4rem 0;
  line-height: 1.6;
  color: var(--text-secondary);
  font-family: var(--font-body);
  font-size: 0.9rem;
}
.modal-content strong { color: var(--text-primary); }

.version-info {
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: 6px;
  padding: 0.85rem 1rem;
  margin: 1rem 0;
  font-family: var(--font-mono);
  font-size: 0.82rem;
  color: var(--text-secondary);
}
.version-info p { font-family: var(--font-mono); color: var(--text-secondary); }
.version-info strong { color: var(--text-primary); }

.warning {
  background: var(--bg-warning);
  border-left: 3px solid var(--color-warning);
  padding: 0.85rem 1rem;
  border-radius: 4px;
  margin: 1rem 0;
  font-family: var(--font-body);
  font-size: 0.85rem;
  color: var(--color-warning);
}
.warning strong { display: block; margin-bottom: 0.3rem; color: var(--color-warning); }
.warning p { margin: 0; color: var(--color-warning); opacity: 0.85; font-family: var(--font-body); }

.release-link {
  text-align: center;
  margin: 0.75rem 0;
}
.release-link a {
  font-family: var(--font-body);
  font-size: 0.85rem;
  color: var(--accent);
  text-decoration: none;
}
.release-link a:hover { text-decoration: underline; }

.modal-actions {
  display: flex;
  gap: 0.75rem;
  margin-top: 1.5rem;
  justify-content: flex-end;
}

.btn-cancel, .btn-update, .btn-close {
  display: inline-flex;
  align-items: center;
  padding: 0.45rem 1.1rem;
  border: none;
  border-radius: 5px;
  cursor: pointer;
  font-family: var(--font-body);
  font-size: 0.88rem;
  font-weight: 600;
  transition: filter 0.15s, background 0.15s;
}
.btn-cancel {
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
}
.btn-cancel:hover { background: var(--bg-elevated); color: var(--text-primary); border-color: var(--border-strong); }
.btn-update, .btn-close { background: var(--accent); color: var(--bg-base); }
.btn-update:hover, .btn-close:hover { filter: brightness(1.1); }

/* Progress state */
.steps-container { margin: 1.25rem 0; }

.step { display: flex; gap: 1rem; margin: 0.85rem 0; align-items: flex-start; }

.step-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: 2px solid var(--border-strong);
  border-radius: 50%;
  color: var(--text-muted);
  font-family: var(--font-body);
  font-size: 0.78rem;
  font-weight: 700;
  flex-shrink: 0;
}
.step.active .step-indicator { border-color: var(--accent); color: var(--accent); }
.step.done  .step-indicator  { border-color: var(--color-success); background: var(--color-success); color: var(--bg-base); }

.spinner { display: inline-block; animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.check { font-size: 1rem; }

.step-content { flex: 1; padding-top: 0.3rem; }
.step-name {
  font-family: var(--font-body);
  font-size: 0.88rem;
  font-weight: 500;
  color: var(--text-primary);
}
.step-message {
  font-family: var(--font-body);
  font-size: 0.8rem;
  color: var(--text-muted);
  margin-top: 0.2rem;
}

.progress-bar-container { margin: 1.25rem 0; }
.progress-bar { height: 6px; background: var(--bg-elevated); border-radius: 3px; overflow: hidden; }
.progress-fill {
  height: 100%;
  background: var(--accent);
  border-radius: 3px;
  transition: width 0.3s ease;
}

/* Success and error */
.modal-content.success, .modal-content.error { text-align: center; }

.success-icon, .error-icon { font-size: 2.5rem; margin: 0.75rem 0; }
.success-icon { color: var(--color-success); }
.error-icon   { color: var(--color-error); }

.spinner-large {
  font-size: 2.5rem;
  text-align: center;
  margin: 0.75rem 0;
  animation: spin 1s linear infinite;
  color: var(--accent);
}

.countdown {
  font-family: var(--font-body);
  font-size: 0.82rem;
  color: var(--text-muted);
}

.error-detail {
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  padding: 0.85rem 1rem;
  margin: 1rem 0;
  text-align: left;
  max-height: 200px;
  overflow-y: auto;
}
.error-detail pre {
  margin: 0;
  font-family: var(--font-mono);
  font-size: 0.78rem;
  color: var(--text-secondary);
  word-wrap: break-word;
  white-space: pre-wrap;
}
</style>
