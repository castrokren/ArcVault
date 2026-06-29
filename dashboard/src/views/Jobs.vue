<template>
  <div>
    <div class="page-header">
      <h1>Jobs</h1>
      <button v-if="!selectedSite" @click="showForm = !showForm">{{ showForm ? 'Cancel' : '+ New Job' }}</button>
      <button @click="load" :disabled="loading">{{ loading ? 'Loading...' : 'Refresh' }}</button>
    </div>

    <!-- Stale data banner: federation site proxy data -->
    <div v-if="stale" class="stale-banner">
      ⚠ Site data last synced {{ fmtStaleTime(staleAsOf) }}. Data may be outdated.
    </div>
    <!-- Stale data banner: local coordinator sync lag -->
    <div v-if="!selectedSite && syncStale" class="stale-banner">
      ⚠ Federation sync in progress. Some job data may be outdated.
    </div>

    <div v-if="!selectedSite && showForm" class="form-card">
      <h3>Create Job</h3>

      <!-- Dispatch Mode Selector -->
      <div class="dispatch-mode">
        <label>Dispatch Mode</label>
        <div class="mode-buttons">
          <button
            class="mode-btn"
            :class="{ active: form.dispatchMode === 'agent' }"
            @click="form.dispatchMode = 'agent'"
          >
            Single Agent
          </button>
          <button
            class="mode-btn"
            :class="{ active: form.dispatchMode === 'group' }"
            @click="form.dispatchMode = 'group'"
          >
            Group
          </button>
        </div>
      </div>

      <div class="form-grid">
        <!-- Agent/Group Selection -->
        <label>{{ form.dispatchMode === 'agent' ? 'Agent' : 'Group' }}</label>
        <select v-if="form.dispatchMode === 'agent'" v-model="form.agent_id">
          <option value="">Select an agent...</option>
          <option v-for="agent in agents" :key="agent.id" :value="agent.id">
            {{ agent.name }} ({{ agent.id }})
          </option>
        </select>
        <select v-else v-model="form.group_id">
          <option value="">Select a group...</option>
          <option v-for="group in groups" :key="group.id" :value="group.id">
            {{ group.name }}
          </option>
        </select>

        <label>Path Authentication <span class="optional">(optional)</span></label>
        <select v-model="form.credential_profile_id" :disabled="form.dispatchMode === 'group' || !form.agent_id">
          <option value="">None</option>
          <option v-for="cred in filteredCredentials" :key="cred.id" :value="cred.id">
            {{ cred.name }} ({{ cred.type }})
          </option>
        </select>

        <label>Name</label>
        <input v-model="form.name" placeholder="nightly-backup" />
        <label>Source Path</label>
        <input v-model="form.source_path" placeholder="C:\src" />
        <label>Dest Path</label>
        <input v-model="form.dest_path" placeholder="D:\backup" />
        <label>Schedule <span class="optional">(optional)</span></label>
        <ScheduleBuilder v-model="form.schedule" />
      </div>
      <div class="sync-flags-row">
        <SyncFlagsBuilder v-model="form.sync_flags" />
      </div>
      <div class="form-actions">
        <button class="primary" @click="createJob" :disabled="creating">
          {{ creating ? 'Creating...' : 'Create' }}
        </button>
      </div>
      <div v-if="formError" class="error">{{ formError }}</div>
    </div>

    <div v-if="error" class="error">{{ error }}</div>

    <div v-if="!selectedSite" class="filters">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="Search jobs by name or agent ID..."
        class="search-input"
        @input="onFilterChange"
      />
      <div class="filter-chips">
        <button
          v-for="s in ['all', 'pending', 'running', 'completed', 'failed']"
          :key="s"
          class="chip"
          :class="{ active: statusFilter === s }"
          @click="setStatus(s)"
        >
          {{ s.charAt(0).toUpperCase() + s.slice(1) }}
        </button>
      </div>
    </div>

    <div v-if="jobs.length === 0 && !loading" class="empty">
      {{ selectedSite ? 'No jobs found for this site.' : (searchQuery || statusFilter !== 'all' ? 'No jobs match your search.' : 'No jobs found.') }}
    </div>

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
          <th v-if="!selectedSite"></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="job in jobs" :key="job.id">
          <td class="mono">{{ job.id }}</td>
          <td>{{ job.name }}</td>
          <td class="mono">{{ job.agent_id }}</td>
          <td class="mono">{{ job.source_path }}</td>
          <td class="mono">{{ job.dest_path }}</td>
          <td>
            <span class="badge" :class="job.status">{{ job.status }}</span>
          </td>
          <td>{{ formatDate(job.created_at) }}</td>
          <td v-if="!selectedSite">
            <button class="action-btn" @click="viewLogs(job.id)">Logs</button>
            <button class="danger-sm" @click="removeJob(job.id)">Delete</button>
          </td>
        </tr>
      </tbody>
    </table>

    <Pagination
      v-if="!selectedSite"
      :page="page"
      :pages="result.pages"
      :total="result.total"
      :limit="limit"
      @page-change="goToPage"
    />

    <!-- Logs Modal -->
    <div v-if="showLogsModal" class="modal-overlay" @click.self="showLogsModal = false">
      <div class="modal">
        <div class="modal-header">
          <h3>Job Logs: {{ logsJobName }}</h3>
          <button class="close-btn" @click="showLogsModal = false">✕</button>
        </div>
        <div class="modal-body">
          <div v-if="logsLoading" class="skeleton-group" aria-busy="true">
            <div class="skeleton skeleton-line" style="width: 55%"></div>
            <div class="skeleton skeleton-line" style="width: 80%"></div>
            <div class="skeleton skeleton-line" style="width: 40%"></div>
          </div>
          <div v-else-if="logsError" class="error">{{ logsError }}</div>
          <div v-else-if="!latestRun" class="empty">No runs found for this job.</div>
          <div v-else class="logs-container">
            <div class="run-info">
              <div><strong>Run ID:</strong> <span class="mono">{{ latestRun.id }}</span></div>
              <div><strong>Exit Code:</strong> <span :class="{ 'exit-success': latestRun.exit_code === 0, 'exit-fail': latestRun.exit_code !== 0 }">{{ latestRun.exit_code ?? '—' }}</span></div>
              <div><strong>Finished:</strong> {{ latestRun.finished_at ? formatDate(latestRun.finished_at) : '—' }}</div>
            </div>
            <pre class="logs-output">{{ latestRun.output || '(no output)' }}</pre>
          </div>
        </div>
        <div class="modal-footer">
          <button class="secondary" @click="showLogsModal = false">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, inject } from 'vue'
