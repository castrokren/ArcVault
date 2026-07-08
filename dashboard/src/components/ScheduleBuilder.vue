<template>
  <div class="schedule-builder">

    <!-- Mode tabs -->
    <div class="sb-tabs">
      <button
        v-for="m in modes"
        :key="m.key"
        type="button"
        class="sb-tab"
        :class="{ active: mode === m.key }"
        @click="setMode(m.key)"
      >
        {{ m.label }}
      </button>
    </div>

    <!-- Off -->
    <div v-if="mode === 'off'" class="sb-body sb-off">
      Runs manually only — no schedule set.
    </div>

    <!-- Interval -->
    <div v-if="mode === 'interval'" class="sb-body sb-row">
      <span class="sb-label">Every</span>
      <select v-model="interval.every" @change="emitValue">
        <option v-for="n in intervalOptions" :key="n" :value="n">{{ n }} minutes</option>
      </select>
    </div>

    <!-- Daily -->
    <div v-if="mode === 'daily'" class="sb-body sb-row">
      <span class="sb-label">At</span>
      <select v-model="daily.hour" @change="emitValue">
        <option v-for="h in 24" :key="h - 1" :value="h - 1">{{ pad(h - 1) }}</option>
      </select>
      <span class="sb-sep">:</span>
      <select v-model="daily.minute" @change="emitValue">
        <option v-for="m in minuteOptions" :key="m" :value="m">{{ pad(m) }}</option>
      </select>
    </div>

    <!-- Weekly -->
    <div v-if="mode === 'weekly'" class="sb-body sb-weekly">
      <div class="sb-days">
        <button
          v-for="(d, i) in dayNames"
          :key="i"
          type="button"
          class="day-btn"
          :class="{ active: weekly.days.includes(i) }"
          @click="toggleDay(i)"
        >
          {{ d.short }}
        </button>
      </div>
      <div class="sb-row">
        <span class="sb-label">at</span>
        <select v-model="weekly.hour" @change="emitValue">
          <option v-for="h in 24" :key="h - 1" :value="h - 1">{{ pad(h - 1) }}</option>
        </select>
        <span class="sb-sep">:</span>
        <select v-model="weekly.minute" @change="emitValue">
          <option v-for="m in minuteOptions" :key="m" :value="m">{{ pad(m) }}</option>
        </select>
      </div>
    </div>

    <!-- Monthly -->
    <div v-if="mode === 'monthly'" class="sb-body sb-row">
      <span class="sb-label">Day</span>
      <select v-model="monthly.day" @change="emitValue">
        <option v-for="d in 28" :key="d" :value="d">{{ d }}</option>
      </select>
      <span class="sb-label sb-at">at</span>
      <select v-model="monthly.hour" @change="emitValue">
        <option v-for="h in 24" :key="h - 1" :value="h - 1">{{ pad(h - 1) }}</option>
      </select>
      <span class="sb-sep">:</span>
      <select v-model="monthly.minute" @change="emitValue">
        <option v-for="m in minuteOptions" :key="m" :value="m">{{ pad(m) }}</option>
      </select>
    </div>

    <!-- Custom -->
    <div v-if="mode === 'custom'" class="sb-body sb-row">
      <input
        :value="customExpr"
        @input="onCustomInput"
        placeholder="0 2 * * *"
        class="sb-custom-input"
        spellcheck="false"
        autocomplete="off"
      />
    </div>

    <!-- Preview -->
    <div v-if="mode !== 'off' && currentCron" class="sb-preview">
      <code class="sb-cron">{{ currentCron }}</code>
      <span v-if="previewText" class="sb-preview-text">— {{ previewText }}</span>
    </div>
    <div v-else-if="mode === 'weekly' && weekly.days.length === 0" class="sb-preview sb-warn">
      Select at least one day.
    </div>

  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { cronPreview } from '../api'
import { parseCronParts } from '../utils/cron.js'

const props = defineProps({
  modelValue: { type: String, default: '' }
})
const emit = defineEmits(['update:modelValue'])

// ── Constants ────────────────────────────────────────────

const modes = [
  { key: 'off',      label: 'Off' },
  { key: 'interval', label: 'Interval' },
  { key: 'daily',    label: 'Daily' },
  { key: 'weekly',   label: 'Weekly' },
  { key: 'monthly',  label: 'Monthly' },
  { key: 'custom',   label: 'Custom' },
]

