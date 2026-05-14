<template>
  <div>
    <div class="page-header">
      <h1>Agents</h1>
      <button @click="load" :disabled="loading">{{ loading ? 'Loading...' : 'Refresh' }}</button>
    </div>

    <div v-if="error" class="error">{{ error }}</div>
    <div v-if="agents.length === 0 && !loading" class="empty">No agents registered.</div>

    <table v-else class="table">
      <thead>
        <tr>
          <th>ID</th>
          <th>Hostname</th>
          <th>OS</th>
          <th>Version</th>
          <th>Status</th>
          <th>Last Seen</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="agent in agents" :key="agent.id">
          <td class="mono">{{ agent.id }}</td>
          <td>{{ agent.hostname }}</td>
          <td>{{ agent.os }}</td>
          <td>{{ agent.version }}</td>
          <td>
            <span class="badge" :class="agent.status">{{ agent.status }}</span>
          </td>
          <td>{{ formatDate(agent.last_seen) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { getAgents } from '../api.js'

const props = defineProps(['lastEvent'])
const agents = ref([])
const loading = ref(false)
const error = ref(null)

async function load() {
  loading.value = true
  error.value = null
  try {
    agents.value = await getAgents()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleString()
}

// refresh on heartbeat or offline detection events
watch(() => props.lastEvent, (ev) => {
  if (ev?.type === 'agent.heartbeat' || ev?.type === 'agent.updated') load()
})

onMounted(load)
</script>

<style scoped>
.page-header { display: flex; align-items: center; gap: 1rem; margin-bottom: 1.5rem; }
.page-header h1 { margin: 0; }
.page-header button {
  padding: 0.4rem 1rem;
  background: #4f8ef7;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.error { color: #e55; margin-bottom: 1rem; }
.empty { color: #888; }

.table { width: 100%; border-collapse: collapse; }
.table th, .table td {
  text-align: left;
  padding: 0.6rem 0.75rem;
  border-bottom: 1px solid #2a2a3e;
}
.table th { color: #888; font-weight: 600; font-size: 0.85rem; text-transform: uppercase; }

.mono { font-family: monospace; font-size: 0.9rem; }

.badge {
  display: inline-block;
  padding: 0.2rem 0.6rem;
  border-radius: 999px;
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: uppercase;
}
.badge.online  { background: #1a3a1a; color: #4caf50; }
.badge.offline { background: #3a1a1a; color: #e55; }
</style>
