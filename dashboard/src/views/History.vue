<template>
  <div class="history-view">
    <div class="history-header">
      <h2 class="history-title">History</h2>
    </div>

    <!-- Stale data banner -->
    <div v-if="stale" class="stale-banner history-section">
      ⚠ Site data last synced {{ fmtStaleTime(staleAsOf) }}. Data may be outdated.
    </div>

    <!-- ── Visualization panels (local only) ─────────────── -->
    <template v-if="!selectedSite">
      <JobTimeline
        :job-rows="timelineRows"
        :loading="tlLoading"
        :selected-job="filterJobId"
        window-label="last 48 runs"
        class="history-section"
        @select-job="onSelectJob"
      />

      <AgentRunChart
        :agents="agentChartData"
        :loading="acLoading"
        :selected-agent="filterAgentId"
        :days="14"
        class="history-section"
        @select-agent="onSelectAgent"
      />
    </template>

    <!-- ── Table: paginated run list ────────────────────── -->
    <div class="history-table-card history-section">
      <div class="table-header">
        <span class="section-label">Run Log</span>

        <div class="table-filters" v-if="!selectedSite">
          <input
            v-model="search"
            class="filter-input"
            placeholder="search job / agent…"
            @input="onFilterChange"
          />
          <select v-model="filterStatus" class="filter-select" @change="onFilterChange">
            <option value="">All statuses</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
            <option value="running">Running</option>
          </select>
          <button v-if="filterJobId || filterAgentId || filterStatus || search" class="btn-clear" @click="clearFilters">
            ✕ clear
          </button>
        </div>
      </div>

      <!-- active filter pills -->
      <div v-if="!selectedSite && (filterJobId || filterAgentId)" class="filter-pills">
        <span v-if="filterJobId" class="pill">
          job: {{ filterJobId }}
          <button class="pill-x" @click="onSelectJob(null)">✕</button>
        </span>
        <span v-if="filterAgentId" class="pill">
          agent: {{ filterAgentId }}
          <button class="pill-x" @click="onSelectAgent(null)">✕</button>
        </span>
      </div>

      <div v-if="runsLoading" class="table-loading">
        <span class="tl-loading-dot"></span> loading…
      </div>

      <div v-else-if="displayRuns.length === 0" class="table-empty">
        No runs match the current filters.
      </div>

      <div v-else class="table-wrap">
        <table class="runs-table">
          <thead>
            <tr>
              <th>Status</th>
              <th>Job</th>
              <th>Agent</th>
              <th>Started</th>
              <th>Duration</th>
              <th>Output</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="run in displayRuns"
              :key="run.id"
              class="run-row"
              :class="run.status"
            >
              <td>
                <span class="status-badge" :class="run.status">{{ run.status }}</span>
              </td>
              <td class="run-job">{{ run.job_id }}</td>
              <td class="run-agent">{{ run.agent_id }}</td>
              <td class="run-time">{{ fmtTime(run.started_at) }}</td>
              <td class="run-dur">{{ fmtDuration(run.started_at, run.finished_at) }}</td>
              <td>
                <button
                  v-if="run.output"
                  class="btn-output"
                  @click="openOutput(run)"
                >view</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination (local only) -->
      <Pagination
        v-if="!selectedSite && pagination.pages > 1"
        :page="pagination.page"
        :pages="pagination.pages"
        :total="pagination.total"
        :limit="pagination.limit"
        @change="onPageChange"
        class="table-pagination"
      />
    </div>

    <!-- Output modal -->
    <div v-if="outputModal.visible" class="modal-backdrop" @click.self="outputModal.visible = false">
      <div class="modal-box">
        <div class="modal-header">
          <span class="modal-title">Output — {{ outputModal.jobId }}</span>
          <button class="modal-close" @click="outputModal.visible = false">✕</button>
        </div>
        <pre class="modal-output">{{ outputModal.text }}</pre>
      </div>
    </div>
  </div>
</template>

