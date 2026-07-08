<template>
  <div v-if="isOpen" class="modal-overlay" @click.self="close">
    <div class="modal">

      <!-- Confirm -->
      <div v-if="state === 'confirm'" class="modal-content">
        <h2>Update Agent</h2>
        <div class="version-info">
          <p><strong>Agent:</strong> {{ agentId }}</p>
          <p><strong>Current:</strong> {{ agentVersion }}</p>
          <p><strong>New:</strong> {{ updateStore.latest }}</p>
        </div>
        <div class="warning">
          <strong>⚠ The agent service will restart during the update.</strong>
          <p>In-progress jobs on this agent may need to be rescheduled.</p>
        </div>
        <div class="modal-actions">
          <button class="btn-cancel" @click="close">Cancel</button>
          <button class="btn-update" @click="startUpdate">Update now</button>
        </div>
      </div>

      <!-- Progress -->
      <div v-else-if="state === 'progress'" class="modal-content">
        <h2>Updating Agent</h2>
        <div class="steps-container">
          <div
            v-for="step in steps"
            :key="step.name"
            class="step"
            :class="{ active: currentStep === step.name, done: isStepDone(step.name) }"
          >
            <div class="step-indicator">
              <span v-if="isStepDone(step.name)" class="check">✓</span>
              <span v-else-if="currentStep === step.name" class="spinner">⟳</span>
              <span v-else class="number">{{ steps.indexOf(step) + 1 }}</span>
            </div>
            <div class="step-content">
              <div class="step-name">{{ step.label }}</div>
            </div>
          </div>
        </div>
        <div v-if="currentStep === 'downloading'" class="progress-bar-container">
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: progressPct + '%' }"></div>
          </div>
        </div>
      </div>

      <!-- Reconnecting -->
      <div v-else-if="state === 'reconnecting'" class="modal-content center">
        <div class="spinner-large">⟳</div>
        <h2>Agent Restarting</h2>
        <p>Waiting for <strong>{{ agentId }}</strong> to reconnect...</p>
        <p v-if="countdown > 0" class="countdown">{{ countdown }}s remaining</p>
      </div>

      <!-- Success -->
      <div v-else-if="state === 'success'" class="modal-content center">
        <div class="success-icon">✓</div>
        <h2>Agent Updated</h2>
        <p><strong>{{ agentId }}</strong> is back online at <strong>{{ updateStore.latest }}</strong>.</p>
        <div class="modal-actions center-actions">
          <button class="btn-close" @click="close">Close</button>
        </div>
      </div>

      <!-- Error -->
      <div v-else-if="state === 'error'" class="modal-content center">
        <div class="error-icon">✕</div>
        <h2>Update Failed</h2>
        <p>{{ errorMessage }}</p>
        <div class="modal-actions center-actions">
          <button class="btn-close" @click="close">Close</button>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup>
import { ref, inject, watch, onUnmounted } from 'vue'
import { getToken, applyAgentUpdate } from '../api'
import { useAuth } from '../composables/useAuth.js'

const props = defineProps({
  isOpen: Boolean,
  agentId: String,
  agentVersion: String,
  agents: Array,     // live agents list from parent, used for reconnect polling
  lastEvent: Object
})

const emit = defineEmits(['close'])

const updateStore = inject('updateStore', {
  current: '',
  latest: '',
  available: false,
  releaseUrl: ''
})

const state = ref('confirm')
const currentStep = ref('downloading')
const progressPct = ref(0)
const errorMessage = ref('')
const countdown = ref(60)

const steps = [
  { name: 'downloading', label: 'Downloading binary' },
  { name: 'verifying',   label: 'Verifying binary' },
  { name: 'staging',     label: 'Staging binary' },
  { name: 'restarting',  label: 'Restarting service' }
]

let pollTimer = null
let countdownTimer = null

// Watch WebSocket events — filter by agent_id so only this agent's events land here.
watch(() => props.lastEvent, (evt) => {
  if (!evt) return
  if (evt.type !== 'update_progress') return

  // The coordinator broadcasts the raw message the agent sent, which is already
  // wrapped: { type: "update_progress", payload: { agent_id, step, pct } }
  // or it may be the payload directly — handle both shapes.
  const data = evt.payload ?? evt
  if (data.agent_id !== props.agentId) return

  currentStep.value = data.step
  progressPct.value = data.pct ?? 0

  if (data.step === 'restarting' && data.pct === 95) {
    state.value = 'reconnecting'
    startReconnectPolling()
  } else if (data.step === 'error') {
    state.value = 'error'
    errorMessage.value = data.message || 'An unknown error occurred.'
    clearTimers()
  }
})

// Also poll agents list for reconnect once we're in reconnecting state.
watch(() => props.agents, (agents) => {
  if (state.value !== 'reconnecting') return
  const agent = agents?.find(a => a.id === props.agentId)
  if (agent?.status === 'online') {
    clearTimers()
    state.value = 'success'
  }
}, { deep: true })

