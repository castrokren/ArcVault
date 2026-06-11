<template>
  <div class="history-view">
    <div class="history-header">
      <h2 class="history-title">History</h2>
    </div>

    <!-- Stale data banner: federation site proxy data -->
    <div v-if="stale" class="stale-banner history-section">
      ⚠ Site data last synced {{ fmtStaleTime(staleAsOf) }}. Data may be outdated.
    </div>
    <!-- Stale data banner: local coordinator sync lag -->
    <div v-if="!selectedSite && syncStale" class="stale-banner history-section">
      ⚠ Federation sync in progress. Some history may be outdated.
    </div>

    <!-- ── Visualization panels (local only) ─────────────── -->
    <!-- Stat cards (local only, fed from already-fetched runs) -->
    <div v-if="!selectedSite && !tlLoading" class="history-stats history-section">
      <div class="stat-card">
        <span class="stat-label">Recent Runs</span>
        <span class="stat-value">{{ statTotals.total }}</span>
        <Sparkline class="stat-spark" :points="statBuckets.total" color="var(--accent-2)" />
      </div>
      <div class="stat-card">
        <span class="stat-label">Completed</span>
        <span class="stat-value">{{ statTotals.completed }}</span>
        <Sparkline class="stat-spark" :points="statBuckets.completed" color="var(--color-success)" />
      </div>
      <div class="stat-card">
        <span class="stat-label">Failed</span>
        <span class="stat-value">{{ statTotals.failed }}</span>
        <Sparkline class="stat-spark" :points="statBuckets.failed" color="var(--color-error)" />
      </div>
      <div class="stat-card">
        <span class="stat-label">Success Rate</span>
        <span class="stat-value">{{ statTotals.rate }}</span>
      </div>
    </div>

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

      <div v-if="runsLoading" class="table-loading" aria-busy="true">
        <div class="skeleton skeleton-line" style="width: 45%"></div>
        <div class="skeleton skeleton-line" style="width: 70%"></div>
        <div class="skeleton skeleton-line" style="width: 55%"></div>
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
              <th>Exit</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="run in displayRuns"
              :key="run.id"
              class="run-row"
              :class="[run.status, { 'run-selected': selectedRun && selectedRun.id === run.id }]"
              @click="selectRun(run)"
            >
              <td>
                <span class="status-badge" :class="run.status">{{ run.status }}</span>
              </td>
              <td class="run-job">{{ run.job_name || run.job_id }}</td>
              <td class="run-agent">{{ run.agent_hostname || run.agent_id }}</td>
              <td class="run-time">{{ fmtTime(run.started_at) }}</td>
              <td class="run-dur">{{ fmtDuration(run.started_at, run.finished_at) }}</td>
              <td class="run-exit" :class="{ 'exit-nonzero': run.exit_code !== 0 }">{{ run.exit_code }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Detail panel -->
      <div v-if="selectedRun" class="detail-panel">
        <div class="detail-header">
          <div class="detail-title-block">
            <span class="detail-job-name">{{ selectedRun.job_name || selectedRun.job_id }}</span>
            <span class="detail-subtitle">Run on {{ selectedRun.agent_hostname || selectedRun.agent_id }} · {{ fmtTime(selectedRun.started_at) }}</span>
          </div>
          <div class="detail-header-right">
            <span class="status-badge" :class="selectedRun.status">{{ selectedRun.status }}</span>
            <button class="detail-close" @click="selectedRun = null">✕</button>
          </div>
        </div>

        <div class="detail-stats">
          <div class="detail-stat">
            <span class="detail-stat-label">Started</span>
            <span class="detail-stat-value">{{ fmtTimeShort(selectedRun.started_at) }}</span>
          </div>
          <div class="detail-stat">
            <span class="detail-stat-label">Finished</span>
            <span class="detail-stat-value">{{ selectedRun.finished_at ? fmtTimeShort(selectedRun.finished_at) : '—' }}</span>
          </div>
          <div class="detail-stat">
            <span class="detail-stat-label">Duration</span>
            <span class="detail-stat-value">{{ fmtDuration(selectedRun.started_at, selectedRun.finished_at) }}</span>
          </div>
          <div class="detail-stat">
            <span class="detail-stat-label">Exit code</span>
            <span class="detail-stat-value" :class="{ 'exit-nonzero': selectedRun.exit_code !== 0 }">{{ selectedRun.exit_code }}</span>
          </div>
        </div>

        <div v-if="selectedRun.source_path || selectedRun.dest_path" class="detail-paths">
          <div v-if="selectedRun.source_path" class="detail-path-item">
            <span class="detail-stat-label">Source</span>
            <code class="detail-path">{{ selectedRun.source_path }}</code>
          </div>
          <div v-if="selectedRun.dest_path" class="detail-path-item">
            <span class="detail-stat-label">Destination</span>
            <code class="detail-path">{{ selectedRun.dest_path }}</code>
          </div>
        </div>

        <div class="detail-output-section">
          <span class="detail-stat-label">Output</span>
          <pre class="detail-output">{{ selectedRun.output || '(no output recorded)' }}</pre>
        </div>
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

  </div>
</template>

<script>
import JobTimeline from '../components/JobTimeline.vue'
import AgentRunChart from '../components/AgentRunChart.vue'
import Pagination from '../components/Pagination.vue'
import Sparkline from '../components/Sparkline.vue'
import { getJobRuns, getFederationHistory } from '../api'
import { fmtStaleTime } from '../utils/format.js'
import { inject, ref } from 'vue'
import { useFederationLag } from '../composables/useFederationLag.js'

export default {
  name: 'HistoryView',
  components: { JobTimeline, AgentRunChart, Pagination, Sparkline },

  setup() {
    const selectedSite = inject('selectedSite', ref(null))
    const { isStale: syncStale } = useFederationLag()
    return { selectedSite, syncStale }
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

      // ── Selected run for detail panel
      selectedRun: null
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

    statBuckets() {
      const days = 14
      const buckets = {
        total: Array(days).fill(0),
        completed: Array(days).fill(0),
        failed: Array(days).fill(0)
      }
      const now = Date.now()
      for (const r of this.tlRuns) {
        const age = Math.floor((now - new Date(r.started_at)) / 86400000)
        if (age < 0 || age >= days) continue
        const i = days - 1 - age
        buckets.total[i]++
        if (r.status === 'completed') buckets.completed[i]++
        if (r.status === 'failed') buckets.failed[i]++
      }
      return buckets
    },

    statTotals() {
      const completed = this.tlRuns.filter(r => r.status === 'completed').length
      const failed = this.tlRuns.filter(r => r.status === 'failed').length
      const done = completed + failed
      return {
        total: this.tlRuns.length,
        completed,
        failed,
        rate: done ? Math.round((completed / done) * 100) + '%' : '—'
      }
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
        const resp = await getJobRuns({
          page,
          limit: 25,
          jobID: this.filterJobId || undefined,
          agentID: this.filterAgentId || undefined,
          status: this.filterStatus || undefined
        })
        this.runs = resp.data ?? []
        this.pagination = { page: resp.page, pages: resp.pages, total: resp.total, limit: resp.limit }
        // Clear detail panel if selected run is no longer in current page
        if (this.selectedRun && !this.runs.find(r => r.id === this.selectedRun.id)) {
          this.selectedRun = null
        }
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

    selectRun(run) {
      this.selectedRun = this.selectedRun?.id === run.id ? null : run
    },

    fmtTime(ts) {
      if (!ts) return '—'
      return new Date(ts).toLocaleString(undefined, {
        month: 'short', day: 'numeric',
        hour: '2-digit', minute: '2-digit'
      })
    },

    fmtTimeShort(ts) {
      if (!ts) return '—'
      return new Date(ts).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' })
    },

    fmtDuration(start, end) {
      if (!start || !end) return '—'
      const secs = Math.round((new Date(end) - new Date(start)) / 1000)
      if (secs < 60) return `${secs}s`
      return `${Math.floor(secs / 60)}m ${secs % 60}s`
    },

    fmtStaleTime(iso) { return fmtStaleTime(iso) }
  }
}
</script>

<style scoped>
/* stale-banner covered globally */

.history-view { max-width: 1280px; margin: 0 auto; }
.history-header { margin-bottom: 1.25rem; }
.history-title {
  font-family: var(--font-display);
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.01em;
}

.history-section { margin-bottom: 1.1rem; }

.history-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 0.9rem;
}

.section-label {
  font-family: var(--font-body);
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-muted);
}

