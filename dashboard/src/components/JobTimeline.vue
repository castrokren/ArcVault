<template>
  <div class="job-timeline">
    <div class="tl-header">
      <span class="tl-label">Job Timeline</span>
      <div class="tl-legend">
        <span class="leg-item"><span class="leg-dot completed"></span>Completed</span>
        <span class="leg-item"><span class="leg-dot failed"></span>Failed</span>
        <span class="leg-item"><span class="leg-dot running"></span>Running</span>
      </div>
      <span class="tl-window">last {{ windowLabel }}</span>
    </div>

    <div v-if="loading" class="tl-loading">
      <span class="tl-loading-dot"></span> loading run history…
    </div>

    <div v-else-if="jobRows.length === 0" class="tl-empty">
      No job runs found.
    </div>

    <div v-else class="tl-rows">
      <div
        v-for="row in jobRows"
        :key="row.jobId"
        class="tl-row"
        :class="{ 'tl-row--active': selectedJob === row.jobId }"
        @click="$emit('select-job', row.jobId === selectedJob ? null : row.jobId)"
      >
        <div class="tl-job-info">
          <span class="tl-job-name">{{ row.jobName }}</span>
          <span class="tl-job-agent">{{ row.agentId }}</span>
        </div>

        <div class="tl-rail-wrap" ref="railRefs">
          <div class="tl-rail-line"></div>
          <div
            v-for="run in row.runs"
            :key="run.id"
            class="tl-block"
            :class="run.status"
            :style="blockStyle(run, row.runs.length)"
            @mouseenter="onBlockEnter($event, run)"
            @mouseleave="onBlockLeave"
          ></div>
        </div>

        <div class="tl-stats">
          <span class="tl-stat-ok">{{ row.okCount }}<svg width="9" height="9" viewBox="0 0 9 9"><polyline points="1.5,5 3.5,7 7.5,2" stroke="currentColor" stroke-width="1.5" fill="none" stroke-linecap="round"/></svg></span>
          <span class="tl-stat-fail">{{ row.failCount }}<svg width="9" height="9" viewBox="0 0 9 9"><line x1="2" y1="2" x2="7" y2="7" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="7" y1="2" x2="2" y2="7" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg></span>
        </div>
      </div>
    </div>

    <!-- Floating tooltip -->
    <div
      v-if="tooltip.visible"
      class="tl-tooltip"
      :style="{ left: tooltip.x + 'px', top: tooltip.y + 'px' }"
    >
      <div class="tt-status" :class="tooltip.status">{{ tooltip.status }}</div>
      <div class="tt-time">{{ tooltip.time }}</div>
      <div v-if="tooltip.duration" class="tt-dur">{{ tooltip.duration }}</div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'JobTimeline',

  props: {
    // Array of { jobId, jobName, agentId, runs: [{ id, status, started_at, finished_at }] }
    jobRows: {
      type: Array,
      default: () => []
    },
    loading: {
      type: Boolean,
      default: false
    },
    selectedJob: {
      type: String,
      default: null
    },
    windowLabel: {
      type: String,
      default: '48 runs'
    }
  },

  emits: ['select-job'],

  data() {
    return {
      tooltip: {
        visible: false,
        x: 0,
        y: 0,
        status: '',
        time: '',
        duration: ''
      }
    }
  },

  methods: {
    blockStyle(run, total) {
      const W = 100 / Math.max(total, 1)
      const idx = run._idx ?? 0
      return {
        left: `${idx * W}%`,
        width: `${Math.max(W * 0.78, 0.4)}%`
      }
    },

    onBlockEnter(e, run) {
      const time = run.started_at
        ? new Date(run.started_at).toLocaleString(undefined, {
            month: 'short', day: 'numeric',
            hour: '2-digit', minute: '2-digit'
          })
        : '—'

      let duration = ''
      if (run.started_at && run.finished_at) {
        const secs = Math.round(
          (new Date(run.finished_at) - new Date(run.started_at)) / 1000
        )
        duration = secs >= 60
          ? `${Math.floor(secs / 60)}m ${secs % 60}s`
          : `${secs}s`
      }

      const rect = e.target.closest('.job-timeline').getBoundingClientRect()
      this.tooltip = {
        visible: true,
        x: e.clientX - rect.left + 12,
        y: e.clientY - rect.top - 48,
        status: run.status,
        time,
        duration
      }
    },

    onBlockLeave() {
      this.tooltip.visible = false
    }
  }
}
</script>

