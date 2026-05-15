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

  // Call the update API
  const token = localStorage.getItem('arcvault_token')
  fetch('/api/update/apply', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    }
  }).catch(err => {
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
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal {
  background: #1e1e2e;
  border: 1px solid #333;
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.5);
  max-width: 500px;
  max-height: 90vh;
  overflow-y: auto;
  color: #fff;
}

.modal-content {
  padding: 2rem;
}

.modal-content h2 {
  margin: 0 0 1.5rem 0;
  font-size: 1.5rem;
}

.modal-content p {
  margin: 0.5rem 0;
  line-height: 1.6;
  color: #ccc;
}

.version-info {
  background: #111;
  border: 1px solid #333;
  border-radius: 4px;
  padding: 1rem;
  margin: 1rem 0;
  font-family: monospace;
  font-size: 0.9rem;
}

.warning {
  background: #3c2c2c;
  border-left: 3px solid #f39c12;
  padding: 1rem;
  border-radius: 4px;
  margin: 1rem 0;
  font-size: 0.95rem;
}

.warning strong {
  display: block;
  margin-bottom: 0.5rem;
}

.warning p {
  margin: 0;
}

.release-link {
  text-align: center;
  margin: 1rem 0;
}

.release-link a {
  color: #4f8ef7;
  text-decoration: none;
  font-size: 0.9rem;
}

.release-link a:hover {
  text-decoration: underline;
}

.modal-actions {
  display: flex;
  gap: 1rem;
  margin-top: 1.5rem;
  justify-content: flex-end;
}

.btn-cancel,
.btn-update,
.btn-close {
  padding: 0.6rem 1.5rem;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.95rem;
  font-weight: 500;
}

.btn-cancel {
  background: #333;
  color: #ccc;
}

.btn-cancel:hover {
  background: #444;
}

.btn-update,
.btn-close {
  background: #4f8ef7;
  color: #fff;
}

.btn-update:hover,
.btn-close:hover {
  background: #3a7fd6;
}

/* Progress state styles */
.steps-container {
  margin: 1.5rem 0;
}

.step {
  display: flex;
  gap: 1rem;
  margin: 1rem 0;
  align-items: flex-start;
}

.step-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 2px solid #444;
  border-radius: 50%;
  color: #aaa;
  font-weight: bold;
  flex-shrink: 0;
}

.step.active .step-indicator {
  border-color: #4f8ef7;
  color: #4f8ef7;
}

.step.done .step-indicator {
  border-color: #4caf50;
  background: #4caf50;
  color: #fff;
}

.spinner {
  display: inline-block;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.check {
  font-size: 1.2rem;
}

.step-content {
  flex: 1;
  padding-top: 0.25rem;
}

.step-name {
  font-weight: 500;
  color: #fff;
}

.step-message {
  font-size: 0.85rem;
  color: #888;
  margin-top: 0.25rem;
}

.progress-bar-container {
  margin: 1.5rem 0;
}

.progress-bar {
  height: 8px;
  background: #333;
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #4f8ef7, #3a7fd6);
  transition: width 0.3s ease;
}

/* Success and error styles */
.modal-content.success {
  text-align: center;
}

.modal-content.error {
  text-align: center;
}

.success-icon,
.error-icon {
  font-size: 3rem;
  margin: 1rem 0;
}

.success-icon {
  color: #4caf50;
}

.error-icon {
  color: #f44336;
}

.spinner-large {
  font-size: 3rem;
  text-align: center;
  margin: 1rem 0;
  animation: spin 1s linear infinite;
  color: #4f8ef7;
}

.countdown {
  font-size: 0.85rem;
  color: #888;
}

.error-detail {
  background: #111;
  border: 1px solid #333;
  border-radius: 4px;
  padding: 1rem;
  margin: 1rem 0;
  text-align: left;
  max-height: 200px;
  overflow-y: auto;
}

.error-detail pre {
  margin: 0;
  font-family: monospace;
  font-size: 0.8rem;
  color: #aaa;
  word-wrap: break-word;
  white-space: pre-wrap;
}
</style>