import { getJobs, createJob as apiCreateJob, deleteJob, getFederationJobs, getAgents, getGroups, getJobRuns, getToken } from '../api'
import { formatDate, fmtStaleTime } from '../utils/format.js'
import Pagination from '../components/Pagination.vue'
import ScheduleBuilder from '../components/ScheduleBuilder.vue'
import SyncFlagsBuilder from '../components/SyncFlagsBuilder.vue'
import { useFederationLag } from '../composables/useFederationLag.js'

const props = defineProps(['lastEvent'])

const selectedSite = inject('selectedSite', ref(null))

const result = ref({ data: [], total: 0, page: 1, pages: 0, limit: 25 })
const fedJobs = ref([])
const stale = ref(false)
const staleAsOf = ref(null)

// Sync lag: shows banner when local coordinator is behind peers (no site selected)
const { isStale: syncStale } = useFederationLag()

const jobs = computed(() => selectedSite.value ? fedJobs.value : (result.value.data || []))

const page = ref(1)
const limit = 25
const loading = ref(false)
const error = ref(null)
const showForm = ref(false)
const creating = ref(false)
const formError = ref(null)
const searchQuery = ref('')
const statusFilter = ref('all')

const agents = ref([])
const groups = ref([])
const credentials = ref([])
const filteredCredentials = ref([])

const form = ref({ dispatchMode: 'agent', agent_id: '', group_id: '', name: '', source_path: '', dest_path: '', schedule: '', credential_profile_id: '', sync_flags: {} })

// Logs modal state
const showLogsModal = ref(false)
const logsJobName = ref('')
const logsLoading = ref(false)
const logsError = ref(null)
const latestRun = ref(null)