<style scoped>
/* ── Layout ───────────────────────────────────────────────── */
.job-timeline {
  position: relative;
  background: var(--card-bg, #1a1a2e);
  border: 1px solid var(--border, #2e2e4a);
  border-radius: 4px;
  padding: 18px 20px 14px;
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', ui-monospace, monospace;
}

.tl-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}

.tl-label {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-muted, #6b6b8a);
}

.tl-window {
  font-size: 10px;
  color: var(--text-muted, #6b6b8a);
  margin-left: auto;
  letter-spacing: 0.08em;
}

/* Legend */
.tl-legend {
  display: flex;
  gap: 12px;
  font-size: 10px;
  color: var(--text-muted, #6b6b8a);
  letter-spacing: 0.06em;
}
.leg-item { display: flex; align-items: center; gap: 4px; }
.leg-dot {
  width: 8px; height: 8px;
  border-radius: 1px;
  flex-shrink: 0;
}
.leg-dot.completed { background: var(--success, #3ecf8e); }
.leg-dot.failed    { background: var(--fail, #ff4d6d); }
.leg-dot.running   { background: var(--running-color, #f5a623); }

/* Loading / empty */
.tl-loading {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-muted, #6b6b8a);
  font-size: 12px;
  padding: 24px 0;
}
.tl-loading-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  background: var(--accent, #4f8ef7);
  animation: blink 1s step-start infinite;
}
@keyframes blink { 0%,100%{opacity:1} 50%{opacity:0.15} }
.tl-empty {
  color: var(--text-muted, #6b6b8a);
  font-size: 12px;
  padding: 20px 0;
  text-align: center;
}

/* ── Rows ─────────────────────────────────────────────────── */
.tl-rows {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.tl-row {
  display: grid;
  grid-template-columns: 160px 1fr 64px;
  align-items: center;
  gap: 14px;
  padding: 9px 8px;
  border-top: 1px solid var(--border, #2e2e4a);
  cursor: pointer;
  border-radius: 3px;
  transition: background 0.12s;
}
.tl-row:hover      { background: var(--surface-hover, #22223a); }
.tl-row--active    { background: var(--surface-active, #1e2240); }
.tl-row:last-child { /* no extra border */ }

.tl-job-info { display: flex; flex-direction: column; gap: 2px; overflow: hidden; }
.tl-job-name {
  font-size: 12px;
  font-weight: 600;
  color: var(--text, #e8e8f0);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tl-job-agent {
  font-size: 10px;
  color: var(--text-muted, #6b6b8a);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Rail ─────────────────────────────────────────────────── */
.tl-rail-wrap {
  position: relative;
  height: 26px;
  display: flex;
  align-items: center;
}
.tl-rail-line {
  position: absolute;
  inset: 50% 0 auto;
  transform: translateY(-50%);
  height: 1px;
  background: var(--border, #2e2e4a);
}
.tl-block {
  position: absolute;
  height: 18px;
  border-radius: 2px;
  transition: transform 0.1s;
}
.tl-block:hover { transform: scaleY(1.5); }
.tl-block.completed { background: var(--success, #3ecf8e); }
.tl-block.failed    { background: var(--fail, #ff4d6d); }
.tl-block.running   {
  background: var(--running-color, #f5a623);
  animation: runpulse 1.2s ease-in-out infinite;
}
@keyframes runpulse { 0%,100%{opacity:1} 50%{opacity:0.4} }

/* ── Stats ────────────────────────────────────────────────── */
.tl-stats {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  font-size: 10px;
  font-weight: 700;
}
.tl-stat-ok {
  color: var(--success, #3ecf8e);
  display: flex;
  align-items: center;
  gap: 2px;
}
.tl-stat-fail {
  color: var(--fail, #ff4d6d);
  display: flex;
  align-items: center;
  gap: 2px;
}

/* ── Tooltip ──────────────────────────────────────────────── */
.tl-tooltip {
  position: absolute;
  pointer-events: none;
  background: var(--tooltip-bg, #22223a);
  border: 1px solid var(--border, #2e2e4a);
  border-radius: 3px;
  padding: 7px 10px;
  font-size: 11px;
  z-index: 50;
  box-shadow: 0 4px 16px rgba(0,0,0,0.45);
  white-space: nowrap;
  animation: fadeIn 0.08s ease;
}
@keyframes fadeIn { from{opacity:0;transform:translateY(2px)} to{opacity:1;transform:none} }

.tt-status {
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  font-size: 10px;
  margin-bottom: 3px;
}
.tt-status.completed { color: var(--success, #3ecf8e); }
.tt-status.failed    { color: var(--fail, #ff4d6d); }
.tt-status.running   { color: var(--running-color, #f5a623); }

.tt-time { color: var(--text, #e8e8f0); }
.tt-dur  { color: var(--text-muted, #6b6b8a); margin-top: 2px; }

/* ── Light theme overrides ────────────────────────────────── */
[data-theme="light"] .job-timeline {
  --card-bg: #ffffff;
  --border: #e0e0ec;
  --text: #1a1a2e;
  --text-muted: #888899;
  --surface-hover: #f4f4fa;
  --surface-active: #eeeef8;
  --tooltip-bg: #f8f8fc;
  --success: #1ea87a;
  --fail: #e03050;
  --running-color: #d4851a;
}
</style>
