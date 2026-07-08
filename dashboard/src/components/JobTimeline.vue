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
.job-timeline {
  position: relative;
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  padding: 1.1rem 1.25rem 0.85rem;
  font-family: var(--font-mono);
}

.tl-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 0.85rem;
  flex-wrap: wrap;
}

.tl-label {
  font-family: var(--font-body);
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-muted);
}

.tl-window {
  font-family: var(--font-body);
  font-size: 0.72rem;
  color: var(--text-muted);
  margin-left: auto;
}

.tl-legend {
  display: flex;
  gap: 0.75rem;
  font-family: var(--font-body);
  font-size: 0.72rem;
  color: var(--text-muted);
}
.leg-item { display: flex; align-items: center; gap: 4px; }
.leg-dot { width: 8px; height: 8px; border-radius: 2px; flex-shrink: 0; }
.leg-dot.completed { background: var(--color-success); }
.leg-dot.failed    { background: var(--color-error); }
.leg-dot.running   { background: var(--color-warning); }

.tl-loading {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  color: var(--text-muted);
  font-family: var(--font-body);
  font-size: 0.82rem;
  padding: 1.5rem 0;
}
.tl-loading-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  background: var(--accent);
  animation: blink 1s step-start infinite;
  flex-shrink: 0;
}
@keyframes blink { 0%,100% { opacity: 1; } 50% { opacity: 0.15; } }
.tl-empty {
  color: var(--text-muted);
  font-family: var(--font-body);
  font-size: 0.82rem;
  padding: 1.25rem 0;
  text-align: center;
}

.tl-rows { display: flex; flex-direction: column; }

.tl-row {
  display: grid;
  grid-template-columns: 160px 1fr 60px;
  align-items: center;
  gap: 0.85rem;
  padding: 0.55rem 0.5rem;
  border-top: 1px solid var(--border-subtle);
  cursor: pointer;
  border-radius: 4px;
  transition: background 0.12s;
}
.tl-row:hover   { background: var(--bg-elevated); }
.tl-row--active { background: var(--bg-elevated); border-color: var(--accent-border); }

.tl-job-info { display: flex; flex-direction: column; gap: 2px; overflow: hidden; }
.tl-job-name {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tl-job-agent {
  font-size: 0.7rem;
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tl-rail-wrap {
  position: relative;
  height: 24px;
  display: flex;
  align-items: center;
}
.tl-rail-line {
  position: absolute;
  inset: 50% 0 auto;
  transform: translateY(-50%);
  height: 1px;
  background: var(--border-default);
}
.tl-block {
  position: absolute;
  height: 16px;
  border-radius: 2px;
  transition: transform 0.1s;
}
.tl-block:hover { transform: scaleY(1.4); }
.tl-block.completed { background: var(--color-success); }
.tl-block.failed    { background: var(--color-error); }
.tl-block.running   {
  background: var(--color-warning);
  animation: runpulse 1.2s ease-in-out infinite;
}
@keyframes runpulse { 0%,100% { opacity: 1; } 50% { opacity: 0.4; } }

.tl-stats { display: flex; flex-direction: column; align-items: flex-end; gap: 2px; font-size: 0.72rem; font-weight: 700; }
.tl-stat-ok   { color: var(--color-success); display: flex; align-items: center; gap: 2px; }
.tl-stat-fail { color: var(--color-error);   display: flex; align-items: center; gap: 2px; }

.tl-tooltip {
  position: absolute;
  pointer-events: none;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 5px;
  padding: 0.45rem 0.65rem;
  font-family: var(--font-body);
  font-size: 0.78rem;
  z-index: 50;
  box-shadow: var(--shadow-md);
  white-space: nowrap;
  animation: fadeIn 0.08s ease;
}
@keyframes fadeIn { from { opacity: 0; transform: translateY(2px); } to { opacity: 1; transform: none; } }

.tt-status { font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; font-size: 0.7rem; margin-bottom: 2px; }
.tt-status.completed { color: var(--color-success); }
.tt-status.failed    { color: var(--color-error); }
.tt-status.running   { color: var(--color-warning); }

.tt-time { color: var(--text-primary); }
.tt-dur  { color: var(--text-muted); margin-top: 2px; }
</style>
