<template>
  <div class="federation-health">
    <h1>Federation Health</h1>

    <div v-if="loading" class="loading">
      <p>Loading federation status...</p>
    </div>

    <div v-else-if="error" class="error-state">
      <p>{{ error }}</p>
      <button @click="loadHealth">Retry</button>
    </div>

    <div v-else-if="healthList.length === 0" class="empty-state">
      <p>No federation peers registered</p>
      <a href="/federation">Configure federation</a>
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
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

h1 {
  font-size: 1.75rem;
  margin-bottom: 24px;
  color: var(--text-primary);
  font-weight: 600;
}

.loading,
.error-state,
.empty-state {
  padding: 40px;
  text-align: center;
  background: var(--bg-secondary);
  border-radius: 8px;
  color: var(--text-secondary);
}

.error-state button,
.empty-state a {
  margin-top: 16px;
  padding: 8px 16px;
  background: var(--accent);
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  text-decoration: none;
  display: inline-block;
}

.error-state button:hover,
.empty-state a:hover {
  opacity: 0.9;
}

.health-table {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9rem;
}

thead tr {
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
}

th {
  padding: 12px 16px;
  text-align: left;
  font-weight: 600;
  color: var(--text-secondary);
  font-size: 0.85rem;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

tbody tr {
  border-bottom: 1px solid var(--border-color);
}

tbody tr:hover {
  background: var(--bg-secondary);
}

tbody tr:last-child {
  border-bottom: none;
}

td {
  padding: 12px 16px;
  color: var(--text-primary);
}

.coordinator-id {
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.status-cell {
  padding: 8px 16px;
}

.status-pill {
  display: inline-block;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: capitalize;
}

.status-online {
  background: oklch(70% 0.15 135);
  color: oklch(25% 0.05 135);
}

.status-offline {
  background: oklch(60% 0.08 0);
  color: oklch(30% 0.04 0);
}

.status-reconnecting {
  background: oklch(75% 0.12 50);
  color: oklch(25% 0.05 50);
}

.last-seen {
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.lag-cell {
  font-weight: 500;
}

.lag-cell.warning {
  color: oklch(65% 0.15 50);
}

.lag-cell.stale {
  color: oklch(55% 0.2 20);
  font-weight: 600;
}

.agent-count {
  color: var(--text-secondary);
  text-align: center;
}

.refresh-note {
  margin-top: 16px;
  text-align: right;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

@media (max-width: 768px) {
  .federation-health {
    padding: 12px;
  }

  table {
    font-size: 0.85rem;
  }

  th,
  td {
    padding: 8px 12px;
  }

  h1 {
    font-size: 1.5rem;
  }
}
</style>