.history-table-card {
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  padding: 1.1rem 1.25rem 1rem;
}

.table-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
  flex-wrap: wrap;
}

.table-filters {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  flex-wrap: wrap;
  margin-left: auto;
}

.filter-input,
.filter-select {
  background: var(--bg-input);
  border: 1px solid var(--border-default);
  border-radius: 5px;
  color: var(--text-primary);
  font-family: var(--font-body);
  font-size: 0.82rem;
  padding: 0.3rem 0.65rem;
  outline: none;
  transition: border-color 0.15s;
}
.filter-input:focus,
.filter-select:focus { border-color: var(--accent); }
.filter-input { width: 185px; }

.btn-clear {
  background: none;
  border: 1px solid var(--border-default);
  border-radius: 5px;
  color: var(--text-muted);
  font-family: var(--font-body);
  font-size: 0.78rem;
  padding: 0.3rem 0.65rem;
  cursor: pointer;
  transition: color 0.12s, border-color 0.12s;
}
.btn-clear:hover { color: var(--color-error); border-color: var(--color-error); }

.filter-pills { display: flex; gap: 0.5rem; margin-bottom: 0.65rem; }
.pill {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  background: var(--accent-dim);
  border: 1px solid var(--accent-border);
  border-radius: 20px;
  padding: 0.2rem 0.65rem;
  font-family: var(--font-body);
  font-size: 0.75rem;
  color: var(--accent);
}
.pill-x {
  background: none;
  border: none;
  color: var(--accent);
  font-size: 0.7rem;
  cursor: pointer;
  padding: 0;
  line-height: 1;
}

