<template>
  <div>
    <div class="page-header">
      <h1>Jobs</h1>
      <button @click="showForm = !showForm">{{ showForm ? 'Cancel' : '+ New Job' }}</button>
      <button @click="load" :disabled="loading">{{ loading ? 'Loading...' : 'Refresh' }}</button>
    </div>

    <div v-if="showForm" class="form-card">
      <h3>Create Job</h3>
      <div class="form-grid">
        <label>Agent ID</label>
        <input v-model="form.agent_id" placeholder="agent-01" />
        <label>Name</label>
        <input v-model="form.name" placeholder="nightly-backup" />
        <label>Source Path</label>
        <input v-model="form.source_path" placeholder="C:\src" />
        <label>Dest Path</label>
        <input v-model="form.dest_path" placeholder="D:\backup" />
        <label>Schedule <span class="optional">(optional)</span></label>
        <input v-model="form.schedule" placeholder="0 2 * * *" />
      </div>
      <div class="form-actions">
        <button class="primary" @click="createJob" :disabled="creating">
          {{ creating ? 'Creating...' : 'Create' }}
        </button>
      </div>
      <div v-if="formError" class="error">{{ formError }}</div>
    </div>

    <div v-if="error" class="error">{{ error }}</div>
    <div v-if="jobs.length === 0 && !loading" class="empty">No jobs found.</div>

    <table v-else class="table">
      <thead>
        <tr>
          <th>ID</th>
          <th>Name</th>
          <th>Agent</th>
          <th>Source</th>
          <th>Dest</th>
          <th>Status</th>
          <th>Created</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="job in jobs" :key="job.id">
          <td class="mono">{{ job.id }}</td>
          <td>{{ job.name }}</td>
          <td class="mono">{{ job.agent_id }}</td>
          <td class="mono">{{ job.source_path }}</td>
          <td class="mono">{{ job.dest_path }}</td>
          <td><span class="badge" :class="job.status">{{ job.status }}</span></td>
          <td>{{ formatDate(job.created_at) }}</td>
          <td>
            <button class="danger-sm" @click="removeJob(job.id)">Delete</button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { getJobs, createJob as apiCreateJob, deleteJob } from '../api.js'

const props = defineProps(['lastEvent'])
const jobs = ref([])
const loading = ref(false)
const error = ref(null)
const showForm = ref(false)
const creating = ref(false)
const formError = ref(null)

const form = ref({ agent_id: '', name: '', source_path: '', dest_path: '', schedule: '' })

async function load() {
  loading.value = true
  error.value = null
  try {
    jobs.value = await getJobs()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function createJob() {
  formError.value = null
  creating.value = true
  try {
    const payload = { ...form.value }
    if (!payload.schedule) delete payload.schedule
    await apiCreateJob(payload)
    form.value = { agent_id: '', name: '', source_path: '', dest_path: '', schedule: '' }
    showForm.value = false
    await load()
  } catch (e) {
    formError.value = e.message
  } finally {
    creating.value = false
  }
}

async function removeJob(id) {
  if (!confirm('Delete this job?')) return
  try {
    await deleteJob(id)
    await load()
  } catch (e) {
    error.value = e.message
  }
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleString()
}

// refresh on job events from WebSocket
watch(() => props.lastEvent, (ev) => {
  if (ev?.type === 'job.updated' || ev?.type === 'job.result') load()
})

onMounted(load)
</script>

<style scoped>
.page-header { display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1.5rem; }
.page-header h1 { margin: 0; flex: 1; }
.page-header button {
  padding: 0.4rem 1rem;
  background: #4f8ef7;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.form-card {
  background: #1e1e2e;
  border: 1px solid #333;
  border-radius: 8px;
  padding: 1.5rem;
  margin-bottom: 1.5rem;
}
.form-card h3 { margin: 0 0 1rem; }

.form-grid {
  display: grid;
  grid-template-columns: 160px 1fr;
  gap: 0.6rem 1rem;
  align-items: center;
  margin-bottom: 1rem;
}
.form-grid label { color: #aaa; font-size: 0.9rem; }
.form-grid input {
  padding: 0.4rem 0.6rem;
  border-radius: 4px;
  border: 1px solid #444;
  background: #111;
  color: #fff;
}
.optional { color: #666; font-size: 0.8rem; }

.form-actions { display: flex; gap: 0.5rem; }
button.primary {
  padding: 0.4rem 1.2rem;
  background: #4caf50;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.error { color: #e55; margin: 0.5rem 0; }
.empty { color: #888; }

.table { width: 100%; border-collapse: collapse; }
.table th, .table td {
  text-align: left;
  padding: 0.6rem 0.75rem;
  border-bottom: 1px solid #2a2a3e;
}
.table th { color: #888; font-weight: 600; font-size: 0.85rem; text-transform: uppercase; }
.mono { font-family: monospace; font-size: 0.85rem; }

.badge {
  display: inline-block;
  padding: 0.2rem 0.6rem;
  border-radius: 999px;
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: uppercase;
}
.badge.pending  { background: #2a2a1a; color: #f0b429; }
.badge.running  { background: #1a2a3a; color: #4f8ef7; }
.badge.completed { background: #1a3a1a; color: #4caf50; }
.badge.failed   { background: #3a1a1a; color: #e55; }

button.danger-sm {
  padding: 0.2rem 0.6rem;
  background: #3a1a1a;
  color: #e55;
  border: 1px solid #e55;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.8rem;
}
</style>
