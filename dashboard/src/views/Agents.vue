<template>
  <div class="fleet-view">
    <!-- ── Command bar ─────────────────────────────────────── -->
    <div class="cmdbar">
      <h1>{{ selectedSite ? 'Site Agents' : 'Fleet' }}</h1>
      <span class="cmd-sub">{{ selectedSite ? selectedSite : 'Local coordinator' }}</span>
      <div class="cmd-spacer"></div>
      <label v-if="!selectedSite" class="cmd-search">
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><circle cx="6" cy="6" r="4.2" stroke="currentColor" stroke-width="1.3"/><path d="M9.2 9.2 12 12" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/></svg>
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search by hostname or ID…"
          @input="onFilterChange"
        />
      </label>
      <button class="btn btn-secondary" @click="load" :disabled="loading">
        {{ loading ? 'Loading…' : 'Refresh' }}
      </button>
    </div>

    <!-- Stale banners -->
    <div v-if="stale" class="stale-banner">
      ⚠ Site data last synced {{ fmtStaleTime(staleAsOf) }}. Data may be outdated.
    </div>
    <div v-if="!selectedSite && syncStale" class="stale-banner">
      ⚠ Federation sync in progress. Some agent data may be outdated.
    </div>
    <div v-if="error" class="error">{{ error }}</div>

    <!-- ── Fleet readout band ──────────────────────────────── -->
    <section v-if="!selectedSite" class="fleet">
      <div class="fleet-cell">
        <div class="fleet-k">Fleet status</div>
        <div class="fleet-big">{{ totalAgents }}<small> agents</small></div>
        <div class="meter">
          <span class="seg-on"  :style="{ width: onlineRate + '%' }"></span>
          <span class="seg-off" :style="{ width: (100 - onlineRate) + '%' }"></span>
        </div>
        <div class="meter-legend">
          <span><i class="dot d-on"></i><b>{{ onlineCount }}</b> online</span>
          <span><i class="dot d-off"></i><b>{{ offlineCount }}</b> offline</span>
        </div>
      </div>

      <div class="fleet-cell">
        <div class="fleet-k">Online rate · recent</div>
        <svg class="chart-svg" viewBox="0 0 300 56" preserveAspectRatio="none">
          <path class="area" :d="onlineSpark.area" />
          <polyline class="line" :points="onlineSpark.line" />
          <circle class="end" :cx="onlineSpark.end[0]" :cy="onlineSpark.end[1]" r="2.6" />
        </svg>
        <div class="meter-legend"><span><b>{{ onlineRate }}%</b> online now</span></div>
      </div>

      <div class="fleet-cell">
        <div class="fleet-k">Version drift</div>
        <div class="fleet-big">{{ updatesPending }}<small> behind</small></div>
        <div v-if="updatesPending" class="drift-note">
          <svg width="11" height="11" viewBox="0 0 12 12" fill="none"><path d="M6 1v6M6 9.5v.5" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/></svg>
          {{ updatesPending }} agent{{ updatesPending === 1 ? '' : 's' }} can update to {{ updateStore.current }}
        </div>
        <div v-else class="drift-note is-ok">
          <svg width="11" height="11" viewBox="0 0 12 12" fill="none"><path d="M2.5 6.5 5 9l4.5-5" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/></svg>
          All agents on {{ updateStore.current || 'current' }}
        </div>
      </div>
    </section>

    <!-- ── Workbench body ──────────────────────────────────── -->
    <div class="fleet-body" :class="{ 'no-rails': selectedSite }">
      <!-- left rail: status segments -->
      <aside v-if="!selectedSite" class="rail">
        <div class="rail-h">Status</div>
        <button
          v-for="s in statusSegs"
          :key="s.key"
          class="seg"
          :class="{ active: statusFilter === s.key }"
          @click="setStatus(s.key)"
        >
          {{ s.label }}<span class="seg-count">{{ s.count }}</span>
        </button>
      </aside>

      <!-- main surface: agents table -->
      <main class="surface">
        <div v-if="agents.length === 0 && !loading" class="empty">
          {{ selectedSite ? 'No agents found for this site.' : (searchQuery || statusFilter !== 'all' ? 'No agents match your search.' : 'No agents registered.') }}
        </div>
        <table v-else class="table">
          <thead>
            <tr>
              <th>Agent</th>
              <th>OS</th>
              <th>Version</th>
              <th>Last Seen</th>
              <th v-if="!selectedSite"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="agent in agents" :key="agent.id">
              <td>
                <div class="agent-cell">
                  <span class="stat-dot" :class="agent.status"></span>
                  <div>
                    <div class="agent-host">{{ agent.hostname }}</div>
                    <div class="agent-id">{{ agent.id }}</div>
                  </div>
                </div>
              </td>
              <td class="os-cell">{{ agent.os }}</td>
              <td>
                <span class="ver mono">{{ agent.version }}</span>
                <span v-if="!selectedSite && updateAvailable(agent)" class="ver-badge">update</span>
              </td>
              <td class="last-seen mono">{{ formatDate(agent.last_seen) }}</td>
              <td v-if="!selectedSite" class="row-action">
                <button
                  v-if="updateAvailable(agent) && agent.status === 'online'"
                  class="btn btn-sm btn-update"
                  @click="openUpdateModal(agent)"
                >
                  Update
                </button>
                <button
                  class="btn btn-sm btn-secondary"
                  @click="openTokenModal(agent)"
                  title="Generate a token for installing this agent on a new machine"
                >
                  Get Token
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
      </main>

      <!-- activity rail: live WS feed -->
      <aside v-if="!selectedSite" class="activity">
        <div class="rail-h activity-h">
          Live activity
          <span class="live-tag" :class="{ off: !wsLive }">● {{ wsLive ? 'live' : 'idle' }}</span>
        </div>
        <div v-if="feed.length === 0" class="feed-empty">Waiting for agent events…</div>
        <div v-else class="feed">
          <div v-for="ev in feed" :key="ev.id" class="ev">
            <span class="ev-dot" :class="'ev-' + ev.kind"></span>
            <div class="ev-body">
              <div class="ev-text"><b>{{ ev.who }}</b> {{ ev.label }}</div>
              <div class="ev-time">{{ ev.time }}</div>
            </div>
          </div>
        </div>
      </aside>
    </div>

    <AgentUpdateModal
      v-if="!selectedSite"
      :isOpen="modalOpen"
      :agentId="selectedAgent?.id"
      :agentVersion="selectedAgent?.version"
      :agents="agents"
      :lastEvent="lastEvent"
      @close="modalOpen = false"
    />

    <AgentTokenModal
      :isOpen="tokenModalOpen"
      :agentId="selectedAgentForToken?.id"
      @close="tokenModalOpen = false"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, inject } from 'vue'
