<template>
  <div class="modal-overlay" @click.self="onOverlayClick">
    <div class="modal">
      <div class="modal-header">
        <h3>{{ title }}</h3>
        <button class="close-btn" @click="$emit('close')" :disabled="state === 'progress'">×</button>
      </div>

      <!-- Confirm state -->
      <div v-if="state === 'confirm'" class="modal-body">
        <p>{{ confirmMessage }}</p>
        <div class="modal-actions">
          <button class="btn btn-secondary" @click="$emit('close')">Cancel</button>
          <button class="btn btn-danger" @click="startRollback">Rollback</button>
        </div>
      </div>

      <!-- Progress state -->
      <div v-else-if="state === 'progress'" class="modal-body">
        <div class="progress-container">
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: progress + '%' }"></div>
          </div>
          <span class="progress-pct">{{ progress }}%</span>
        </div>
        <p class="progress-log">{{ currentMessage }}</p>
      </div>

      <!-- Success state -->
      <div v-else-if="state === 'success'" class="modal-body">
        <div class="status-icon success">✓</div>
        <p>{{ successMessage }}</p>
        <div class="modal-actions">
          <button class="btn btn-primary" @click="$emit('close')">Done</button>
        </div>
      </div>

      <!-- Error state -->
      <div v-else-if="state === 'error'" class="modal-body">
        <div class="status-icon error">✗</div>
        <p>{{ errorText }}</p>
        <div class="modal-actions">
          <button class="btn btn-secondary" @click="$emit('close')">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { applyRollback, applyAgentRollback } from '../api.js'

const props = defineProps({
  // 'coordinator' or 'agent'
  target: {
    type: String,
    required: true
  },
  agentId: {
    type: String,
    default: null
  }
})

const emit = defineEmits(['close', 'complete'])

const state = ref('confirm')
const progress = ref(0)
const currentMessage = ref('')
const errorText = ref('')

const title = computed(() =>
  props.target === 'coordinator' ? 'Rollback Coordinator' : `Rollback Agent: ${props.agentId}`
)

const confirmMessage = computed(() =>
  props.target === 'coordinator'
    ? 'Roll back the coordinator to the previous version? The service will restart automatically.'
    : `Roll back agent "${props.agentId}" to its previous version? The agent will restart automatically.`
)

const successMessage = computed(() =>
  props.target === 'coordinator'
    ? 'Coordinator rolled back successfully. Service has restarted.'
    : `Agent "${props.agentId}" rolled back successfully. It will reconnect shortly.`
)

function onOverlayClick() {
  if (state.value !== 'progress') emit('close')
}

async function startRollback() {
  if (props.target === 'agent') {
    // Agent rollback: fire-and-forget POST, no WebSocket stream
    state.value = 'progress'
    progress.value = 50
    currentMessage.value = 'Sending rollback command to agent…'
    try {
      await applyAgentRollback(props.agentId)
      progress.value = 100
      currentMessage.value = 'Command sent.'
      state.value = 'success'
      emit('complete')
    } catch (err) {
      errorText.value = err.message || 'Rollback command failed.'
      state.value = 'error'
    }
    return
  }

  // Coordinator rollback: WebSocket progress stream (same as UpdateModal)
  state.value = 'progress'
  progress.value = 0
  currentMessage.value = 'Initiating rollback…'

  // Trigger the rollback via REST; progress arrives over WebSocket
  try {
    await applyRollback()
  } catch (err) {
    errorText.value = err.message || 'Failed to initiate rollback.'
    state.value = 'error'
    return
  }

  // Connect to the same admin WebSocket used for coordinator updates
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  const ws = new WebSocket(`${proto}://${location.host}/ws/admin`)

  ws.onmessage = (evt) => {
    let msg
    try { msg = JSON.parse(evt.data) } catch { return }

    if (msg.type !== 'rollback_progress') return

    if (msg.percent !== undefined) progress.value = msg.percent
    if (msg.message) currentMessage.value = msg.message

    if (msg.status === 'done') {
      ws.close()
      state.value = 'success'
      emit('complete')
    } else if (msg.status === 'error') {
      ws.close()
      errorText.value = msg.message || 'Rollback failed.'
      state.value = 'error'
    }
  }

  ws.onerror = () => {
    errorText.value = 'WebSocket connection lost.'
    state.value = 'error'
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: var(--bg-card, #1e2130);
  border: 1px solid var(--border, #2a2d3e);
  border-radius: 8px;
  width: 420px;
  max-width: 95vw;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border, #2a2d3e);
}

.modal-header h3 {
  margin: 0;
  font-size: 1rem;
  color: var(--text-primary, #e8eaf6);
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-secondary, #9091a0);
  font-size: 1.4rem;
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
}
.close-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.close-btn:not(:disabled):hover { color: var(--text-primary, #e8eaf6); }

.modal-body {
  padding: 24px 20px 20px;
}

.modal-body p {
  margin: 0 0 16px;
  color: var(--text-secondary, #9091a0);
  font-size: 0.9rem;
  line-height: 1.5;
}

.modal-actions {
  display: flex;
  gap: 10px;
  justify-content: flex-end;
  margin-top: 20px;
}

/* Progress */
.progress-container {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.progress-bar {
  flex: 1;
  height: 6px;
  background: var(--bg-surface, #12141f);
  border-radius: 3px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--accent, #4f8ef7);
  border-radius: 3px;
  transition: width 0.3s ease;
}

.progress-pct {
  font-size: 0.8rem;
  color: var(--text-secondary, #9091a0);
  min-width: 36px;
  text-align: right;
}

.progress-log {
  font-size: 0.82rem;
  color: var(--text-secondary, #9091a0);
  font-family: monospace;
  margin: 0;
  min-height: 1.4em;
}

/* Status icons */
.status-icon {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.3rem;
  margin: 0 auto 16px;
}
.status-icon.success { background: rgba(72, 199, 142, 0.15); color: #48c78e; }
.status-icon.error   { background: rgba(241, 70, 104, 0.15);  color: #f14668; }

/* Buttons */
.btn {
  padding: 8px 18px;
  border: none;
  border-radius: 5px;
  font-size: 0.88rem;
  cursor: pointer;
  transition: opacity 0.15s;
}
.btn:hover { opacity: 0.85; }
.btn-primary   { background: var(--accent, #4f8ef7); color: #fff; }
.btn-secondary { background: var(--bg-surface, #12141f); color: var(--text-primary, #e8eaf6); border: 1px solid var(--border, #2a2d3e); }
.btn-danger    { background: #f14668; color: #fff; }
</style>
