<template>
  <div class="alerts-container">
    <h1>Alert Rules & History</h1>

    <!-- Alert Rules Section -->
    <div class="section rules-section">
      <h2>Alert Rules</h2>

      <!-- Create Rule Form (Admin Only) -->
      <div v-if="isAdmin" class="create-rule-form">
        <div class="form-group">
          <input v-model="newRule.jobId" placeholder="Job ID (leave blank for all)" />
        </div>
        <div class="form-group">
          <select v-model="newRule.ruleType">
            <option value="">Select rule type</option>
            <option value="on_failure">On Failure</option>
            <option value="duration_exceeded">Duration Exceeded</option>
            <option value="missed_schedule">Missed Schedule</option>
          </select>
        </div>
        <div class="form-group">
          <input v-model.number="newRule.threshold" type="number" placeholder="Threshold (seconds)" />
        </div>
        <div class="form-group checkbox">
          <label>
            <input v-model="newRule.enabled" type="checkbox" />
            Enabled
          </label>
        </div>
        <button @click="createRule" :disabled="!newRule.ruleType">Create Rule</button>
      </div>

      <!-- Rules Table -->
      <div class="rules-table">
        <table>
          <thead>
            <tr>
              <th>Job</th>
              <th>Rule Type</th>
              <th>Threshold (s)</th>
              <th>Enabled</th>
              <th v-if="isAdmin">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="rules.length === 0">
              <td colspan="5" class="empty-state">No alert rules configured</td>
            </tr>
            <tr v-for="rule in rules" :key="rule.id">
              <td>{{ rule.job_id || '(All Jobs)' }}</td>
              <td>{{ formatRuleType(rule.rule_type) }}</td>
              <td>{{ rule.threshold || '—' }}</td>
              <td>
                <span :class="['status-pill', rule.enabled ? 'enabled' : 'disabled']">
                  {{ rule.enabled ? 'Yes' : 'No' }}
                </span>
              </td>
              <td v-if="isAdmin">
                <button @click="deleteRule(rule.id)" class="btn-delete">Delete</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Alert History Section -->
    <div class="section history-section">
      <h2>Alert History</h2>

      <!-- History Table -->
      <div class="history-table">
        <table>
          <thead>
            <tr>
              <th>Time</th>
              <th>Job</th>
              <th>Rule Type</th>
              <th>Channel</th>
              <th>Status</th>
              <th v-if="isAdmin">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="history.length === 0">
              <td colspan="6" class="empty-state">No alerts have fired</td>
            </tr>
            <tr v-for="alert in history" :key="alert.id">
              <td>{{ formatTime(alert.fired_at) }}</td>
              <td>{{ alert.job_id }}</td>
              <td>{{ formatRuleType(alert.rule_type) }}</td>
              <td>{{ alert.channel }}</td>
              <td>
                <span :class="['status-pill', 'status-' + alert.status]">
                  {{ formatStatus(alert.status) }}
                </span>
              </td>
              <td v-if="isAdmin">
                <button @click="retryAlert(alert.id)" class="btn-retry">Retry</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script>
import { getAlertRules, createAlertRule, deleteAlertRule, getAlertHistory, retryAlert } from '../api'

export default {
  name: 'Alerts',
  data() {
    return {
      rules: [],
      history: [],
      newRule: {
        jobId: '',
        ruleType: '',
        threshold: 0,
        enabled: true
      },
      isAdmin: false,
      refreshInterval: null
    }
  },
  computed: {
    currentRole() {
      return this.$store?.state?.user?.role || 'viewer'
    }
  },
  watch: {
    currentRole(newRole) {
      this.isAdmin = newRole === 'admin'
    }
  },
  methods: {
    loadRules() {
      getAlertRules()
        .then(rules => {
          this.rules = rules || []
        })
        .catch(err => console.error('Failed to load rules:', err))
    },
    loadHistory() {
      getAlertHistory()
        .then(history => {
          this.history = history || []
        })
        .catch(err => console.error('Failed to load history:', err))
    },
    createRule() {
      if (!this.newRule.ruleType) return

      const rule = {
        job_id: this.newRule.jobId || null,
        rule_type: this.newRule.ruleType,
        threshold: this.newRule.threshold,
        enabled: this.newRule.enabled
      }

      createAlertRule(rule)
        .then(() => {
          this.loadRules()
          this.newRule = { jobId: '', ruleType: '', threshold: 0, enabled: true }
        })
        .catch(err => console.error('Failed to create rule:', err))
    },
    deleteRule(id) {
      if (!confirm('Delete this alert rule?')) return

      deleteAlertRule(id)
        .then(() => this.loadRules())
        .catch(err => console.error('Failed to delete rule:', err))
    },
    retryAlert(id) {
      retryAlert(id)
        .then(() => {
          this.loadHistory()
        })
        .catch(err => console.error('Failed to retry alert:', err))
    },
    formatRuleType(type) {
      const map = {
        'on_failure': 'On Failure',
        'duration_exceeded': 'Duration Exceeded',
        'missed_schedule': 'Missed Schedule'
      }
      return map[type] || type
    },
    formatStatus(status) {
      const map = {
        'delivered': 'Delivered',
        'retrying': 'Retrying',
        'failed': 'Failed'
      }
      return map[status] || status
    },
    formatTime(datetime) {
      if (!datetime) return '—'
      const d = new Date(datetime)
      return d.toLocaleString()
    }
  },
  mounted() {
    this.isAdmin = this.currentRole === 'admin'
    this.loadRules()
    this.loadHistory()

    // Auto-refresh history every 30 seconds
    this.refreshInterval = setInterval(() => {
      this.loadHistory()
    }, 30000)
  },
  beforeUnmount() {
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval)
    }
  }
}
</script>