import { getAgents, getFederationAgents } from '../api'
import { formatDate, fmtStaleTime } from '../utils/format.js'
import AgentUpdateModal from '../components/AgentUpdateModal.vue'
import AgentTokenModal from '../components/AgentTokenModal.vue'
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
const tokenModalOpen = ref(false)
const selectedAgentForToken = ref(null)

const updateStore = inject('updateStore', { available: false, latest: '', current: '' })

function updateAvailable(agent) {
  // An agent needs updating only when it's behind the coordinator's current running version.
  if (!updateStore.current) return false
  const normalize = v => String(v).replace(/^v/, '')
  return normalize(agent.version) !== normalize(updateStore.current)
}

function openUpdateModal(agent) {
  selectedAgent.value = agent
  modalOpen.value = true
}

function openTokenModal(agent) {
  selectedAgentForToken.value = agent
  tokenModalOpen.value = true
}

/* ── Fleet-band derived values (read straight off loaded data) ── */
const totalAgents = computed(() => result.value.total || agents.value.length)
const onlineCount = computed(() => agents.value.filter(a => a.status === 'online').length)
const offlineCount = computed(() => agents.value.length - onlineCount.value)
const onlineRate = computed(() => {
  const n = agents.value.length
  return n ? Math.round((onlineCount.value / n) * 100) : 0
})
const updatesPending = computed(() =>
  agents.value.filter(a => !selectedSite.value && updateAvailable(a)).length
)

