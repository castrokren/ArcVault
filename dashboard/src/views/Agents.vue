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

    <!-- ── Stat row (new) ─────────────────────────────────── -->
    <div v-if="!selectedSite" class="stat-grid">
      <!-- Total agents · bar chart -->
      <div class="stat-card">
        <div class="stat-label">Total Agents</div>
        <div class="stat-value">{{ totalAgents }}</div>
        <div class="bar-chart">
          <span v-for="(h, i) in growthTrend" :key="i" :style="{ height: barHeight(h, growthTrend) }"></span>
        </div>
      </div>

      <!-- Online now · area sparkline -->
      <div class="stat-card">
        <div class="stat-label">Online Now</div>
        <div class="stat-value">{{ onlineCount }}</div>
        <svg class="spark-area" viewBox="0 0 130 50" preserveAspectRatio="none">
          <path class="area" :d="onlineSpark.area" />
          <polyline class="line" :points="onlineSpark.line" />
        </svg>
      </div>

      <!-- Online rate · donut -->
      <div class="stat-card is-donut">
        <div class="stat-body">
          <div class="stat-label">Online Rate</div>
          <div class="stat-value">{{ onlineRate }}<span class="pct">%</span></div>
          <div class="stat-sub">{{ updatesPending }} need updates</div>
        </div>
        <div class="donut" :style="{ '--pct': onlineRate }">
          <div class="donut-hole">{{ onlineRate }}</div>
        </div>
      </div>
    </div>

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


  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, inject } from 'vue'
import { getAgents, getFederationAgents } from '../api'
import { formatDate, fmtStaleTime } from '../utils/format.js'
import AgentUpdateModal from '../components/AgentUpdateModal.vue'
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

/* ── Stat-row derived values ──────────────────────────────
   These read straight off the data you already load. */
const totalAgents = computed(() => result.value.total || agents.value.length)
const onlineCount = computed(() => agents.value.filter(a => a.status === 'online').length)
const onlineRate = computed(() => {
  const n = agents.value.length
  return n ? Math.round((onlineCount.value / n) * 100) : 0
})
const updatesPending = computed(() =>
  agents.value.filter(a => !selectedSite.value && updateAvailable(a)).length
)

// TODO: wire these trends to a real metrics endpoint when available.
// Until then they render a representative shape so the cards aren't empty.
const growthTrend = ref([18, 26, 22, 34, 30, 40, 36, 44])
const onlineTrend = ref([38, 40, 39, 42, 41, 43, 42, 43])

function barHeight(v, arr) {
  const max = Math.max(...arr, 1)
  return Math.round((v / max) * 100) + '%'
}

// Build the line + filled-area path strings for an SVG sparkline.
function buildSpark(data, w = 130, h = 50, pad = 6) {
  const max = Math.max(...data), min = Math.min(...data)
  const span = max - min || 1
  const step = w / (data.length - 1)
  const pts = data.map((v, i) => {
    const x = Math.round(i * step)
    const y = Math.round(pad + (h - pad * 2) * (1 - (v - min) / span))
    return [x, y]
  })
  const line = pts.map(p => p.join(',')).join(' ')
  const area = `M${pts[0][0]},${pts[0][1]} ` +
    pts.slice(1).map(p => `L${p[0]},${p[1]}`).join(' ') +
    ` L${w},${h} L0,${h} Z`
  return { line, area }
}
const onlineSpark = computed(() => buildSpark(onlineTrend.value))

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
/* Almost everything here now comes from the global, token-based system in style.css
   (.page-header, .table, .badge, .chip, .search-input, .stat-card, .stat-label,
   .stat-value, .filters, .empty, .error, .stale-banner). Only a couple of
   view-specific bits remain — and they use tokens, so they theme automatically. */

.stat-value .pct {
  font-size: 1rem;
  color: var(--accent);
  margin-left: 1px;
}

.actions-cell {
  display: flex;
  gap: 0.4rem;
  align-items: center;
}

/* .update-badge and .btn-update-agent live in charts.css (shared, tokenized). */
</style>