<template>
  <div>
    <div class="page-header">
      <h1>Agents</h1>
      <button @click="load" :disabled="loading">{{ loading ? 'Loading...' : 'Refresh' }}</button>
    </div>

    <!-- Stale data banner: federation site proxy data -->
    <div v-if="stale" class="stale-banner">
      ⚠ Site data last synced {{ fmtStaleTime(staleAsOf) }}. Data may be outdated.
    </div>
    <!-- Stale data banner: local coordinator sync lag -->
    <div v-if="!selectedSite && syncStale" class="stale-banner">
      ⚠ Federation sync in progress. Some agent data may be outdated.
    </div>

    <div v-if="error" class="error">{{ error }}</div>

    <div class="filters" v-if="!selectedSite">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="Search agents by ID or hostname..."
        class="search-input"
        @input="onFilterChange"
      />
      <div class="filter-chips">
        <button
          v-for="s in ['all', 'online', 'offline']"
          :key="s"
          class="chip"
          :class="{ active: statusFilter === s }"
          @click="setStatus(s)"
        >
          {{ s.charAt(0).toUpperCase() + s.slice(1) }}
        </button>
      </div>
    </div>

    <div v-if="agents.length === 0 && !loading" class="empty">
      {{ selectedSite ? 'No agents found for this site.' : (searchQuery || statusFilter !== 'all' ? 'No agents match your search.' : 'No agents registered.') }}
    </div>

    <table v-else class="table">
      <thead>
        <tr>
          <th>ID</th>
          <th>Hostname</th>
          <th>OS</th>
          <th>Version</th>
          <th>Status</th>
          <th>Last Seen</th>
          <th v-if="!selectedSite"></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="agent in agents" :key="agent.id">
          <td class="mono">{{ agent.id }}</td>
          <td>{{ agent.hostname }}</td>
          <td>{{ agent.os }}</td>
          <td>
            {{ agent.version }}
            <span v-if="!selectedSite && updateAvailable(agent)" class="update-badge">Update</span>
          </td>
          <td>
            <span class="badge" :class="agent.status">{{ agent.status }}</span>
          </td>
          <td>{{ formatDate(agent.last_seen) }}</td>
          <td v-if="!selectedSite" class="actions-cell">
            <button
              v-if="updateAvailable(agent) && agent.status === 'online'"
              class="btn-update-agent"
              @click="openUpdateModal(agent)"
            >
              Update
            </button>
            <button
              v-if="agent.rollback_available"
              class="btn-rollback-agent"
              @click="openRollbackModal(agent)"
            >
              ↩ Rollback
            </button>
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

    <AgentUpdateModal
      v-if="!selectedSite"
      :isOpen="modalOpen"
      :agentId="selectedAgent?.id"
      :agentVersion="selectedAgent?.version"
      :agents="agents"
      :lastEvent="lastEvent"
      @close="modalOpen = false"
    />

    <RollbackModal
      v-if="!selectedSite && rollbackModal.show"
      target="agent"
      :agentId="rollbackModal.agentId"
      @close="rollbackModal.show = false"
      @complete="onRollbackComplete"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, inject } from 'vue'
import { getAgents, getFederationAgents } from '../api'
import AgentUpdateModal from '../components/AgentUpdateModal.vue'
import RollbackModal from '../components/RollbackModal.vue'
import Pagination from '../components/Pagination.vue'
import { useFederationLag } from '../composables/useFederationLag.js'

const props = defineProps(['lastEvent'])

const selectedSite = inject('selectedSite', ref(null))

const result = ref({ data: [], total: 0, page: 1, pages: 0, limit: 25 })
const fedAgents = ref([])
const stale = ref(false)
const staleAsOf = ref(null)

// Sync lag: shows banner when local coordinator is behind peers (no site selected)
const { isStale: syncStale } = useFederationLag()

const agents = computed(() => selectedSite.value ? fedAgents.value : (result.value.data || []))

const page = ref(1)
const limit = 25
const loading = ref(false)
const error = ref(null)
const searchQuery = ref('')
const statusFilter = ref('all')
const modalOpen = ref(false)
const selectedAgent = ref(null)
const rollbackModal = ref({ show: false, agentId: null })

const updateStore = inject('updateStore', { available: false, latest: '' })

function updateAvailable(agent) {
  // An agent needs updating only when it's behind the coordinator's current running version.
  // Avoids false positives caused by reusing the coordinator's update-available state.
  if (!updateStore.current) return false
  const normalize = v => String(v).replace(/^v/, '')
  return normalize(agent.version) !== normalize(updateStore.current)
}

function openUpdateModal(agent) {
  selectedAgent.value = agent
  modalOpen.value = true
}

function openRollbackModal(agent) {
  rollbackModal.value = { show: true, agentId: agent.id }
}

function onRollbackComplete() {
  rollbackModal.value.show = false
  load()
}

async function load() {
  loading.value = true
  error.value = null
  stale.value = false

  try {
    if (selectedSite.value) {
      const data = await getFederationAgents(selectedSite.value)
      fedAgents.value = data.agents || []
      stale.value = data.stale || false
      staleAsOf.value = data.as_of || null
    } else {
      result.value = await getAgents({
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

function goToPage(n) {
  page.value = n
  load()
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleString()
}

function fmtStaleTime(iso) {
  if (!iso) return 'an unknown time ago'
  const secs = Math.floor((Date.now() - new Date(iso)) / 1000)
  if (secs < 60) return `${secs}s ago`
  if (secs < 3600) return `${Math.floor(secs / 60)}m ago`
  return `${Math.floor(secs / 3600)}h ago`
}

watch(selectedSite, () => {
  page.value = 1
  load()
})

watch(() => props.lastEvent, (ev) => {
  if (!selectedSite.value && (ev?.type === 'agent.heartbeat' || ev?.type === 'agent.updated')) load()
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

.stale-banner {
  background: rgba(255, 167, 38, 0.12);
  border: 1px solid rgba(255, 167, 38, 0.4);
  color: #ffa726;
  padding: 0.5rem 0.85rem;
  border-radius: 4px;
  font-size: 0.85rem;
  margin-bottom: 1rem;
}

.error { color: #e55; margin-bottom: 1rem; }
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

.filter-chips { display: flex; gap: 0.5rem; }

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

.update-badge {
  display: inline-block;
  margin-left: 0.4rem;
  padding: 0.1rem 0.4rem;
  background: #3a2a10;
  color: #f39c12;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 600;
}

.actions-cell { display: flex; gap: 0.4rem; align-items: center; }

.btn-update-agent {
  padding: 0.3rem 0.8rem;
  background: #f39c12;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.8rem;
  font-weight: 600;
}
.btn-update-agent:hover { background: #e08e00; }

.btn-rollback-agent {
  padding: 0.3rem 0.8rem;
  background: transparent;
  color: #e6a817;
  border: 1px solid #e6a817;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.8rem;
  font-weight: 600;
  transition: background 0.15s;
}
.btn-rollback-agent:hover { background: rgba(230, 168, 23, 0.15); }
</style>