async function load() {
  loading.value = true
  error.value = null
  stale.value = false

  try {
    if (selectedSite.value) {
      const data = await getFederationJobs(selectedSite.value)
      fedJobs.value = data.jobs || []
      stale.value = data.stale || false
      staleAsOf.value = data.as_of || null
    } else {
      result.value = await getJobs({
        page: page.value,
        limit,
        search: searchQuery.value,
        status: statusFilter.value === 'all' ? '' : statusFilter.value,
      })

    }
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function onFilterChange() {
  page.value = 1
  load()
}

function setStatus(s) {
  statusFilter.value = s
  page.value = 1
  load()
}

async function loadCredentials() {
  try {
    const response = await fetch('/api/credential-profiles', {
      headers: {
        Authorization: `Bearer ${getToken()}`,
      },
    })
    if (response.ok) {
      const data = await response.json()
      credentials.value = Array.isArray(data) ? data : []
      filterCredentials()
    }
  } catch (e) {
    console.error('Failed to load credentials:', e)
  }
}

function getAgentOS(agentId) {
  const agent = agents.value.find(a => a.id === agentId)
  return agent ? agent.os : ''
}

function filterCredentials() {
  const agentOS = getAgentOS(form.value.agent_id)
  if (!agentOS) {
    filteredCredentials.value = credentials.value
    return
  }

  // Filter by OS compatibility
  const osTypeMap = {
    'windows': ['SMB', 'AWS', 'Database'],
    'linux': ['SSH', 'AWS', 'Database'],
    'darwin': ['SSH', 'AWS', 'Database'],
  }

  const compatibleTypes = osTypeMap[agentOS.toLowerCase()] || []
  filteredCredentials.value = credentials.value.filter(c => compatibleTypes.includes(c.type))
}

function goToPage(n) {
  page.value = n
  load()
}

async function createJob() {
  formError.value = null

  // Validate dispatch mode
  if (form.value.dispatchMode === 'agent' && !form.value.agent_id) {
    formError.value = 'Please select an agent'
    return
  }
  if (form.value.dispatchMode === 'group' && !form.value.group_id) {
    formError.value = 'Please select a group'
    return
  }

  if (!form.value.name || !form.value.source_path || !form.value.dest_path) {
    formError.value = 'Please fill in all required fields'
    return
  }

  creating.value = true
  try {
    const payload = { ...form.value }
    // Remove the unused dispatch field
    delete payload.dispatchMode
    // Remove empty fields
    if (!payload.schedule) delete payload.schedule
    if (form.value.dispatchMode === 'agent') delete payload.group_id
    if (form.value.dispatchMode === 'group') delete payload.agent_id
    // Strip null/empty/false values from sync_flags
    if (payload.sync_flags) {
      const cleaned = {}
      for (const [key, value] of Object.entries(payload.sync_flags)) {
        const keep = value !== null && value !== false && value !== '' &&
          !(Array.isArray(value) && value.length === 0)
        if (keep) cleaned[key] = value
      }
      if (Object.keys(cleaned).length === 0) {
        delete payload.sync_flags
      } else {
        payload.sync_flags = cleaned
      }
    }

    await apiCreateJob(payload)
    form.value = { dispatchMode: 'agent', agent_id: '', group_id: '', name: '', source_path: '', dest_path: '', schedule: '', sync_flags: {} }
    showForm.value = false
    page.value = 1
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

async function viewLogs(jobId) {
  logsLoading.value = true
  logsError.value = null
  latestRun.value = null

  // Find the job name for display
  const job = jobs.value.find(j => j.id === jobId)
  logsJobName.value = job?.name || jobId

  try {
    const runsData = await getJobRuns({ jobID: jobId, limit: 1 })
    const runs = runsData.data || []

    if (runs.length === 0) {
      latestRun.value = null
    } else {
      latestRun.value = runs[0]
    }
  } catch (e) {
    logsError.value = e.message
  } finally {
    logsLoading.value = false
    showLogsModal.value = true
  }
}

async function loadAgentsAndGroups() {
  try {
    const [agentsData, groupsData] = await Promise.all([
      getAgents(),
      getGroups(),
    ])
    agents.value = agentsData.data || []
    groups.value = groupsData.data || []
  } catch (e) {
    console.error('Failed to load agents and groups:', e)
  }
}

watch(selectedSite, () => {
  page.value = 1
  load()
})

watch(() => props.lastEvent, (ev) => {
  if (!ev || selectedSite.value) return
  if (ev.type === 'job.updated' || ev.type === 'job.result') {
    load()
  }
})

watch(() => form.value.agent_id, () => {
  filterCredentials()
})

onMounted(() => {
  load()
  loadAgentsAndGroups()
  loadCredentials()
})
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

.stale-banner {
  background: rgba(255, 167, 38, 0.12);
  border: 1px solid rgba(255, 167, 38, 0.4);
  color: #ffa726;
  padding: 0.5rem 0.85rem;
  border-radius: 4px;
  font-size: 0.85rem;
  margin-bottom: 1rem;
}

.form-card {
  background: #1e1e2e;
  border: 1px solid #333;
  border-radius: 8px;
  padding: 1.5rem;
  margin-bottom: 1.5rem;
}
.form-card h3 { margin: 0 0 1rem; }

.dispatch-mode {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
}

.dispatch-mode label { color: #aaa; font-size: 0.9rem; }

.mode-buttons {
  display: flex;
  gap: 0.5rem;
}

.mode-btn {
  flex: 1;
  padding: 0.6rem 1rem;
  background: #333;
  color: #aaa;
  border: 1px solid #444;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9rem;
  transition: all 0.2s;
}

.mode-btn:hover {
  background: #404040;
  border-color: #555;
}

.mode-btn.active {
  background: #4f8ef7;
  color: #fff;
  border-color: #4f8ef7;
}

.form-grid {
  display: grid;
  grid-template-columns: 160px 1fr;
  gap: 0.6rem 1rem;
  align-items: center;
  margin-bottom: 1rem;
}
.form-grid label { color: #aaa; font-size: 0.9rem; }
.form-grid input,
.form-grid select {
  padding: 0.4rem 0.6rem;
  border-radius: 4px;
  border: 1px solid #444;
  background: #111;
  color: #fff;
  font-size: 0.9rem;
}

.form-grid select {
  cursor: pointer;
}

.form-grid input:focus,
.form-grid select:focus {
  outline: none;
  border-color: #4f8ef7;
  box-shadow: 0 0 0 2px rgba(79, 142, 247, 0.1);
}
.optional { color: #666; font-size: 0.8rem; }

.sync-flags-row {
  margin-bottom: 1rem;
}

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
.empty { color: #888; margin: 2rem 0; }

.filters { margin-bottom: 1.5rem; }

.search-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  margin-bottom: 1rem;
  border-radius: 4px;
  border: 1px solid #2a2a3e;
  background: #0f0f1a;
  color: #fff;
  font-size: 0.95rem;
}

.search-input:focus {
  outline: none;
  border-color: #4f8ef7;
  box-shadow: 0 0 0 2px rgba(79, 142, 247, 0.1);
}

.filter-chips { display: flex; gap: 0.5rem; flex-wrap: wrap; }

.chip {
  padding: 0.4rem 1rem;
  border-radius: 999px;
  border: 1px solid #2a2a3e;
  background: transparent;
  color: #aaa;
  cursor: pointer;
  font-size: 0.85rem;
  transition: all 0.2s;
}

.chip:hover { border-color: #4f8ef7; color: #4f8ef7; }
.chip.active { background: #4f8ef7; border-color: #4f8ef7; color: #fff; }

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
.badge.pending   { background: #2a2a1a; color: #f0b429; }
.badge.running   { background: #1a2a3a; color: #4f8ef7; }
.badge.completed { background: #1a3a1a; color: #4caf50; }
.badge.failed    { background: #3a1a1a; color: #e55; }

button.danger-sm {
  padding: 0.2rem 0.6rem;
  background: #3a1a1a;
  color: #e55;
  border: 1px solid #e55;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.8rem;
}

.action-btn {
  padding: 0.2rem 0.6rem;
  background: #1a3a4a;
  color: #4f8ef7;
  border: 1px solid #4f8ef7;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.8rem;
  transition: all 0.15s ease;
}

.action-btn:hover {
  background: #2a4a5a;
  color: #fff;
}

/* Logs Modal */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: #1e1e2e;
  border: 1px solid #333;
  border-radius: 8px;
  width: 90%;
  max-width: 900px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.5);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid #333;
}

.modal-header h3 {
  margin: 0;
  font-size: 1.1rem;
}

.close-btn {
  background: none;
  border: none;
  color: #888;
  font-size: 1.5rem;
  cursor: pointer;
  padding: 0;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.close-btn:hover {
  color: #fff;
}

.modal-body {
  flex: 1;
  overflow: auto;
  padding: 1.5rem;
}

.logs-container {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.run-info {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  padding: 1rem;
  background: #252535;
  border-radius: 4px;
  font-size: 0.9rem;
}

.run-info div {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.run-info strong {
  color: #aaa;
}

.exit-success {
  color: #4caf50;
  font-weight: 600;
}

.exit-fail {
  color: #e55;
  font-weight: 600;
}

.logs-output {
  background: #0f0f1a;
  border: 1px solid #333;
  border-radius: 4px;
  padding: 1rem;
  font-size: 0.85rem;
  line-height: 1.5;
  max-height: 400px;
  overflow: auto;
  color: #ccc;
  white-space: pre-wrap;
  word-break: break-word;
}

.modal-footer {
  padding: 1rem 1.5rem;
  border-top: 1px solid #333;
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}

.modal-footer button.secondary {
  padding: 0.4rem 1rem;
  background: #333;
  color: #aaa;
  border: 1px solid #444;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9rem;
}

.modal-footer button.secondary:hover {
  background: #404040;
  color: #fff;
}
</style>
