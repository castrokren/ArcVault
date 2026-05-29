<template>
  <teleport to="body">
    <div v-if="isOpen" class="modal-overlay" @click.self="close">
      <div class="modal-container">
        <!-- Header -->
        <div class="modal-header">
          <div class="header-content">
            <div class="job-meta">
              <h2>{{ job.name }}</h2>
              <div class="meta-row">
                <span class="meta-label">Job ID:</span>
                <span class="mono">{{ job.id }}</span>
              </div>
              <div class="meta-row">
                <span class="meta-label">Agent:</span>
                <span>{{ job.agent_id || job.group_id }}</span>
              </div>
              <div class="meta-row">
                <span class="meta-label">Status:</span>
                <span :class="['status-badge', status]">{{ status }}</span>
              </div>
            </div>
            <div class="header-actions">
              <button @click="downloadLogs" class="btn-download">⬇ Download</button>
              <button @click="close" class="btn-close">✕</button>
            </div>
          </div>
        </div>

        <!-- Logs Container -->
        <div class="logs-container">
          <div v-if="loading" class="logs-loading">
            <div class="spinner"></div>
            <span>Loading logs...</span>
          </div>
          <div v-else-if="logs.length === 0" class="logs-empty">
            No logs available
          </div>
          <div v-else class="logs-list">
            <div
              v-for="(log, idx) in logs"
              :key="idx"
              class="log-line"
              :class="{ streaming: isStreamingIndex(idx) }"
            >
              <span class="log-num">{{ startIndex + idx + 1 }}</span>
              <span class="log-text">{{ log }}</span>
            </div>
            <div v-if="isLiveUpdating" class="live-indicator">
              <span class="pulse"></span>
              Live updates
            </div>
          </div>
        </div>

        <!-- Pagination & Info -->
        <div class="modal-footer">
          <div class="pagination-info">
            <span v-if="logs.length > 0">
              Showing {{ startIndex + 1 }}–{{ startIndex + logs.length }} of {{ totalLogs }} logs
            </span>
            <span v-else>0 logs</span>
          </div>

          <div v-if="totalPages > 1" class="pagination-controls">
            <button
              @click="prevPage"
              :disabled="currentPage === 1"
              class="pag-btn"
            >
              ← Prev
            </button>
            <div class="page-indicator">
              Page {{ currentPage }} of {{ totalPages }}
            </div>
            <button
              @click="nextPage"
              :disabled="currentPage === totalPages"
              class="pag-btn"
            >
              Next →
            </button>
          </div>
        </div>
      </div>
    </div>
  </teleport>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  job: {
    type: Object,
    required: true,
  },
  isOpen: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['close'])

const currentPage = ref(1)
const logsPerPage = 50
const logs = ref([])
const totalLogs = ref(0)
const totalPages = ref(0)
const loading = ref(false)
const streamingIndices = ref(new Set())
const isLiveUpdating = ref(false)
const ws = ref(null)

const status = computed(() => {
  // Get status from latest progress or job status
  return 'running' // This will be updated via WebSocket
})

const startIndex = computed(() => (currentPage.value - 1) * logsPerPage)

const isStreamingIndex = (idx) => streamingIndices.value.has(idx)

async function loadLogs() {
  loading.value = true
  try {
    const response = await fetch(
      `/api/jobs/${props.job.id}/logs?page=${currentPage.value}&limit=${logsPerPage}`
    )
    if (!response.ok) throw new Error(`HTTP ${response.status}`)

    const data = await response.json()
    logs.value = data.data || []
    totalLogs.value = data.total || 0
    totalPages.value = data.pages || 0
    streamingIndices.value.clear()
  } catch (error) {
    console.error('Failed to load logs:', error)
    logs.value = []
    totalLogs.value = 0
  } finally {
    loading.value = false
  }
}

function nextPage() {
  if (currentPage.value < totalPages.value) {
    currentPage.value++
  }
}

function prevPage() {
  if (currentPage.value > 1) {
    currentPage.value--
  }
}