function startUpdate() {
  state.value = 'progress'
  progressPct.value = 0
  currentStep.value = 'downloading'

  const token = getToken()
  if (!token) {
    state.value = 'error'
    errorMessage.value = 'Missing authentication token. Please sign in again.'
    return
  }

  const auth = useAuth()

  console.log('Refreshing token before agent update...')

  // Refresh token to ensure it's valid, then apply update
  auth.refreshToken()
    .then(success => {
      if (!success) {
        throw new Error('Token refresh failed')
      }
      console.log('Token refreshed, applying agent update for', props.agentId)
      return applyAgentUpdate(props.agentId)
    })
    .then(() => {
      console.log('Agent update request accepted for', props.agentId)
    })
    .catch(err => {
      state.value = 'error'
      errorMessage.value = String(err)
    })
}

function startReconnectPolling() {
  countdown.value = 60
  countdownTimer = setInterval(() => {
    if (--countdown.value <= 0) clearInterval(countdownTimer)
  }, 1000)

  pollTimer = setTimeout(() => {
    if (state.value === 'reconnecting') {
      state.value = 'error'
      errorMessage.value = 'Agent may still be restarting. Refresh in a moment.'
    }
  }, 60_000)
}

function isStepDone(stepName) {
  const cur = steps.findIndex(s => s.name === currentStep.value)
  const idx = steps.findIndex(s => s.name === stepName)
  return idx < cur
}

function close() {
  state.value = 'confirm'
  clearTimers()
  emit('close')
}

function clearTimers() {
  clearInterval(countdownTimer)
  clearTimeout(pollTimer)
  countdownTimer = null
  pollTimer = null
}

onUnmounted(clearTimers)
</script>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,0.7);
  backdrop-filter: blur(4px);
  display: flex; justify-content: center; align-items: center;
  z-index: 1000;
}

.modal {
  background: var(--bg-elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-card);
  box-shadow: var(--shadow-lg), var(--edge-highlight);
  animation: modal-pop 0.18s ease-out;
  width: 480px;
  max-width: 95vw;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-content { padding: 1.75rem 2rem; }
.modal-content.center { text-align: center; }

h2 {
  margin: 0 0 1.25rem;
  font-family: var(--font-display);
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--text-primary);
}
p  { margin: 0.4rem 0; color: var(--text-secondary); line-height: 1.6; font-family: var(--font-body); font-size: 0.9rem; }

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

.modal-actions { display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem; }
.center-actions { justify-content: center; }

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
  transition: filter 0.15s;
}
.btn-cancel {
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
}
.btn-cancel:hover { background: var(--bg-elevated); color: var(--text-primary); border-color: var(--border-strong); }
.btn-update, .btn-close { background: var(--accent); color: var(--bg-base); }
.btn-update:hover, .btn-close:hover { filter: brightness(1.1); }

/* Steps */
.steps-container { margin: 1.25rem 0; }
.step { display: flex; gap: 1rem; margin: 0.85rem 0; align-items: flex-start; }

.step-indicator {
  display: flex; align-items: center; justify-content: center;
  width: 30px; height: 30px;
  border: 2px solid var(--border-strong);
  border-radius: 50%;
  color: var(--text-muted);
  font-family: var(--font-body);
  font-size: 0.78rem;
  font-weight: 700;
  flex-shrink: 0;
}
.step.active .step-indicator { border-color: var(--accent); color: var(--accent); }
.step.done .step-indicator   { border-color: var(--color-success); background: var(--color-success); color: var(--bg-base); }

.spinner { animation: spin 1s linear infinite; display: inline-block; }
@keyframes spin { to { transform: rotate(360deg); } }
.check { font-size: 1rem; }

.step-name {
  font-family: var(--font-body);
  font-size: 0.88rem;
  font-weight: 500;
  color: var(--text-primary);
  padding-top: 0.3rem;
}

/* Progress bar */
.progress-bar-container { margin: 1rem 0; }
.progress-bar { height: 6px; background: var(--bg-elevated); border-radius: 3px; overflow: hidden; }
.progress-fill {
  height: 100%;
  background: var(--accent);
  border-radius: 3px;
  transition: width 0.3s ease;
}

/* Success / error icons */
.success-icon {
  font-size: 2.5rem;
  color: var(--color-success);
  margin: 0.75rem 0;
}
.error-icon {
  font-size: 2.5rem;
  color: var(--color-error);
  margin: 0.75rem 0;
}

.spinner-large {
  font-size: 2.5rem;
  margin: 0.75rem 0;
  animation: spin 1s linear infinite;
  color: var(--accent);
}

.countdown {
  font-family: var(--font-body);
  font-size: 0.82rem;
  color: var(--text-muted);
}
</style>
