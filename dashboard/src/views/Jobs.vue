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
            <button
              v-if="(job.status === 'pending' || job.status === 'running') && auth.hasRole('operator')"
              class="cancel-btn"
              :disabled="cancellingJobId === job.id"
              @click="cancelJob(job)"
            >
              {{ cancellingJobId === job.id ? 'Canceling...' : 'Cancel' }}
            </button>
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
          <button class="modal-close" @click="showLogsModal = false">×</button>
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
import { getJobs, createJob as apiCreateJob, deleteJob, cancelJob as apiCancelJob, getFederationJobs, getAgents, getGroups, getJobRuns, getToken } from '../api'
import { useAuth } from '../composables/useAuth'
import { formatDate, fmtStaleTime } from '../utils/format.js'
import Pagination from '../components/Pagination.vue'
import ScheduleBuilder from '../components/ScheduleBuilder.vue'
import SyncFlagsBuilder from '../components/SyncFlagsBuilder.vue'
import { useFederationLag } from '../composables/useFederationLag.js'

const props = defineProps(['lastEvent'])

const selectedSite = inject('selectedSite', ref(null))

const { hasRole } = useAuth()
const auth = useAuth()

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

const cancellingJobId = ref(null)

async function cancelJob(job) {
  if (!confirm(`Are you sure you want to cancel job "${job.name}"?`)) return
  cancellingJobId.value = job.id
  try {
    const response = await apiCancelJob(job.id)
    if (!response.ok) {
      const text = await response.text()
      alert(`Failed to cancel job: ${text}`)
      return
    }
    await load()
  } catch (err) {
    alert(`Failed to cancel job: ${err}`)
  } finally {
    cancellingJobId.value = null
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
/* Jobs-specific styles using global design tokens.
   Global token-driven classes from style.css handle: .page-header, .form-card,
   .form-grid, .dispatch-mode, .mode-btn, .table, .badge, .chip, .search-input,
   .filters, .error, .empty, .modal-overlay, .modal, .modal-header, .modal-body,
   .modal-footer, .modal-close, .mono, .danger-sm. */

.sync-flags-row {
  margin-bottom: 1rem;
}

/* Table action buttons — Jobs-specific */
.action-btn {
  padding: 0.22rem 0.65rem;
  background: var(--accent-2-dim);
  border: 1px solid var(--accent-2-border);
  border-radius: var(--radius-ctrl);
  color: var(--accent-2);
  cursor: pointer;
  font-family: var(--font-body);
  font-size: 0.78rem;
  font-weight: 600;
  transition: filter 0.15s, transform 0.15s;
}
.action-btn:hover {
  filter: brightness(1.15);
  transform: translateY(-1px);
}

.cancel-btn {
  padding: 0.22rem 0.65rem;
  background: var(--bg-warning);
  border: 1px solid rgba(251, 191, 36, 0.35);
  border-radius: var(--radius-ctrl);
  color: var(--color-warning);
  cursor: pointer;
  font-family: var(--font-body);
  font-size: 0.78rem;
  font-weight: 600;
  transition: filter 0.15s;
}
.cancel-btn:hover:not(:disabled) { filter: brightness(1.1); }
.cancel-btn:disabled { opacity: 0.45; cursor: not-allowed; }

/* ponytail: cancelled/canceling badges not in global — keep here */
:deep(.badge.canceling) { background: var(--bg-warning);  color: var(--color-warning); border-color: rgba(251,191,36,0.25); }
:deep(.badge.cancelled) { background: var(--bg-elevated); color: var(--text-muted);    border-color: var(--border-default); }

/* Logs modal — wider than global (900px vs 520px) */
.modal {
  width: min(900px, 94vw);
  max-height: 82vh;
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-muted);
  font-size: 1.1rem;
  cursor: pointer;
  padding: 0.2rem 0.4rem;
  border-radius: 6px;
  line-height: 1;
  transition: color 0.15s, background 0.15s;
}
.close-btn:hover { color: var(--text-primary); background: var(--bg-card); }

.logs-container {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.run-info {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  padding: 0.85rem 1rem;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-ctrl);
  font-size: 0.85rem;
}
.run-info div { display: flex; flex-direction: column; gap: 0.25rem; }
.run-info strong { color: var(--text-muted); font-size: 0.72rem; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase; }

.exit-success { color: var(--color-success); font-weight: 600; }
.exit-fail    { color: var(--color-error);   font-weight: 600; }

.logs-output {
  background: var(--bg-base);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-ctrl);
  padding: 0.85rem 1rem;
  font-family: var(--font-mono);
  font-size: 0.8rem;
  line-height: 1.6;
  max-height: 420px;
  overflow: auto;
  color: var(--text-secondary);
  white-space: pre-wrap;
  word-break: break-word;
}

.modal-footer button.secondary {
  display: inline-flex;
  align-items: center;
  padding: 0.4rem 1rem;
  background: var(--bg-elevated);
  color: var(--text-primary);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-ctrl);
  cursor: pointer;
  font-family: var(--font-body);
  font-size: 0.85rem;
  font-weight: 500;
  transition: background 0.15s, border-color 0.15s;
}
.modal-footer button.secondary:hover {
  background: var(--bg-card);
  border-color: var(--border-strong);
}
</style>
