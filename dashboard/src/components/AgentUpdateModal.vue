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

  const token = localStorage.getItem('arcvault_token')
  fetch(`/api/agents/${props.agentId}/update`, {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${token}` }
  })
    .then(r => {
      if (!r.ok) return r.json().then(b => Promise.reject(b.error || `HTTP ${r.status}`))
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
  display: flex; justify-content: center; align-items: center;
  z-index: 1000;
}

.modal {
  background: #1e1e2e;
  border: 1px solid #333;
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0,0,0,0.5);
  width: 480px;
  max-height: 90vh;
  overflow-y: auto;
  color: #fff;
}

.modal-content { padding: 2rem; }
.modal-content.center { text-align: center; }

h2 { margin: 0 0 1.5rem; font-size: 1.4rem; }
p  { margin: 0.4rem 0; color: #ccc; line-height: 1.6; }

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
  font-size: 0.9rem;
}
.warning strong { display: block; margin-bottom: 0.4rem; }
.warning p { margin: 0; }

.modal-actions { display: flex; gap: 1rem; justify-content: flex-end; margin-top: 1.5rem; }
.center-actions { justify-content: center; }

.btn-cancel, .btn-update, .btn-close {
  padding: 0.6rem 1.4rem;
  border: none; border-radius: 4px;
  cursor: pointer; font-size: 0.95rem; font-weight: 500;
}
.btn-cancel { background: #333; color: #ccc; }
.btn-cancel:hover { background: #444; }
.btn-update, .btn-close { background: #4f8ef7; color: #fff; }
.btn-update:hover, .btn-close:hover { background: #3a7fd6; }

/* Steps */
.steps-container { margin: 1.5rem 0; }
.step { display: flex; gap: 1rem; margin: 1rem 0; align-items: flex-start; }

.step-indicator {
  display: flex; align-items: center; justify-content: center;
  width: 32px; height: 32px;
  border: 2px solid #444; border-radius: 50%;
  color: #aaa; font-weight: bold; flex-shrink: 0;
}
.step.active .step-indicator { border-color: #4f8ef7; color: #4f8ef7; }
.step.done .step-indicator  { border-color: #4caf50; background: #4caf50; color: #fff; }

.spinner { animation: spin 1s linear infinite; display: inline-block; }
@keyframes spin { to { transform: rotate(360deg); } }
.check { font-size: 1.2rem; }

.step-name { font-weight: 500; color: #fff; padding-top: 0.25rem; }

/* Progress bar */
.progress-bar-container { margin: 1rem 0; }
.progress-bar { height: 8px; background: #333; border-radius: 4px; overflow: hidden; }
.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #4f8ef7, #3a7fd6);
  transition: width 0.3s ease;
}

/* Success / error */
.success-icon { font-size: 3rem; color: #4caf50; margin: 1rem 0; }
.error-icon   { font-size: 3rem; color: #f44336; margin: 1rem 0; }

.spinner-large {
  font-size: 3rem; margin: 1rem 0;
  animation: spin 1s linear infinite;
  color: #4f8ef7;
}

.countdown { font-size: 0.85rem; color: #888; }
</style>