const statusSegs = computed(() => [
  { key: 'all',     label: 'All',     count: totalAgents.value },
  { key: 'online',  label: 'Online',  count: onlineCount.value },
  { key: 'offline', label: 'Offline', count: offlineCount.value },
])

// ponytail: representative online-rate shape until a metrics endpoint exists.
// TODO: wire to a real time-series endpoint when available.
const onlineTrend = ref([38, 40, 39, 42, 41, 43, 42, 43, 41, 44, 43, 45])

// Build line + filled-area path strings for an SVG sparkline.
function buildSpark(data, w = 300, h = 56, pad = 6) {
  const max = Math.max(...data), min = Math.min(...data)
  const span = max - min || 1
  const step = w / (data.length - 1)
  const pts = data.map((v, i) => [
    Math.round(i * step),
    Math.round(pad + (h - pad * 2) * (1 - (v - min) / span)),
  ])
  const line = pts.map(p => p.join(',')).join(' ')
  const area = `M${pts[0][0]},${pts[0][1]} ` +
    pts.slice(1).map(p => `L${p[0]},${p[1]}`).join(' ') +
    ` L${w},${h} L0,${h} Z`
  return { line, area, end: pts[pts.length - 1] }
}
const onlineSpark = computed(() => buildSpark(onlineTrend.value))

/* ── Live activity feed (real WS events) ────────────────────── */
const wsLive = ref(false)
const feed = ref([])
let feedSeq = 0

const EVENT_LABELS = {
  'agent.heartbeat': 'sent a heartbeat',
  'agent.updated':   'updated',
  'agent.registered':'registered',
  'agent.deleted':   'was removed',
  'job.updated':     'job updated',
  'job.result':      'job finished',
}
const EVENT_KIND = {
  'agent.heartbeat': 'heartbeat',
  'agent.updated':   'update',
  'agent.registered':'connect',
  'agent.deleted':   'offline',
  'job.updated':     'update',
  'job.result':      'update',
}

function pushEvent(ev) {
  if (!ev || !ev.type) return
  wsLive.value = true
  const p = ev.payload || ev
  const who = ev.hostname || ev.agent_id || ev.id || p.hostname || p.agent_id || p.id || 'coordinator'
  feed.value.unshift({
    id: ++feedSeq,
    who,
    label: EVENT_LABELS[ev.type] || ev.type,
    kind: EVENT_KIND[ev.type] || 'update',
    time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }),
  })
  if (feed.value.length > 8) feed.value.length = 8
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

watch(selectedSite, () => {
  page.value = 1
  load()
})

watch(() => props.lastEvent, (ev) => {
  pushEvent(ev)
  if (!selectedSite.value && (ev?.type === 'agent.heartbeat' || ev?.type === 'agent.updated')) load()
})

onMounted(load)
</script>

<style scoped>
/* Reuses global tokenized classes: .table .badge .btn .stale-banner .error .empty
   (style.css). Only the Workbench structure is view-scoped below — all colors and
   fonts by token reference, per design.md (Kiln). */

.fleet-view { max-width: 1360px; margin: 0 auto; }

/* ── Command bar ─────────────────────────────────────────── */
.cmdbar { display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1.25rem; flex-wrap: wrap; }
.cmdbar h1 {
  font-family: var(--font-display); font-weight: 700; font-size: 1.4rem;
  letter-spacing: -0.02em; color: var(--text-primary); margin: 0;
}
.cmd-sub {
  font-family: var(--font-mono); font-size: 0.68rem; letter-spacing: 0.08em;
  text-transform: uppercase; color: var(--text-muted);
}
.cmd-spacer { flex: 1; }
.cmd-search {
  display: flex; align-items: center; gap: 0.45rem; padding: 0.4rem 0.7rem;
  background: var(--bg-input); border: 1px solid var(--border-default);
  border-radius: var(--radius-ctrl); color: var(--text-muted); min-width: 240px;
  transition: border-color 0.15s;
}
.cmd-search:focus-within { border-color: var(--accent); }
.cmd-search input {
  background: none; border: none; outline: none; color: var(--text-primary);
  font-family: var(--font-body); font-size: 0.85rem; width: 100%;
}
.cmd-search input::placeholder { color: var(--text-muted); }

