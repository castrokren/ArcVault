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
import { getAgents, getFederationAgents } from '../api.js'
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
  if (!updateStore.available || !updateStore.latest) return false
  return agent.version !== updateStore.latest
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
/* Global design system covers: page-header, stale-banner, error, empty,
   filters, search-input, filter-chips, chip, table, mono, badge */

.update-badge {
  display: inline-block;
  margin-left: 0.4rem;
  padding: 0.15rem 0.45rem;
  background: var(--bg-warning);
  color: var(--color-warning);
  border-radius: 4px;
  font-family: var(--font-body);
  font-size: 0.72rem;
  font-weight: 600;
}

.actions-cell {
  display: flex;
  gap: 0.4rem;
  align-items: center;
}

.btn-update-agent {
  padding: 0.25rem 0.75rem;
  background: var(--color-warning);
  color: var(--bg-base);
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-family: var(--font-body);
  font-size: 0.78rem;
  font-weight: 600;
  transition: filter 0.15s;
}
.btn-update-agent:hover { filter: brightness(1.1); }

.btn-rollback-agent {
  padding: 0.25rem 0.75rem;
  background: transparent;
  color: var(--color-warning);
  border: 1px solid rgba(245, 158, 11, 0.4);
  border-radius: 4px;
  cursor: pointer;
  font-family: var(--font-body);
  font-size: 0.78rem;
  font-weight: 600;
  transition: background 0.15s;
}
.btn-rollback-agent:hover { background: var(--bg-warning); }
</style>