const dayNames = [
  { short: 'Su', full: 'Sunday' },
  { short: 'Mo', full: 'Monday' },
  { short: 'Tu', full: 'Tuesday' },
  { short: 'We', full: 'Wednesday' },
  { short: 'Th', full: 'Thursday' },
  { short: 'Fr', full: 'Friday' },
  { short: 'Sa', full: 'Saturday' },
]

const intervalOptions = [5, 10, 15, 20, 30, 60]
const minuteOptions   = [0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55]

// ── Per-mode sub-state ───────────────────────────────────

const mode     = ref('off')
const interval = ref({ every: 30 })
const daily    = ref({ hour: 2, minute: 0 })
const weekly   = ref({ days: [1], hour: 2, minute: 0 }) // Mon default
const monthly  = ref({ day: 1, hour: 2, minute: 0 })
const customExpr = ref('')

// ── Helpers ──────────────────────────────────────────────

function pad(n) { return String(n).padStart(2, '0') }

function fmtTime(h, m) {
  const suffix = h >= 12 ? 'PM' : 'AM'
  const h12    = h === 0 ? 12 : h > 12 ? h - 12 : h
  const mStr   = m === 0 ? '' : `:${pad(m)}`
  return `${h12}${mStr} ${suffix}`
}

// ── Parse incoming cron string → mode + sub-state ────────

function parseValue(val) {
  const p = parseCronParts(val)
  if (!p) {
    if (!val || !val.trim()) {
      mode.value = 'off'
    } else {
      mode.value = 'custom'
      customExpr.value = val
    }
    return
  }

  const { min, hour, dom, month, dow } = p

  // Interval: */N * * * *
  if (min.startsWith('*/') && hour === '*' && dom === '*' && month === '*' && dow === '*') {
    const n = parseInt(min.slice(2), 10)
    mode.value = 'interval'
    interval.value.every = intervalOptions.includes(n) ? n : 30
    return
  }

  // Daily: M H * * *
  if (!min.includes('*') && !min.includes('/') &&
      !hour.includes('*') && !hour.includes('/') &&
      dom === '*' && month === '*' && dow === '*') {
    mode.value = 'daily'
    daily.value.hour   = parseInt(hour, 10)
    daily.value.minute = parseInt(min,  10)
    return
  }

  // Weekly: M H * * D  or  M H * * D,D,...
  if (!min.includes('*') && !min.includes('/') &&
      !hour.includes('*') && !hour.includes('/') &&
      dom === '*' && month === '*' &&
      !dow.includes('*') && !dow.includes('/')) {
    const days = dow.split(',').map(Number)
    if (days.every(d => d >= 0 && d <= 6)) {
      mode.value = 'weekly'
      weekly.value.days   = days
      weekly.value.hour   = parseInt(hour, 10)
      weekly.value.minute = parseInt(min,  10)
      return
    }
  }

  // Monthly: M H D * *
  if (!min.includes('*') && !min.includes('/') &&
      !hour.includes('*') && !hour.includes('/') &&
      !dom.includes('*') && !dom.includes('/') &&
      month === '*' && dow === '*') {
    mode.value = 'monthly'
    monthly.value.day    = parseInt(dom,  10)
    monthly.value.hour   = parseInt(hour, 10)
    monthly.value.minute = parseInt(min,  10)
    return
  }

  // Fallback: custom
  mode.value = 'custom'
  customExpr.value = val
}

// ── Computed cron string ──────────────────────────────────

const currentCron = computed(() => {
  switch (mode.value) {
    case 'off':      return ''
    case 'interval': return `*/${interval.value.every} * * * *`
    case 'daily':    return `${daily.value.minute} ${daily.value.hour} * * *`
    case 'weekly': {
      if (weekly.value.days.length === 0) return ''
      const d = weekly.value.days.slice().sort((a, b) => a - b).join(',')
      return `${weekly.value.minute} ${weekly.value.hour} * * ${d}`
    }
    case 'monthly':  return `${monthly.value.minute} ${monthly.value.hour} ${monthly.value.day} * *`
    case 'custom':   return customExpr.value.trim()
    default:         return ''
  }
})

// ── Human-readable preview ────────────────────────────────

const previewText = computed(() => {
  const cron = currentCron.value
  if (!cron) return ''

  // Weekly multi-day: build ourselves since cronPreview handles only single-day
  if (mode.value === 'weekly' && weekly.value.days.length > 0) {
    const time = fmtTime(weekly.value.hour, weekly.value.minute)
    if (weekly.value.days.length === 7) return `Every day at ${time}`
    const names = weekly.value.days.slice().sort((a, b) => a - b).map(d => dayNames[d].full)
    return `Every ${names.join(', ')} at ${time}`
  }

  return cronPreview(cron) || ''
})

