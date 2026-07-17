<template>
  <div v-if="isOpen" class="modal-overlay" @click.self="close">
    <div class="modal">
      <div class="modal-header">
        <h2>Agent Token</h2>
        <button class="close-btn" @click="close" aria-label="Close">✕</button>
      </div>

      <div class="modal-body">
        <div v-if="loading" class="loading">Generating token…</div>
        <div v-else-if="error" class="error">{{ error }}</div>
        <div v-else class="token-display">
          <p class="label">Copy this token for the agent installer:</p>
          <div class="token-box">
            <code>{{ token }}</code>
            <button class="btn btn-sm btn-copy" @click="copyToken">
              {{ copied ? 'Copied!' : 'Copy' }}
            </button>
          </div>
          <p class="note">This token is scoped to agent <strong>{{ agentId }}</strong> and required for installation.</p>
        </div>
      </div>

      <div class="modal-footer">
        <button class="btn btn-primary" @click="close">Done</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { createAgentToken } from '../api'

const props = defineProps({
  isOpen: Boolean,
  agentId: String,
})

const emit = defineEmits(['close'])

const token = ref('')
const loading = ref(false)
const error = ref(null)
const copied = ref(false)

watch(() => props.isOpen, async (open) => {
  if (open) {
    token.value = ''
    error.value = null
    loading.value = true
    copied.value = false

    try {
      const result = await createAgentToken(props.agentId)
      token.value = result.token
    } catch (err) {
      error.value = err.message || 'Failed to generate token'
    } finally {
      loading.value = false
    }
  }
})

const copyToken = async () => {
  try {
    await navigator.clipboard.writeText(token.value)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch (err) {
    error.value = 'Failed to copy token'
  }
}

const close = () => {
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

.modal {
  background: var(--bg-primary);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  max-width: 500px;
  width: 90%;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid var(--border);
}

.modal-header h2 {
  margin: 0;
  font-size: 1.25rem;
}

.close-btn {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: var(--text-secondary);
  padding: 0;
  width: 2rem;
  height: 2rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.close-btn:hover {
  color: var(--text-primary);
}

.modal-body {
  padding: 1.5rem;
  flex: 1;
}

.loading,
.error {
  text-align: center;
  padding: 1rem;
}

.error {
  color: var(--error);
}

.token-display {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.label {
  margin: 0;
  font-size: 0.9rem;
  color: var(--text-secondary);
}

.token-box {
  display: flex;
  gap: 0.5rem;
  align-items: flex-start;
}

code {
  flex: 1;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 0.75rem;
  font-family: monospace;
  font-size: 0.85rem;
  word-break: break-all;
  line-height: 1.4;
}

.btn-copy {
  flex-shrink: 0;
  white-space: nowrap;
}

.note {
  margin: 0;
  font-size: 0.85rem;
  color: var(--text-secondary);
  padding: 0.75rem;
  background: var(--bg-secondary);
  border-radius: 4px;
}

.modal-footer {
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--border);
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}
</style>
