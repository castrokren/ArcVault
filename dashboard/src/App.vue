<template>
  <div class="app">
    <header class="nav">
      <span class="nav-brand">ArcVault</span>
      <nav>
        <router-link to="/agents">Agents</router-link>
        <router-link to="/jobs">Jobs</router-link>
        <router-link to="/history">History</router-link>
        <router-link to="/templates">Templates</router-link>
        <router-link to="/alerts">Alerts</router-link>
        <router-link
          to="/users"
          :class="{ disabled: !isAdmin }"
          :title="!isAdmin ? 'Requires admin role' : ''"
        >Users</router-link>
        <router-link
          to="/groups"
          :class="{ disabled: !isAdmin }"
          :title="!isAdmin ? 'Requires admin role' : ''"
        >Groups</router-link>
      </nav>
      <div class="nav-right">
        <button
          v-if="rollbackAvailable"
          class="btn-rollback-header"
          title="Roll back coordinator to previous version"
          @click="showRollbackModal = true"
        >
          ↩ Rollback
        </button>
        <button class="theme-toggle" @click="toggleTheme" :title="`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`">
          <span v-if="theme === 'dark'">☀️</span>
          <span v-else>🌙</span>
        </button>
        <span class="ws-indicator" :class="{ connected: wsConnected }">
          {{ wsConnected ? '● Live' : '○ Disconnected' }}
        </span>
        <div v-if="auth.isAuthenticated.value" class="user-menu">
          <span class="user-info">{{ auth.currentUser?.username }} ({{ auth.currentUser?.role }})</span>
          <button class="btn-user-action" @click="showChangePasswordModal = true" title="Change password">
            🔐
          </button>
          <button class="btn-user-action" @click="handleLogout" title="Logout">
            🚪
          </button>
        </div>
      </div>
    </header>

    <UpdateBanner v-if="updateStore.available" :onUpdate="showUpdateModal" />

    <main>
      <router-view :lastEvent="lastEvent" />
    </main>

    <UpdateModal :isOpen="updateModalOpen" :lastEvent="lastEvent" @close="updateModalOpen = false" />

    <ChangePasswordModal
      :isOpen="showChangePasswordModal"
      @close="showChangePasswordModal = false"
      @success="showChangePasswordModal = false"
    />

    <RollbackModal
      v-if="showRollbackModal"
      target="coordinator"
      @close="showRollbackModal = false"
      @complete="onCoordinatorRollbackComplete"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, provide, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { getRollbackAvailable } from './api.js'
import { useAuth } from './composables/useAuth.js'
import { useWebSocket } from './composables/useWebSocket.js'
import UpdateBanner from './components/UpdateBanner.vue'
import UpdateModal from './components/UpdateModal.vue'
import RollbackModal from './components/RollbackModal.vue'
import ChangePasswordModal from './components/ChangePasswordModal.vue'

const router = useRouter()
const auth = useAuth()
const updateModalOpen = ref(false)
const showRollbackModal = ref(false)
const showChangePasswordModal = ref(false)
const rollbackAvailable = ref(false)
const theme = ref(localStorage.getItem('arcvault-theme') || 'dark')
const { connected: wsConnected, lastEvent, connect } = useWebSocket()

const isAdmin = computed(() => auth.hasRole('admin'))

// Reactive update info store
const updateStore = reactive({
  current: 'v0.2.0',
  latest: 'v0.2.0',
  available: false,
  releaseUrl: '',
  assetUrl: ''
})

// Provide updateStore to child components
provide('updateStore', updateStore)

onMounted(() => {
  applyTheme(theme.value)
  if (auth.isAuthenticated.value) {
    connect()
    checkForUpdates()
    checkRollbackAvailable()
  }
})

function applyTheme(val) {
  document.documentElement.setAttribute('data-theme', val)
  localStorage.setItem('arcvault-theme', val)
}

function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
  applyTheme(theme.value)
}

function checkForUpdates() {
  const token = auth.getToken()
  if (!token) return

  fetch('/api/update/check', {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    }
  })
    .then(r => r.json())
    .then(data => {
      updateStore.current = data.current
      updateStore.latest = data.latest
      updateStore.available = data.update_available
      updateStore.releaseUrl = data.release_url
      updateStore.assetUrl = data.asset_url
    })
    .catch(err => {
      console.error('Failed to check for updates:', err)
    })
}

async function checkRollbackAvailable() {
  try {
    const data = await getRollbackAvailable()
    rollbackAvailable.value = data.available === true
  } catch {
    rollbackAvailable.value = false
  }
}

function showUpdateModal() {
  updateModalOpen.value = true
}

function onCoordinatorRollbackComplete() {
  showRollbackModal.value = false
  rollbackAvailable.value = false
}

async function handleLogout() {
  await auth.logout()
  router.push('/login')
}
</script>

<style scoped>
.app { display: flex; flex-direction: column; min-height: 100vh; }

.nav {
  display: flex;
  align-items: center;
  gap: 2rem;
  padding: 0.75rem 1.5rem;
  background: #1a1a2e;
  color: #fff;
}

.nav-brand { font-weight: 700; font-size: 1.2rem; letter-spacing: 0.05em; }

.nav a {
  color: #aaa;
  text-decoration: none;
  font-size: 0.95rem;
  transition: color 0.2s;
}
.nav a.router-link-active { color: #fff; border-bottom: 2px solid #4f8ef7; padding-bottom: 2px; }
.nav a.disabled { opacity: 0.5; cursor: not-allowed; pointer-events: none; }

.nav-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 1rem;
}

.btn-rollback-header {
  padding: 0.3rem 0.8rem;
  background: transparent;
  color: #e6a817;
  border: 1px solid #e6a817;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.82rem;
  font-weight: 600;
  transition: background 0.15s;
  white-space: nowrap;
}
.btn-rollback-header:hover { background: rgba(230, 168, 23, 0.15); }

.theme-toggle {
  background: none;
  border: none;
  color: inherit;
  cursor: pointer;
  font-size: 1.2rem;
  padding: 0;
  display: flex;
  align-items: center;
}

.theme-toggle:hover { opacity: 0.7; }

.ws-indicator { font-size: 0.85rem; color: #e55; }
.ws-indicator.connected { color: #4caf50; }

.user-menu {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.user-info {
  font-size: 0.9rem;
  color: #aaa;
}

.btn-user-action {
  background: none;
  border: none;
  color: inherit;
  cursor: pointer;
  font-size: 1.1rem;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  transition: background 0.2s, opacity 0.2s;
}

.btn-user-action:hover {
  background: rgba(255, 255, 255, 0.1);
}

main { padding: 1.5rem; flex: 1; }

[data-theme="light"] {
  --bg-primary: #ffffff;
  --bg-secondary: #f5f5f5;
  --text-primary: #1a1a1a;
  --text-secondary: #666666;
  --border-color: #e0e0e0;
  --card-bg: #fafafa;
}

[data-theme="light"] .app { background: #fff; color: #1a1a1a; }

[data-theme="light"] .nav {
  background: #f5f5f5;
  color: #1a1a1a;
  border-bottom: 1px solid #e0e0e0;
}

[data-theme="light"] .nav a { color: #666; }

[data-theme="light"] .nav a.router-link-active { color: #1a1a1a; }

[data-theme="light"] .user-info { color: #666; }

[data-theme="light"] .btn-user-action:hover { background: rgba(0, 0, 0, 0.05); }

[data-theme="light"] .ws-indicator { color: #e74c3c; }

[data-theme="light"] .ws-indicator.connected { color: #27ae60; }
</style>
