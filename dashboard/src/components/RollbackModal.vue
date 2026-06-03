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
import { applyRollback, applyAgentRollback } from '../api'

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
  background: rgba(0, 0, 0, 0.65);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: 10px;
  width: 420px;
  max-width: 95vw;
  box-shadow: var(--shadow-lg);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--border-subtle);
}

.modal-header h3 {
  margin: 0;
  font-family: var(--font-display);
  font-size: 1rem;
  font-weight: 700;
  color: var(--text-primary);
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-muted);
  font-size: 1.4rem;
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
  border-radius: 4px;
  transition: color 0.15s, background 0.15s;
}
.close-btn:disabled { opacity: 0.35; cursor: not-allowed; }
.close-btn:not(:disabled):hover { color: var(--text-primary); background: var(--bg-elevated); }

.modal-body {
  padding: 1.5rem 1.25rem 1.25rem;
  font-family: var(--font-body);
}

.modal-body p {
  margin: 0 0 1rem;
  color: var(--text-secondary);
  font-size: 0.88rem;
  line-height: 1.55;
}

.modal-actions {
  display: flex;
  gap: 0.65rem;
  justify-content: flex-end;
  margin-top: 1.25rem;
}

/* Progress */
.progress-container {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.progress-bar {
  flex: 1;
  height: 5px;
  background: var(--bg-surface);
  border-radius: 3px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--accent);
  border-radius: 3px;
  transition: width 0.3s ease;
}

.progress-pct {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  color: var(--text-secondary);
  min-width: 36px;
  text-align: right;
}

.progress-log {
  font-family: var(--font-mono);
  font-size: 0.8rem;
  color: var(--text-muted);
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
  margin: 0 auto 1rem;
}
.status-icon.success { background: var(--bg-success); color: var(--color-success); }
.status-icon.error   { background: var(--bg-error);   color: var(--color-error); }

/* Buttons */
.btn {
  display: inline-flex;
  align-items: center;
  padding: 0.4rem 1rem;
  border: none;
  border-radius: 5px;
  font-family: var(--font-body);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: filter 0.15s, background 0.15s;
}
.btn-primary   { background: var(--accent); color: var(--bg-base); }
.btn-primary:hover { filter: brightness(1.1); }
.btn-secondary {
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
}
.btn-secondary:hover { background: var(--bg-elevated); color: var(--text-primary); border-color: var(--border-strong); }
.btn-danger    { background: var(--color-error); color: var(--bg-base); }
.btn-danger:hover { filter: brightness(1.1); }
</style>