/* ── Fleet readout band ──────────────────────────────────── */
.fleet {
  display: grid; grid-template-columns: minmax(0,1.15fr) minmax(0,1.5fr) minmax(0,1fr);
  background: var(--bg-card); border: 1px solid var(--border-default);
  border-radius: var(--radius-card); overflow: hidden; margin-bottom: 1.25rem;
}
.fleet-cell { padding: 1rem 1.25rem; border-right: 1px solid var(--border-subtle); min-width: 0; }
.fleet-cell:last-child { border-right: none; }
.fleet-k {
  font-family: var(--font-mono); font-size: 0.65rem; letter-spacing: 0.1em;
  text-transform: uppercase; color: var(--text-muted); margin-bottom: 0.55rem;
}
.fleet-big {
  font-family: var(--font-display); font-weight: 800; font-size: 2rem; line-height: 1;
  letter-spacing: -0.02em; color: var(--text-primary); font-variant-numeric: tabular-nums;
}
.fleet-big small { font-size: 1rem; color: var(--text-muted); font-weight: 500; }

.meter { display: flex; height: 8px; border-radius: 2px; overflow: hidden; margin-top: 0.85rem; background: var(--bg-elevated); }
.meter span { display: block; height: 100%; }
.seg-on  { background: var(--color-success); }
.seg-off { background: color-mix(in oklab, var(--text-muted) 55%, transparent); }
.meter-legend { display: flex; gap: 1.1rem; margin-top: 0.6rem; font-family: var(--font-mono); font-size: 0.66rem; letter-spacing: 0.04em; color: var(--text-secondary); }
.meter-legend b { color: var(--text-primary); font-weight: 600; }
.dot { display: inline-block; width: 7px; height: 7px; border-radius: 50%; margin-right: 0.35rem; vertical-align: middle; }
.d-on { background: var(--color-success); }
.d-off { background: color-mix(in oklab, var(--text-muted) 60%, transparent); }

.chart-svg { width: 100%; height: 56px; margin-top: 0.4rem; display: block; }
.chart-svg .area { fill: var(--accent-dim); }
.chart-svg .line { fill: none; stroke: var(--accent); stroke-width: 1.6; }
.chart-svg .end { fill: var(--accent); }

.drift-note {
  display: inline-flex; align-items: center; gap: 0.4rem; margin-top: 0.7rem;
  font-family: var(--font-mono); font-size: 0.68rem; color: var(--accent-2);
  background: var(--accent-2-dim); border: 1px solid var(--accent-2-border);
  border-radius: 3px; padding: 0.15rem 0.5rem;
}
.drift-note.is-ok { color: var(--color-success); background: var(--bg-success); border-color: color-mix(in oklab, var(--color-success) 30%, transparent); }

/* ── Workbench body ──────────────────────────────────────── */
.fleet-body { display: grid; grid-template-columns: 190px minmax(0,1fr) 258px; gap: 1.25rem; align-items: start; }
.fleet-body.no-rails { grid-template-columns: 1fr; }

.rail-h {
  font-family: var(--font-mono); font-size: 0.63rem; letter-spacing: 0.12em;
  text-transform: uppercase; color: var(--text-muted); margin: 0 0 0.6rem; padding-left: 0.15rem;
}
.seg {
  display: flex; align-items: center; justify-content: space-between; gap: 0.5rem;
  width: 100%; text-align: left; padding: 0.45rem 0.6rem; margin-bottom: 0.15rem;
  background: none; border: 1px solid transparent; border-radius: var(--radius-ctrl);
  color: var(--text-secondary); font-family: var(--font-body); font-size: 0.85rem; cursor: pointer;
  transition: background 0.12s, color 0.12s, border-color 0.12s;
}
.seg:hover { background: var(--bg-card); color: var(--text-primary); }
.seg.active { background: var(--accent-dim); border-color: var(--accent-border); color: var(--text-primary); font-weight: 600; }
.seg-count { font-family: var(--font-mono); font-size: 0.72rem; color: var(--text-muted); }
.seg.active .seg-count { color: var(--accent); }

