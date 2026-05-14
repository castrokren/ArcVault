<template>
  <div>
    <div class="page-header">
      <h1>Job History</h1>
      <button @click="load" :disabled="loading">{{ loading ? 'Loading...' : 'Refresh' }}</button>
    </div>

    <div v-if="error" class="error">{{ error }}</div>
    <div v-if="runs.length === 0 && !loading" class="empty">No job runs yet.</div>

    <table v-else class="table">
      <thead>
        <tr>
          <th>Run ID</th>
          <th>Job</th>
          <th>Exit Code</th>
          <th>Output</th>
          <th>Finished</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="run in runs" :key="run.id">
          <td class="mono">{{ run.id }}</td>
          <td>
            <span class="mono">{{ run.job_id }}</span>
            <span v-if="jobName(run.job_id)" class="job-name"> — {{ jobName(run.job_id) }}</span>
          </td>
          <td>
            <span class="badge" :class="run.exit_code === 0 ? 'success' : 'fail'">
              {{ run.exit_code }}
            </span>
          </td>
          <td class="output">{{ run.output || '—' }}</td>
          <td>{{ formatDate(run.finished_at) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { getJobs } from '../api.js'

const props = defineProps(['lastEvent'])
const runs = ref([])
const jobs = ref([])
const loading = ref(false)
const error = ref(null)

const BASE_URL = 'http://localhost:443'

function authHeaders() {
  return { Authorization: `Bearer ${localStorage.getItem('arcvault_token')}` }
}

async function load() {
  loading.value = true
  error.value = null
  try {
    jobs.value = await getJobs()

    const results = await Promise.all(
      jobs.value.map(j =>
        fetch(`${BASE_URL}/api/jobs/${j.id}/runs`, { headers: authHeaders() })
          .then(r => r.ok ? r.json() : [])
          .catch(() => [])
      )
    )
    runs.value = results.flat().sort((a, b) =>
      new Date(b.finished_at) - new Date(a.finished_at)
    )
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function jobName(jobID) {
  return jobs.value.find(j => j.id === jobID)?.name || ''
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleString()
}

watch(() => props.lastEvent, (ev) => {
  if (ev?.type === 'job.result') load()
})

onMounted(load)
</script>

<style scoped>
.page-header { display: flex; align-items: center; gap: 1rem; margin-bottom: 1.5rem; }
.page-header h1 { margin: 0; flex: 1; }
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
.mono { font-family: monospace; font-size: 0.85rem; }
.job-name { color: #aaa; font-size: 0.85rem; }

.output {
  font-family: monospace;
  font-size: 0.8rem;
  max-width: 300px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: #aaa;
}

.badge {
  display: inline-block;
  padding: 0.2rem 0.6rem;
  border-radius: 999px;
  font-size: 0.8rem;
  font-weight: 600;
}
.badge.success { background: #1a3a1a; color: #4caf50; }
.badge.fail    { background: #3a1a1a; color: #e55; }
</style>