// ── Initialise ───────────────────────────────────────────

parseValue(props.modelValue)

// Re-parse when parent drives a new value (e.g. editing an existing record)
watch(() => props.modelValue, (val) => {
  if (val !== currentCron.value) parseValue(val)
})

// ── Handlers ─────────────────────────────────────────────

function setMode(m) {
  mode.value = m
  emitValue()
}

function toggleDay(i) {
  const idx = weekly.value.days.indexOf(i)
  if (idx === -1) {
    weekly.value.days.push(i)
  } else if (weekly.value.days.length > 1) {
    // Keep at least one day selected
    weekly.value.days.splice(idx, 1)
  }
  emitValue()
}

function onCustomInput(e) {
  customExpr.value = e.target.value
  emitValue()
}

function emitValue() {
  emit('update:modelValue', currentCron.value)
}
</script>

<style scoped>
.schedule-builder {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

/* ── Tabs ───────────────────────────────────────────────── */
.sb-tabs {
  display: flex;
  gap: 0.25rem;
  flex-wrap: wrap;
}

.sb-tab {
  padding: 0.22rem 0.65rem;
  border-radius: 4px;
  border: 1px solid var(--border-default);
  background: transparent;
  color: var(--text-secondary);
  font-family: var(--font-body);
  font-size: 0.78rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
  line-height: 1.4;
}
.sb-tab:hover {
  background: var(--bg-elevated);
  border-color: var(--border-strong);
  color: var(--text-primary);
}
.sb-tab.active {
  background: var(--accent-dim);
  border-color: var(--accent-border);
  color: var(--accent);
}

/* ── Body ───────────────────────────────────────────────── */
.sb-body {
  padding: 0.5rem 0.65rem;
  background: var(--bg-input);
  border: 1px solid var(--border-default);
  border-radius: 5px;
}

.sb-off {
  color: var(--text-muted);
  font-family: var(--font-body);
  font-size: 0.82rem;
}

/* ── Row layout ─────────────────────────────────────────── */
.sb-row {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  flex-wrap: wrap;
}

.sb-label {
  color: var(--text-secondary);
  font-family: var(--font-body);
  font-size: 0.82rem;
  white-space: nowrap;
}

.sb-at { margin-left: 0.35rem; }

.sb-sep {
  color: var(--text-muted);
  font-weight: 600;
  font-size: 0.9rem;
}

.sb-body select {
  padding: 0.22rem 0.5rem;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 4px;
  color: var(--text-primary);
  font-family: var(--font-body);
  font-size: 0.82rem;
  cursor: pointer;
  outline: none;
  transition: border-color 0.15s;
}
.sb-body select:focus { border-color: var(--accent); box-shadow: var(--glow-accent); }

/* ── Weekly day picker ──────────────────────────────────── */
.sb-weekly {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.sb-days {
  display: flex;
  gap: 0.25rem;
  flex-wrap: wrap;
}

.day-btn {
  width: 2rem;
  height: 2rem;
  border-radius: 50%;
  border: 1px solid var(--border-default);
  background: var(--bg-elevated);
  color: var(--text-secondary);
  font-family: var(--font-body);
  font-size: 0.72rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
}
.day-btn:hover {
  border-color: var(--border-strong);
  color: var(--text-primary);
}
.day-btn.active {
  background: var(--accent-dim);
  border-color: var(--accent-border);
  color: var(--accent);
}

/* ── Custom input ───────────────────────────────────────── */
.sb-custom-input {
  width: 100%;
  padding: 0.28rem 0.5rem;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 4px;
  color: var(--text-primary);
  font-family: var(--font-mono);
  font-size: 0.82rem;
  outline: none;
  transition: border-color 0.15s;
}
.sb-custom-input:focus { border-color: var(--accent); box-shadow: var(--glow-accent); }

/* ── Preview ────────────────────────────────────────────── */
.sb-preview {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.sb-cron {
  font-family: var(--font-mono);
  font-size: 0.78rem;
  color: var(--text-secondary);
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: 3px;
  padding: 0.1rem 0.4rem;
}

.sb-preview-text {
  color: var(--accent);
  font-family: var(--font-body);
  font-size: 0.78rem;
}

.sb-warn {
  color: var(--color-warning);
  font-family: var(--font-body);
  font-size: 0.78rem;
  background: var(--bg-warning);
  border-color: rgba(245, 158, 11, 0.3);
  padding: 0.3rem 0.65rem;
}
</style>