/* table surface */
.surface { background: var(--bg-card); border: 1px solid var(--border-default); border-radius: var(--radius-card); overflow: hidden; }
.surface .table th { top: 0; background: var(--bg-surface); }
.surface .empty { margin: 2.5rem 0; }
.agent-cell { display: flex; align-items: center; gap: 0.65rem; }
.stat-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; background: color-mix(in oklab, var(--text-muted) 60%, transparent); }
.stat-dot.online { background: var(--color-success); box-shadow: 0 0 0 3px var(--bg-success); }
.agent-host { font-weight: 600; font-size: 0.87rem; color: var(--text-primary); }
.agent-id { font-family: var(--font-mono); font-size: 0.68rem; color: var(--text-muted); }
.os-cell { color: var(--text-secondary); font-size: 0.82rem; }
.ver { font-size: 0.75rem; color: var(--text-secondary); }
.ver-badge {
  font-family: var(--font-mono); font-size: 0.6rem; letter-spacing: 0.06em; text-transform: uppercase;
  color: var(--color-warning); background: var(--bg-warning);
  border: 1px solid color-mix(in oklab, var(--color-warning) 40%, transparent);
  border-radius: 3px; padding: 0.05rem 0.35rem; margin-left: 0.45rem;
}
.last-seen { font-size: 0.75rem; color: var(--text-secondary); }
.row-action { text-align: right; }
.btn-sm { padding: 0.28rem 0.6rem; font-size: 0.72rem; }
.btn-update { background: var(--accent-dim); border-color: var(--accent-border); color: var(--accent); }
.btn-update:hover:not(:disabled) { background: var(--accent); color: var(--text-inverse); border-color: var(--accent); }

/* activity rail */
.activity { background: var(--bg-card); border: 1px solid var(--border-default); border-radius: var(--radius-card); padding: 0.9rem 0.95rem; }
.activity-h { display: flex; align-items: center; justify-content: space-between; margin-bottom: 0.7rem; }
.live-tag { color: var(--color-success); font-size: 0.63rem; letter-spacing: 0.06em; }
.live-tag.off { color: var(--text-muted); }
.feed-empty { color: var(--text-muted); font-size: 0.78rem; padding: 0.5rem 0.1rem; }
.feed { display: flex; flex-direction: column; }
.ev { display: grid; grid-template-columns: 10px 1fr; gap: 0.6rem; padding: 0.5rem 0.1rem; border-bottom: 1px solid var(--border-subtle); }
.ev:last-child { border-bottom: none; }
.ev-dot { width: 7px; height: 7px; border-radius: 50%; margin-top: 0.35rem; }
.ev-heartbeat { background: var(--color-success); }
.ev-update { background: var(--accent); }
.ev-connect { background: var(--color-info); }
.ev-offline { background: var(--color-error); }
.ev-body { min-width: 0; }
.ev-text { font-size: 0.78rem; line-height: 1.35; color: var(--text-secondary); }
.ev-text b { font-family: var(--font-mono); font-weight: 600; font-size: 0.74rem; color: var(--text-primary); }
.ev-time { font-family: var(--font-mono); font-size: 0.64rem; color: var(--text-muted); margin-top: 0.1rem; }

@media (max-width: 1080px) {
  .fleet-body { grid-template-columns: 1fr; }
  .fleet { grid-template-columns: 1fr; }
  .fleet-cell { border-right: none; border-bottom: 1px solid var(--border-subtle); }
  .rail { display: none; }
}
@media (max-width: 620px) {
  .cmd-search { min-width: 0; flex: 1 1 100%; order: 3; }
}
</style>