<script>
import JobTimeline from '../components/JobTimeline.vue'
import AgentRunChart from '../components/AgentRunChart.vue'
import Pagination from '../components/Pagination.vue'
import { getJobRuns, getFederationHistory } from '../api.js'
import { inject, ref } from 'vue'

export default {
  name: 'HistoryView',
  components: { JobTimeline, AgentRunChart, Pagination },

  setup() {
    const selectedSite = inject('selectedSite', ref(null))
    return { selectedSite }
  },

  data() {
    return {
      // ── Timeline data
      tlLoading: true,
      tlRuns: [],
      tlJobMeta: {},

      // ── Agent chart data
      acLoading: true,
      acRuns: [],

      // ── Table data
      runsLoading: false,
      runs: [],
      fedRuns: [],
      pagination: { page: 1, pages: 1, total: 0, limit: 25 },

      // ── Federation stale state
      stale: false,
      staleAsOf: null,

      // ── Filters
      search: '',
      filterStatus: '',
      filterJobId: null,
      filterAgentId: null,

      // ── Output modal
      outputModal: { visible: false, jobId: '', text: '' }
    }
  },

  computed: {
    displayRuns() {
      return this.selectedSite ? this.fedRuns : this.runs
    },

    timelineRows() {
      const byJob = {}
      for (const run of this.tlRuns) {
        if (!byJob[run.job_id]) {
          byJob[run.job_id] = {
            jobId: run.job_id,
            jobName: this.tlJobMeta[run.job_id]?.name ?? run.job_id,
            agentId: run.agent_id,
            runs: []
          }
        }
        byJob[run.job_id].runs.push(run)
      }
      return Object.values(byJob).map(row => {
        const sorted = [...row.runs].sort(
          (a, b) => new Date(a.started_at) - new Date(b.started_at)
        ).slice(-48)
        sorted.forEach((r, i) => (r._idx = i))
        const okCount = sorted.filter(r => r.status === 'completed').length
        const failCount = sorted.filter(r => r.status === 'failed').length
        return { ...row, runs: sorted, okCount, failCount }
      })
    },

    agentChartData() {
      const byAgent = {}
      for (const run of this.acRuns) {
        if (!byAgent[run.agent_id]) {
          byAgent[run.agent_id] = { agentId: run.agent_id, hostname: run.agent_id, runs: [] }
        }
        byAgent[run.agent_id].runs.push(run)
      }
      return Object.values(byAgent)
    }
  },

  watch: {
    selectedSite(val) {
      this.stale = false
      this.fedRuns = []
      if (val) {
        this.loadFedHistory()
      } else {
        this.loadTableRuns()
      }
    }
  },

  async created() {
    if (this.selectedSite) {
      await this.loadFedHistory()
    } else {
      await Promise.all([this.loadTimelineData(), this.loadAgentChartData()])
      this.loadTableRuns()
    }
  },

  methods: {
    async loadFedHistory() {
      this.runsLoading = true
      this.stale = false
      try {
        const data = await getFederationHistory(this.selectedSite)
        this.fedRuns = data.history || []
        this.stale = data.stale || false
        this.staleAsOf = data.as_of || null
      } catch (e) {
        console.error('Fed history load failed', e)
      } finally {
        this.runsLoading = false
      }
    },

    async loadTimelineData() {
      this.tlLoading = true
      try {
        const resp = await getJobRuns({ limit: 500, page: 1 })
        this.tlRuns = resp.data ?? []
        for (const r of this.tlRuns) {
          if (!this.tlJobMeta[r.job_id]) {
            this.tlJobMeta[r.job_id] = { name: r.job_id }
          }
        }
      } catch (e) {
        console.error('Timeline load failed', e)
      } finally {
        this.tlLoading = false
      }
    },

    async loadAgentChartData() {
      this.acLoading = true
      const cutoff = new Date(Date.now() - 14 * 86400000).toISOString()
      try {
        const resp = await getJobRuns({ limit: 1000, page: 1, after: cutoff })
        this.acRuns = resp.data ?? []
      } catch (e) {
        console.error('Agent chart load failed', e)
        this.acRuns = this.tlRuns
      } finally {
        this.acLoading = false
      }
    },

    async loadTableRuns(page = 1) {
      this.runsLoading = true
      try {
        const params = {
          page,
          limit: 25,
          search: this.search || undefined,
          status: this.filterStatus || undefined,
          job_id: this.filterJobId || undefined,
          agent_id: this.filterAgentId || undefined
        }
        const resp = await getJobRuns(params)
        this.runs = resp.data ?? []
        this.pagination = { page: resp.page, pages: resp.pages, total: resp.total, limit: resp.limit }
      } catch (e) {
        console.error('Run table load failed', e)
      } finally {
        this.runsLoading = false
      }
    },

    onSelectJob(jobId) { this.filterJobId = jobId; this.loadTableRuns() },
    onSelectAgent(agentId) { this.filterAgentId = agentId; this.loadTableRuns() },
    onFilterChange() { this.loadTableRuns(1) },

    clearFilters() {
      this.search = ''
      this.filterStatus = ''
      this.filterJobId = null
      this.filterAgentId = null
      this.loadTableRuns(1)
    },

    onPageChange(page) { this.loadTableRuns(page) },

    openOutput(run) {
      this.outputModal = { visible: true, jobId: run.job_id, text: run.output ?? '(no output)' }
    },

    fmtTime(ts) {
      if (!ts) return '—'
      return new Date(ts).toLocaleString(undefined, {
        month: 'short', day: 'numeric',
        hour: '2-digit', minute: '2-digit'
      })
    },

    fmtDuration(start, end) {
      if (!start || !end) return '—'
      const secs = Math.round((new Date(end) - new Date(start)) / 1000)
      if (secs < 60) return `${secs}s`
      return `${Math.floor(secs / 60)}m ${secs % 60}s`
    },

    fmtStaleTime(iso) {
      if (!iso) return 'an unknown time ago'
      const secs = Math.floor((Date.now() - new Date(iso)) / 1000)
      if (secs < 60) return `${secs}s ago`
      if (secs < 3600) return `${Math.floor(secs / 60)}m ago`
      return `${Math.floor(secs / 3600)}h ago`
    }
  }
}
</script>

