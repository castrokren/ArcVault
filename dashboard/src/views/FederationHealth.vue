<template>
  <div class="federation-health">
    <div class="view-header">
      <h1>Federation Health</h1>
    </div>

    <div v-if="loading" class="skeleton-group" aria-busy="true">
      <div class="skeleton skeleton-line" style="width: 38%"></div>
      <div class="skeleton skeleton-block"></div>
      <div class="skeleton skeleton-line" style="width: 62%"></div>
    </div>

    <div v-else-if="error" class="error-state">
      <p>{{ error }}</p>
      <button @click="loadHealth">Retry</button>
    </div>

    <div v-else-if="healthList.length === 0" class="empty-state">
      <p>No federation peers registered.</p>
      <router-link to="/federation">Configure federation</router-link>
    </div>

    <div v-else class="health-table">
      <table>
        <thead>
          <tr>
            <th>Coordinator ID</th>
            <th>Status</th>
            <th>Last Seen</th>
            <th>Event Lag</th>
            <th>Agent Count</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="coordinator in healthList" :key="coordinator.id" :class="`status-${coordinator.status}`">
            <td class="coordinator-id">{{ coordinator.id }}</td>
            <td class="status-cell">
              <span class="status-pill" :class="`status-${coordinator.status}`">
                {{ coordinator.status }}
              </span>
            </td>
            <td class="last-seen">
              {{ formatTime(coordinator.last_seen) }}
            </td>
            <td class="lag-cell" :class="{ stale: coordinator.lag_events > 50, warning: coordinator.lag_events > 0 }">
              {{ coordinator.lag_events }} event{{ coordinator.lag_events !== 1 ? 's' : '' }}
            </td>
            <td class="agent-count">{{ coordinator.agent_count }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="refresh-note">
      Auto-refreshing every 15 seconds
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue';
import { getFederationHealth } from '../api';

const loading = ref(true);
const error = ref(null);
const healthList = ref([]);
let refreshInterval = null;

const loadHealth = async () => {
  try {
    loading.value = true;
    error.value = null;
    const response = await getFederationHealth();
    healthList.value = response || [];
  } catch (err) {
    error.value = `Failed to load federation health: ${err.message}`;
    healthList.value = [];
  } finally {
    loading.value = false;
  }
};

const formatTime = (timestamp) => {
  if (!timestamp) return '—';
  const date = new Date(timestamp);
  const now = new Date();
  const diff = Math.floor((now - date) / 1000); // seconds

  if (diff < 60) return 'now';
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return date.toLocaleDateString();
};

onMounted(() => {
  loadHealth();
  refreshInterval = setInterval(loadHealth, 15000);
});

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval);
});
</script>

<style scoped>
.federation-health {
  max-width: 1000px;
}

.view-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-subtle);
}

.view-header h1 {
  margin: 0;
  font-family: var(--font-display);
  font-size: 1.3rem;
  font-weight: 700;
  color: var(--text-primary);
}

.loading,
.error-state,
.empty-state {
  padding: 3rem;
  text-align: center;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  color: var(--text-muted);
  font-family: var(--font-body);
  font-size: 0.88rem;
}

.error-state p,
.empty-state p {
  margin: 0 0 1rem;
}

.error-state button,
.empty-state a {
  display: inline-flex;
  align-items: center;
  padding: 0.4rem 1rem;
  background: var(--accent);
  color: var(--bg-base);
  border: none;
  border-radius: 5px;
  cursor: pointer;
  font-family: var(--font-body);
  font-size: 0.85rem;
  font-weight: 600;
  text-decoration: none;
  transition: filter 0.15s;
}

.error-state button:hover,
.empty-state a:hover {
  filter: brightness(1.1);
}

.health-table {
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  overflow: hidden;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-family: var(--font-body);
  font-size: 0.88rem;
}

thead tr {
  background: var(--bg-elevated);
  border-bottom: 1px solid var(--border-default);
}

th {
  padding: 0.55rem 1rem;
  text-align: left;
  font-size: 0.72rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-muted);
}

tbody tr {
  border-bottom: 1px solid var(--border-subtle);
}

tbody tr:hover td {
  background: var(--bg-elevated);
}

tbody tr:last-child {
  border-bottom: none;
}

td {
  padding: 0.65rem 1rem;
  color: var(--text-primary);
}

.coordinator-id {
  font-family: var(--font-mono);
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.status-cell {
  padding: 0.5rem 1rem;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  padding: 0.2rem 0.6rem;
  border-radius: 4px;
  font-size: 0.72rem;
  font-weight: 600;
  text-transform: capitalize;
}

.status-pill.status-online {
  background: var(--bg-success);
  color: var(--color-success);
}

.status-pill.status-offline {
  background: var(--bg-error);
  color: var(--color-error);
}

.status-pill.status-reconnecting {
  background: var(--bg-warning);
  color: var(--color-warning);
}

.last-seen {
  color: var(--text-secondary);
}

.lag-cell {
  font-weight: 500;
  color: var(--text-primary);
}

.lag-cell.warning {
  color: var(--color-warning);
}

.lag-cell.stale {
  color: var(--color-error);
  font-weight: 700;
}

.agent-count {
  color: var(--text-secondary);
  text-align: center;
}

.refresh-note {
  margin-top: 1rem;
  text-align: right;
  font-family: var(--font-body);
  font-size: 0.75rem;
  color: var(--text-muted);
}
</style>