function connectWebSocket() {
  try {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws`
    ws.value = new WebSocket(wsUrl)

    ws.value.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data)

        // Listen for progress events that include new logs
        if (message.type === 'progress' && message.job_id === props.job.id) {
          isLiveUpdating.value = true
          setTimeout(() => {
            isLiveUpdating.value = false
          }, 2000)

          // If we're on the last page, reload to see new logs
          if (currentPage.value === totalPages.value && totalPages.value > 0) {
            loadLogs()
          } else if (totalLogs.value === 0) {
            // First load
            loadLogs()
          }
        }
      } catch (e) {
        console.error('WebSocket parse error:', e)
      }
    }

    ws.value.onerror = () => {
      console.warn('WebSocket error, will retry')
    }

    ws.value.onclose = () => {
      // Attempt to reconnect after 3 seconds
      setTimeout(connectWebSocket, 3000)
    }
  } catch (e) {
    console.error('WebSocket connection failed:', e)
  }
}

function downloadLogs() {
  const allLogs = logs.value.join('\n')
  const blob = new Blob([allLogs], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${props.job.id}-logs.txt`
  a.click()
  URL.revokeObjectURL(url)
}

function close() {
  emit('close')
}

watch(
  () => props.isOpen,
  (open) => {
    if (open) {
      currentPage.value = 1
      loadLogs()
      connectWebSocket()
    } else if (ws.value) {
      ws.value.close()
    }
  }
)

watch(currentPage, () => {
  loadLogs()
})

onUnmounted(() => {
  if (ws.value) {
    ws.value.close()
  }
})
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(2px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  animation: fadeIn 0.15s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.modal-container {
  background: #1e1e2e;
  border: 1px solid #333;
  border-radius: 8px;
  width: 90%;
  max-width: 1000px;
  height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.8);
  animation: slideUp 0.2s ease-out;
}

@keyframes slideUp {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

/* ===== HEADER ===== */
.modal-header {
  border-bottom: 1px solid #333;
  padding: 1.5rem;
  background: #16161e;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 2rem;
}

.job-meta h2 {
  margin: 0 0 1rem 0;
  font-size: 1.4rem;
  color: #fff;
}

.meta-row {
  display: flex;
  gap: 0.75rem;
  font-size: 0.9rem;
  margin-bottom: 0.5rem;
  align-items: center;
}

.meta-label {
  color: #888;
  font-weight: 500;
}

.mono {
  font-family: 'Courier New', monospace;
  color: #4f8ef7;
  font-size: 0.85rem;
}

.status-badge {
  display: inline-block;
  padding: 0.2rem 0.6rem;
  border-radius: 4px;
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: uppercase;
}

.status-badge.running {
  background: #1a2a3a;
  color: #4f8ef7;
}

.status-badge.completed {
  background: #1a3a1a;
  color: #4caf50;
}

.status-badge.failed {
  background: #3a1a1a;
  color: #e55;
}

.status-badge.pending {
  background: #2a2a1a;
  color: #f0b429;
}

.header-actions {
  display: flex;
  gap: 0.5rem;
}

.btn-download,
.btn-close {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9rem;
  transition: all 0.15s ease;
}

.btn-download {
  background: #4f8ef7;
  color: #fff;
}

.btn-download:hover {
  background: #3d75d1;
}

.btn-close {
  background: #333;
  color: #aaa;
  width: 36px;
  height: 36px;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-close:hover {
  background: #444;
  color: #fff;
}

/* ===== LOGS CONTAINER ===== */
.logs-container {
  flex: 1;
  overflow-y: auto;
  background: #0f0f1a;
  font-family: 'Courier New', 'Monaco', monospace;
  font-size: 0.85rem;
  line-height: 1.5;
  padding: 1rem;
  position: relative;
}

.logs-container::-webkit-scrollbar {
  width: 8px;
}

.logs-container::-webkit-scrollbar-track {
  background: transparent;
}

.logs-container::-webkit-scrollbar-thumb {
  background: #333;
  border-radius: 4px;
}

.logs-container::-webkit-scrollbar-thumb:hover {
  background: #444;
}

.logs-loading,
.logs-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #666;
  gap: 0.75rem;
}

.spinner {
  width: 20px;
  height: 20px;
  border: 2px solid #333;
  border-top: 2px solid #4f8ef7;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.logs-list {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.log-line {
  display: flex;
  gap: 1rem;
  padding: 0.4rem 0.75rem;
  border-left: 2px solid transparent;
  transition: all 0.15s ease;
  color: #d0d0d0;
}

.log-line:hover {
  background: rgba(79, 142, 247, 0.08);
  border-left-color: #4f8ef7;
}

.log-line.streaming {
  animation: streamIn 0.4s ease-out;
  background: rgba(76, 175, 80, 0.06);
}

@keyframes streamIn {
  from {
    opacity: 0;
    transform: translateX(-10px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

.log-num {
  color: #555;
  min-width: 3rem;
  text-align: right;
  flex-shrink: 0;
  user-select: none;
}

.log-text {
  flex: 1;
  word-break: break-word;
  white-space: pre-wrap;
  color: #d0d0d0;
}

.live-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 1rem;
  padding: 0.75rem;
  background: rgba(76, 175, 80, 0.1);
  border: 1px solid rgba(76, 175, 80, 0.3);
  border-radius: 4px;
  color: #4caf50;
  font-size: 0.8rem;
  justify-content: center;
}

.pulse {
  display: inline-block;
  width: 6px;
  height: 6px;
  background: #4caf50;
  border-radius: 50%;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

/* ===== FOOTER ===== */
.modal-footer {
  border-top: 1px solid #333;
  padding: 1rem 1.5rem;
  background: #16161e;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}

.pagination-info {
  color: #888;
  font-size: 0.9rem;
}

.pagination-controls {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.page-indicator {
  color: #aaa;
  font-size: 0.9rem;
  min-width: 120px;
  text-align: center;
}

.pag-btn {
  padding: 0.4rem 0.8rem;
  background: #333;
  color: #aaa;
  border: 1px solid #444;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.85rem;
  transition: all 0.15s ease;
}

.pag-btn:hover:not(:disabled) {
  background: #404040;
  border-color: #555;
  color: #fff;
}

.pag-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
