<template>
  <div class="agent-run-chart">
    <div class="arc-header">
      <span class="arc-label">Agent Run Breakdown</span>
      <span class="arc-window">last {{ days }} days</span>
    </div>

    <div v-if="loading" class="arc-loading">
      <span class="arc-loading-dot"></span> loading agent stats…
    </div>

    <div v-else-if="agentData.length === 0" class="arc-empty">
      No agent data available.
    </div>

    <div v-else class="arc-grid">
      <div
        v-for="agent in agentData"
        :key="agent.agentId"
        class="arc-card"
        :class="{ 'arc-card--selected': selectedAgent === agent.agentId }"
        @click="$emit('select-agent', agent.agentId === selectedAgent ? null : agent.agentId)"
      >
        <div class="arc-agent-id">{{ agent.agentId }}</div>
        <div class="arc-agent-host">{{ agent.hostname || agent.agentId }}</div>

        <!-- SVG bar chart -->
        <svg
          :viewBox="`0 0 ${chartW} ${chartH}`"
          class="arc-svg"
          preserveAspectRatio="none"
        >
          <g v-for="(col, i) in agent.cols" :key="i">
            <!-- fail segment (stacked on top) -->
            <rect
              v-if="col.failH > 0"
              :x="colX(i)"
              :y="chartH - col.okH - col.failH"
              :width="colW - 1"
              :height="col.failH"
              rx="1"
              class="bar-fail"
            >
              <title>{{ col.label }}: {{ col.ok }} ok / {{ col.fail }} failed</title>
            </rect>
            <!-- ok segment -->
            <rect
              v-if="col.okH > 0"
              :x="colX(i)"
              :y="chartH - col.okH"
              :width="colW - 1"
              :height="col.okH"
              rx="1"
              class="bar-ok"
            >
              <title>{{ col.label }}: {{ col.ok }} ok / {{ col.fail }} failed</title>
            </rect>
          </g>
        </svg>

        <div class="arc-footer">
          <span class="arc-total">{{ agent.totalRuns }} runs</span>
          <span class="arc-ok-val">{{ agent.totalOk }}<svg width="9" height="9" viewBox="0 0 9 9" style="vertical-align:middle"><polyline points="1.5,5 3.5,7 7.5,2" stroke="currentColor" stroke-width="1.5" fill="none" stroke-linecap="round"/></svg></span>
          <span class="arc-fail-val">{{ agent.totalFail }}<svg width="9" height="9" viewBox="0 0 9 9" style="vertical-align:middle"><line x1="2" y1="2" x2="7" y2="7" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="7" y1="2" x2="2" y2="7" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg></span>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'AgentRunChart',

  props: {
    // Array of { agentId, hostname, runs: [{ status, started_at }] }
    agents: {
      type: Array,
      default: () => []
    },
    loading: {
      type: Boolean,
      default: false
    },
    selectedAgent: {
      type: String,
      default: null
    },
    days: {
      type: Number,
      default: 14
    }
  },

  emits: ['select-agent'],

  data() {
    return {
      chartW: 200,
      chartH: 52
    }
  },

  computed: {
    colW() {
      return Math.max(this.chartW / this.days - 1, 2)
    },

    agentData() {
      const maxDate = new Date()
      const minDate = new Date(maxDate - this.days * 86400000)

      return this.agents.map(agent => {
        // Bucket runs into day slots
        const buckets = Array.from({ length: this.days }, (_, i) => {
          const d = new Date(minDate.getTime() + i * 86400000)
          return {
            label: d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
            ok: 0,
            fail: 0
          }
        })

        for (const run of (agent.runs || [])) {
          if (!run.started_at) continue
          const rt = new Date(run.started_at).getTime()
          if (rt < minDate.getTime()) continue
          const idx = Math.floor((rt - minDate.getTime()) / 86400000)
          if (idx < 0 || idx >= this.days) continue
          if (run.status === 'completed') buckets[idx].ok++
          else if (run.status === 'failed') buckets[idx].fail++
        }

        const maxTotal = Math.max(...buckets.map(b => b.ok + b.fail), 1)
        const cols = buckets.map(b => ({
          ...b,
          okH: Math.round((b.ok / maxTotal) * (this.chartH - 4)),
          failH: Math.round((b.fail / maxTotal) * (this.chartH - 4))
        }))

        const totalOk = buckets.reduce((s, b) => s + b.ok, 0)
        const totalFail = buckets.reduce((s, b) => s + b.fail, 0)

        return {
          agentId: agent.agentId,
          hostname: agent.hostname,
          cols,
          totalRuns: totalOk + totalFail,
          totalOk,
          totalFail
        }
      })
    }
  },

  methods: {
    colX(i) {
      return i * (this.chartW / this.days)
    }
  }
}
</script>

<style scoped>
/* ── Layout ───────────────────────────────────────────────── */
.agent-run-chart {
  font-family: var(--font-mono);
}

.arc-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.arc-label {
  font-family: var(--font-body);
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-muted);
}
.arc-window {
  font-family: var(--font-body);
  font-size: 0.72rem;
  color: var(--text-muted);
}

/* Loading / empty */
.arc-loading {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-muted);
  font-family: var(--font-body);
  font-size: 0.82rem;
  padding: 20px 0;
}
.arc-loading-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  background: var(--accent);
  animation: blink 1s step-start infinite;
  flex-shrink: 0;
}
@keyframes blink { 0%,100%{opacity:1} 50%{opacity:0.15} }
.arc-empty {
  color: var(--text-muted);
  font-family: var(--font-body);
  font-size: 0.82rem;
  padding: 20px 0;
  text-align: center;
}

/* ── Card grid ────────────────────────────────────────────── */
.arc-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}

.arc-card {
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  padding: 14px 16px 12px;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.arc-card:hover          { border-color: var(--accent-border); background: var(--bg-elevated); }
.arc-card--selected      { border-color: var(--accent-border); background: var(--bg-elevated); }

.arc-agent-id {
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--accent);
  margin-bottom: 2px;
}
.arc-agent-host {
  font-size: 0.68rem;
  color: var(--text-muted);
  margin-bottom: 10px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── SVG chart ────────────────────────────────────────────── */
.arc-svg {
  width: 100%;
  height: 52px;
  display: block;
  margin-bottom: 8px;
  overflow: visible;
}
.bar-ok {
  fill: var(--color-success);
  transition: opacity 0.1s;
}
.bar-fail {
  fill: var(--color-error);
  transition: opacity 0.1s;
}
.arc-card:hover .bar-ok   { opacity: 0.8; }
.arc-card:hover .bar-fail { opacity: 0.8; }

/* ── Footer stats ─────────────────────────────────────────── */
.arc-footer {
  display: flex;
  justify-content: space-between;
  font-size: 0.68rem;
  padding-top: 6px;
  border-top: 1px solid var(--border-subtle);
}
.arc-total    { color: var(--text-muted); }
.arc-ok-val   { color: var(--color-success); font-weight: 700; display: flex; align-items: center; gap: 2px; }
.arc-fail-val { color: var(--color-error);   font-weight: 700; display: flex; align-items: center; gap: 2px; }
</style>