<style scoped>
.alerts-container {
  max-width: 1200px;
  margin: 0 auto;
}

h1 {
  font-family: var(--font-display);
  font-size: 1.3rem;
  font-weight: 700;
  margin-bottom: 1.5rem;
  color: var(--text-primary);
}

h2 {
  font-family: var(--font-display);
  font-size: 1rem;
  font-weight: 700;
  margin-bottom: 1.1rem;
  color: var(--text-primary);
}

.section {
  background: var(--bg-card);
  border-radius: 8px;
  padding: 1.25rem;
  margin-bottom: 1.5rem;
  border: 1px solid var(--border-default);
}

.create-rule-form {
  display: flex;
  gap: 0.65rem;
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
  padding: 1rem;
  background: var(--bg-elevated);
  border-radius: 6px;
  border: 1px solid var(--border-subtle);
}

.form-group {
  flex: 1;
  min-width: 140px;
}

.form-group input,
.form-group select {
  width: 100%;
  padding: 0.45rem 0.7rem;
  border: 1px solid var(--border-default);
  border-radius: 5px;
  font-family: var(--font-body);
  font-size: 0.88rem;
  background: var(--bg-input);
  color: var(--text-primary);
  transition: border-color 0.15s;
}

.form-group input:focus,
.form-group select:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-dim);
}

.form-group.checkbox {
  display: flex;
  align-items: flex-end;
  flex: none;
  padding-bottom: 0.1rem;
}

.form-group.checkbox label {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  cursor: pointer;
  color: var(--text-secondary);
  font-family: var(--font-body);
  font-size: 0.85rem;
}

/* Scoped button overrides for Alerts (avoid clobbering global btn) */
.create-rule-form > button,
.rules-table button,
.history-table button {
  padding: 0.45rem 1rem;
  background: var(--accent);
  color: var(--bg-base);
  border: none;
  border-radius: 5px;
  cursor: pointer;
  font-family: var(--font-body);
  font-size: 0.85rem;
  font-weight: 600;
  transition: filter 0.15s;
}
.create-rule-form > button:hover:not(:disabled),
.rules-table button:hover:not(:disabled),
.history-table button:hover:not(:disabled) { filter: brightness(1.1); }

.create-rule-form > button:disabled,
.rules-table button:disabled,
.history-table button:disabled { opacity: 0.45; cursor: not-allowed; }

.btn-delete {
  background: var(--bg-error) !important;
  color: var(--color-error) !important;
  border: 1px solid rgba(255,92,122,0.3) !important;
  padding: 0.25rem 0.75rem !important;
  font-size: 0.78rem !important;
}
.btn-delete:hover:not(:disabled) { background: var(--color-error) !important; color: var(--bg-base) !important; }

.btn-retry {
  background: var(--bg-warning) !important;
  color: var(--color-warning) !important;
  border: 1px solid rgba(245,158,11,0.3) !important;
  padding: 0.25rem 0.75rem !important;
  font-size: 0.78rem !important;
}
.btn-retry:hover:not(:disabled) { filter: brightness(1.1); }

table {
  width: 100%;
  border-collapse: collapse;
  font-family: var(--font-body);
}

thead th {
  padding: 0.55rem 0.75rem;
  text-align: left;
  font-size: 0.72rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border-default);
}

tbody td {
  padding: 0.65rem 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
  color: var(--text-primary);
  font-size: 0.88rem;
}

tbody tr:last-child td { border-bottom: none; }
tbody tr:hover td { background: var(--bg-elevated); }

.empty-state {
  text-align: center;
  padding: 1.75rem 0.75rem;
  color: var(--text-muted);
  font-family: var(--font-body);
  font-size: 0.85rem;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  padding: 0.2rem 0.65rem;
  border-radius: 4px;
  font-family: var(--font-body);
  font-size: 0.72rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.status-pill.enabled       { background: var(--bg-success); color: var(--color-success); }
.status-pill.disabled      { background: var(--bg-elevated); color: var(--text-muted); }
.status-pill.status-delivered { background: var(--bg-success); color: var(--color-success); }
.status-pill.status-retrying  { background: var(--bg-warning); color: var(--color-warning); }
.status-pill.status-failed    { background: var(--bg-error);   color: var(--color-error);   }

@media (max-width: 768px) {
  .create-rule-form { flex-direction: column; }
  .form-group { min-width: auto; }
  table { font-size: 0.82rem; }
  thead th, tbody td { padding: 0.45rem; }
}
</style>