<style scoped>
.history-view {
  padding: 24px;
  max-width: 1280px;
  margin: 0 auto;
  font-family: 'JetBrains Mono', 'Fira Code', ui-monospace, monospace;
}

.history-header { margin-bottom: 20px; }
.history-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text, #e8e8f0);
  letter-spacing: -0.01em;
}

.history-section { margin-bottom: 18px; }

.stale-banner {
  background: rgba(255, 167, 38, 0.12);
  border: 1px solid rgba(255, 167, 38, 0.4);
  color: #ffa726;
  padding: 0.5rem 0.85rem;
  border-radius: 4px;
  font-size: 0.85rem;
}

.section-label {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-muted, #6b6b8a);
}

.history-table-card {
  background: var(--card-bg, #1a1a2e);
  border: 1px solid var(--border, #2e2e4a);
  border-radius: 4px;
  padding: 18px 20px 16px;
}

.table-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.table-filters {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
  margin-left: auto;
}

.filter-input,
.filter-select {
  background: var(--input-bg, #12122a);
  border: 1px solid var(--border, #2e2e4a);
  border-radius: 3px;
  color: var(--text, #e8e8f0);
  font-family: inherit;
  font-size: 11px;
  padding: 5px 9px;
  outline: none;
  transition: border-color 0.15s;
}
.filter-input:focus,
.filter-select:focus  { border-color: var(--accent, #4f8ef7); }
.filter-input         { width: 180px; }

.btn-clear {
  background: none;
  border: 1px solid var(--border, #2e2e4a);
  border-radius: 3px;
  color: var(--text-muted, #6b6b8a);
  font-family: inherit;
  font-size: 10px;
  padding: 5px 9px;
  cursor: pointer;
  transition: color 0.12s, border-color 0.12s;
}
.btn-clear:hover { color: var(--fail, #ff4d6d); border-color: var(--fail, #ff4d6d); }

.filter-pills { display: flex; gap: 8px; margin-bottom: 10px; }
.pill {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--surface2, #22223a);
  border: 1px solid var(--accent, #4f8ef7);
  border-radius: 20px;
  padding: 3px 10px;
  font-size: 10px;
  color: var(--accent, #4f8ef7);
}
.pill-x {
  background: none;
  border: none;
  color: var(--accent, #4f8ef7);
  font-size: 9px;
  cursor: pointer;
  padding: 0;
  line-height: 1;
}

.table-loading {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-muted, #6b6b8a);
  font-size: 12px;
  padding: 20px 0;
}
.tl-loading-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  background: var(--accent, #4f8ef7);
  animation: blink 1s step-start infinite;
}
@keyframes blink { 0%,100%{opacity:1} 50%{opacity:0.15} }
.table-empty {
  color: var(--text-muted, #6b6b8a);
  font-size: 12px;
  padding: 20px 0;
  text-align: center;
}

.table-wrap { overflow-x: auto; }

.runs-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 11px;
}
.runs-table th {
  text-align: left;
  padding: 7px 10px;
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-muted, #6b6b8a);
  border-bottom: 1px solid var(--border, #2e2e4a);
}
.runs-table td {
  padding: 8px 10px;
  border-bottom: 1px solid var(--border-subtle, #252538);
  vertical-align: middle;
}
.run-row:last-child td    { border-bottom: none; }
.run-row:hover td         { background: var(--surface-hover, #22223a); }
.run-row.failed td        { background: rgba(255, 77, 109, 0.03); }

.run-job, .run-agent      { color: var(--text, #e8e8f0); }
.run-time, .run-dur       { color: var(--text-muted, #6b6b8a); white-space: nowrap; }

.status-badge {
  display: inline-block;
  padding: 2px 7px;
  border-radius: 2px;
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}
.status-badge.completed { background: rgba(62,207,142,0.12); color: var(--success, #3ecf8e); }
.status-badge.failed    { background: rgba(255, 77,109,0.12); color: var(--fail, #ff4d6d); }
.status-badge.running   { background: rgba(245,166, 35,0.12); color: var(--running-color, #f5a623); }

.btn-output {
  background: none;
  border: 1px solid var(--border, #2e2e4a);
  border-radius: 2px;
  color: var(--text-muted, #6b6b8a);
  font-family: inherit;
  font-size: 10px;
  padding: 3px 8px;
  cursor: pointer;
  transition: color 0.1s, border-color 0.1s;
}
.btn-output:hover { color: var(--accent, #4f8ef7); border-color: var(--accent, #4f8ef7); }

.table-pagination { margin-top: 14px; }

.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
  backdrop-filter: blur(2px);
}
.modal-box {
  background: var(--surface, #1a1a2e);
  border: 1px solid var(--border, #2e2e4a);
  border-radius: 4px;
  width: min(680px, 92vw);
  max-height: 70vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 16px 48px rgba(0,0,0,0.6);
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 18px;
  border-bottom: 1px solid var(--border, #2e2e4a);
  flex-shrink: 0;
}
.modal-title {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--text-muted, #6b6b8a);
}
.modal-close {
  background: none;
  border: none;
  color: var(--text-muted, #6b6b8a);
  font-size: 14px;
  cursor: pointer;
  padding: 0;
  line-height: 1;
}
.modal-close:hover { color: var(--text, #e8e8f0); }
.modal-output {
  padding: 16px 18px;
  overflow: auto;
  font-family: inherit;
  font-size: 11px;
  line-height: 1.6;
  color: var(--text, #e8e8f0);
  white-space: pre-wrap;
  word-break: break-all;
  flex: 1;
}

[data-theme="light"] .history-view {
  --card-bg: #ffffff;
  --border: #e0e0ec;
  --border-subtle: #ebebf5;
  --text: #1a1a2e;
  --text-muted: #888899;
  --surface: #f8f8fc;
  --surface-hover: #f0f0fa;
  --surface2: #eeeefc;
  --input-bg: #f4f4fa;
  --success: #1ea87a;
  --fail: #e03050;
  --running-color: #d4851a;
}
</style>
