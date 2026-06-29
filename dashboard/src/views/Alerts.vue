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
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

h1 {
  font-size: 2rem;
  font-weight: 600;
  margin-bottom: 30px;
  color: var(--text-primary, #1a1a1a);
}

h2 {
  font-size: 1.3rem;
  font-weight: 600;
  margin-bottom: 20px;
  color: var(--text-primary, #1a1a1a);
}

.section {
  background: var(--bg-secondary, #f5f5f5);
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 30px;
  border: 1px solid var(--border-color, #e0e0e0);
}

.create-rule-form {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
  flex-wrap: wrap;
  padding: 15px;
  background: var(--bg-primary, #fff);
  border-radius: 6px;
}

.form-group {
  flex: 1;
  min-width: 150px;
}

.form-group input,
.form-group select {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border-color, #e0e0e0);
  border-radius: 4px;
  font-size: 0.95rem;
  background: var(--bg-primary, #fff);
  color: var(--text-primary, #1a1a1a);
}

.form-group.checkbox {
  display: flex;
  align-items: center;
  flex: none;
}

.form-group.checkbox label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

button {
  padding: 8px 16px;
  background: var(--accent-color, #0066cc);
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 600;
  transition: background 0.2s;
}

button:hover:not(:disabled) {
  background: var(--accent-dark, #0052a3);
}

button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-delete, .btn-retry {
  padding: 6px 12px;
  font-size: 0.85rem;
  background: var(--error-color, #d32f2f);
}

.btn-retry {
  background: var(--warning-color, #f57c00);
}

.btn-delete:hover, .btn-retry:hover {
  opacity: 0.9;
}

table {
  width: 100%;
  border-collapse: collapse;
  background: var(--bg-primary, #fff);
}

thead th {
  padding: 12px;
  text-align: left;
  background: var(--bg-tertiary, #f0f0f0);
  font-weight: 600;
  border-bottom: 2px solid var(--border-color, #e0e0e0);
  color: var(--text-primary, #1a1a1a);
}

tbody td {
  padding: 12px;
  border-bottom: 1px solid var(--border-color, #e0e0e0);
  color: var(--text-primary, #1a1a1a);
}

tbody tr:hover {
  background: var(--bg-tertiary, #f9f9f9);
}

.empty-state {
  text-align: center;
  padding: 30px 12px;
  color: var(--text-secondary, #999);
  font-style: italic;
}

.status-pill {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 0.85rem;
  font-weight: 600;
  white-space: nowrap;
}

.status-pill.enabled {
  background: rgba(76, 175, 80, 0.1);
  color: #2e7d32;
}

.status-pill.disabled {
  background: rgba(158, 158, 158, 0.1);
  color: #616161;
}

.status-pill.status-delivered {
  background: rgba(76, 175, 80, 0.1);
  color: #2e7d32;
}

.status-pill.status-retrying {
  background: rgba(255, 193, 7, 0.1);
  color: #f57f17;
}

.status-pill.status-failed {
  background: rgba(244, 67, 54, 0.1);
  color: #c62828;
}

@media (max-width: 768px) {
  .create-rule-form {
    flex-direction: column;
  }

  .form-group {
    min-width: auto;
  }

  table {
    font-size: 0.9rem;
  }

  thead th, tbody td {
    padding: 8px;
  }
}
</style>