.table-loading {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  color: var(--text-muted);
  font-family: var(--font-body);
  font-size: 0.82rem;
  padding: 1.25rem 0;
}
.tl-loading-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  background: var(--accent);
  animation: blink 1s step-start infinite;
  flex-shrink: 0;
}
@keyframes blink { 0%,100% { opacity: 1; } 50% { opacity: 0.15; } }

.table-empty {
  color: var(--text-muted);
  font-family: var(--font-body);
  font-size: 0.85rem;
  padding: 1.5rem 0;
  text-align: center;
}

.table-wrap { overflow-x: auto; }

.runs-table {
  width: 100%;
  border-collapse: collapse;
  font-family: var(--font-body);
  font-size: 0.85rem;
}
.runs-table th {
  text-align: left;
  padding: 0.5rem 0.65rem;
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border-default);
}
.runs-table td {
  padding: 0.55rem 0.65rem;
  border-bottom: 1px solid var(--border-subtle);
  vertical-align: middle;
}
.run-row:last-child td { border-bottom: none; }
.run-row:hover td      { background: var(--bg-elevated); }
.run-row.failed td     { background: rgba(255, 92, 122, 0.02); }

.run-job               { color: var(--text-primary); font-weight: 500; }
.run-agent             { color: var(--text-muted); font-size: 0.82rem; }
.run-time, .run-dur    { color: var(--text-muted); white-space: nowrap; }
.run-exit              { color: var(--text-muted); font-family: var(--font-mono); font-size: 0.8rem; }
.exit-nonzero          { color: var(--color-error); font-weight: 600; }

.run-row               { cursor: pointer; }
.run-row.run-selected td { background: var(--accent-dim); }

/* Use global .badge where possible; status-badge is an alias kept for this view */
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.2rem 0.55rem;
  border-radius: 4px;
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}
.status-badge.completed { background: var(--bg-success); color: var(--color-success); }
.status-badge.failed    { background: var(--bg-error);   color: var(--color-error);   }
.status-badge.running   { background: var(--bg-running); color: var(--color-running); }

.btn-output {
  background: none;
  border: 1px solid var(--border-default);
  border-radius: 4px;
  color: var(--text-muted);
  font-family: var(--font-body);
  font-size: 0.78rem;
  padding: 0.2rem 0.6rem;
  cursor: pointer;
  transition: color 0.1s, border-color 0.1s;
}
.btn-output:hover { color: var(--accent); border-color: var(--accent-border); }

.table-pagination { margin-top: 0.85rem; }

/* ── Detail panel ───────────────────────────────────── */
.detail-panel {
  margin-top: 0.75rem;
  border: 1px solid var(--accent-border);
  border-radius: 8px;
  background: var(--bg-card);
  overflow: hidden;
}

.detail-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  background: var(--bg-elevated);
  border-bottom: 1px solid var(--border-subtle);
  gap: 0.75rem;
}

.detail-title-block {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.detail-job-name {
  font-family: var(--font-body);
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
}

.detail-subtitle {
  font-family: var(--font-body);
  font-size: 0.75rem;
  color: var(--text-muted);
}

.detail-header-right {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  flex-shrink: 0;
}

.detail-close {
  background: none;
  border: none;
  color: var(--text-muted);
  font-size: 0.85rem;
  cursor: pointer;
  padding: 0.1rem 0.3rem;
  border-radius: 3px;
  line-height: 1;
  transition: color 0.1s;
}
.detail-close:hover { color: var(--text-primary); }

.detail-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  border-bottom: 1px solid var(--border-subtle);
}

.detail-stat {
  padding: 0.65rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  border-right: 1px solid var(--border-subtle);
}
.detail-stat:last-child { border-right: none; }

.detail-stat-label {
  font-family: var(--font-body);
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-muted);
}

.detail-stat-value {
  font-family: var(--font-body);
  font-size: 0.85rem;
  color: var(--text-primary);
}

.detail-paths {
  display: flex;
  gap: 2rem;
  padding: 0.65rem 1rem;
  border-bottom: 1px solid var(--border-subtle);
  flex-wrap: wrap;
}

.detail-path-item {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.detail-path {
  font-family: var(--font-mono);
  font-size: 0.78rem;
  color: var(--text-primary);
  background: none;
}

.detail-output-section {
  padding: 0.65rem 1rem 0.85rem;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.detail-output {
  font-family: var(--font-mono);
  font-size: 0.78rem;
  line-height: 1.65;
  color: var(--text-primary);
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: 5px;
  padding: 0.65rem 0.85rem;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 200px;
  overflow: auto;
  margin: 0;
}
</style>
