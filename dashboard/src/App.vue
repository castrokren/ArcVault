<template>
  <div class="app">
    <header v-if="auth.isAuthenticated.value" class="nav">
      <div class="nav-brand">
        <svg class="nav-logo" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M10 2L17 6V10C17 13.87 13.94 17.5 10 18.5C6.06 17.5 3 13.87 3 10V6L10 2Z"
            stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/>
          <path d="M7 10L9.5 12.5L13 8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        <span class="nav-brand-text">ArcVault</span>
        <span class="nav-version">{{ updateStore.current }}</span>
      </div>

      <div class="nav-divider"></div>

      <nav class="nav-links">
        <router-link to="/agents">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><rect x="1.5" y="1.5" width="4" height="4" rx="1" stroke="currentColor" stroke-width="1.3"/><rect x="8.5" y="1.5" width="4" height="4" rx="1" stroke="currentColor" stroke-width="1.3"/><rect x="1.5" y="8.5" width="4" height="4" rx="1" stroke="currentColor" stroke-width="1.3"/><rect x="8.5" y="8.5" width="4" height="4" rx="1" stroke="currentColor" stroke-width="1.3"/></svg>
          Agents
        </router-link>
        <router-link to="/jobs">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M2 4h10M2 7h6M2 10h4" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/></svg>
          Jobs
        </router-link>
        <router-link to="/admin/credentials">
          <svg width="13" height="13" viewBox="0 0 14 14" fill="none"><rect x="2" y="6" width="10" height="6" rx="1" stroke="currentColor" stroke-width="1.3"/><path d="M4 6V4.5a3 3 0 0 1 6 0V6" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/><circle cx="7" cy="9" r="0.75" fill="currentColor"/></svg>
          Credentials
        </router-link>
        <router-link to="/history">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><circle cx="7" cy="7" r="5" stroke="currentColor" stroke-width="1.3"/><path d="M7 4.5V7l2 1.5" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/></svg>
          History
        </router-link>
        <router-link to="/templates">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><rect x="1.5" y="1.5" width="11" height="11" rx="1.5" stroke="currentColor" stroke-width="1.3"/><path d="M1.5 5.5h11" stroke="currentColor" stroke-width="1.3"/><path d="M5.5 5.5v7" stroke="currentColor" stroke-width="1.3"/></svg>
          Templates
        </router-link>
        <router-link to="/alerts">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M7 1.5L1.5 10.5h11L7 1.5z" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"/><path d="M7 6v2.5" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/><circle cx="7" cy="10" r="0.6" fill="currentColor"/></svg>
          Alerts
        </router-link>
        <div v-if="isAdmin" class="nav-admin-menu" @mouseenter="adminMenuOpen = true" @mouseleave="adminMenuOpen = false">
          <span class="nav-admin-trigger">
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><circle cx="7" cy="4" r="2" stroke="currentColor" stroke-width="1.3"/><path d="M2 12c0-2.76 2.24-5 5-5s5 2.24 5 5" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/><circle cx="11" cy="3" r="1.5" stroke="currentColor" stroke-width="1.2"/></svg>
            Admin
            <svg width="10" height="10" viewBox="0 0 10 10" fill="none"><path d="M2.5 4l2.5 2.5L7.5 4" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/></svg>
          </span>
          <div v-if="adminMenuOpen" class="nav-admin-dropdown">
            <router-link to="/users" @click="adminMenuOpen = false">
              <svg width="13" height="13" viewBox="0 0 14 14" fill="none"><circle cx="7" cy="5" r="2.5" stroke="currentColor" stroke-width="1.3"/><path d="M2 12c0-2.76 2.24-5 5-5s5 2.24 5 5" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/></svg>
              Users
            </router-link>
            <router-link to="/groups" @click="adminMenuOpen = false">
              <svg width="13" height="13" viewBox="0 0 14 14" fill="none"><circle cx="5" cy="5" r="2" stroke="currentColor" stroke-width="1.3"/><circle cx="9.5" cy="4.5" r="1.75" stroke="currentColor" stroke-width="1.3"/><path d="M1 12c0-2.21 1.79-4 4-4s4 1.79 4 4" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/><path d="M9.5 8c1.65 0 3 1.35 3 3" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/></svg>
              Groups
            </router-link>
          </div>
        </div>
      </nav>

      <div class="nav-right">
        <button class="theme-toggle" @click="toggleTheme" :title="`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`">
          <!-- Sun -->
          <svg v-if="theme === 'dark'" width="15" height="15" viewBox="0 0 15 15" fill="none"><circle cx="7.5" cy="7.5" r="2.5" stroke="currentColor" stroke-width="1.3"/><path d="M7.5 1v1.5M7.5 12.5V14M14 7.5h-1.5M2.5 7.5H1M12.07 2.93l-1.06 1.06M4 11l-1.06 1.06M12.07 12.07l-1.06-1.06M4 4 2.93 2.93" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/></svg>
          <!-- Moon -->
          <svg v-else width="15" height="15" viewBox="0 0 15 15" fill="none"><path d="M12 9a5 5 0 1 1-6-6 3.5 3.5 0 0 0 6 6z" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"/></svg>
        </button>

        <div class="ws-indicator" :class="{ connected: wsConnected }" :title="wsConnected ? 'Live â€” WebSocket connected' : 'Disconnected'">
          <span class="ws-dot"></span>
          <span class="ws-label">{{ wsConnected ? 'Live' : 'Off' }}</span>
        </div>

        <div v-if="auth.isAuthenticated.value" class="user-menu">
          <div v-if="userInitials" class="user-avatar">{{ userInitials }}</div>
          <span class="user-info">
            <span class="user-name">{{ auth.currentUser?.username }}</span>
            <span class="user-role">{{ auth.currentUser?.role }}</span>
          </span>
          <div class="user-actions">
            <button class="icon-btn" @click="showChangePasswordModal = true" title="Change password">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><rect x="3" y="6" width="8" height="6" rx="1" stroke="currentColor" stroke-width="1.3"/><path d="M5 6V4.5a2 2 0 0 1 4 0V6" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/><circle cx="7" cy="9" r="0.75" fill="currentColor"/></svg>
            </button>
            <button class="icon-btn icon-btn--logout" @click="handleLogout" title="Logout">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M9 9.5l2.5-2.5L9 4.5" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/><path d="M11.5 7H5" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/><path d="M5 2H2.5A1 1 0 0 0 1.5 3v8a1 1 0 0 0 1 1H5" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/></svg>
            </button>
          </div>
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


  </div>
</template>

<script setup>
import { ref, onMounted, provide, reactive, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { checkUpdate } from './api'
import { useAuth } from './composables/useAuth.js'
import { useWebSocket } from './composables/useWebSocket.js'
import UpdateBanner from './components/UpdateBanner.vue'
import UpdateModal from './components/UpdateModal.vue'
import ChangePasswordModal from './components/ChangePasswordModal.vue'

const router = useRouter()
const auth = useAuth()
const updateModalOpen = ref(false)
const showChangePasswordModal = ref(false)
const adminMenuOpen = ref(false)
const theme = ref(localStorage.getItem('arcvault-theme') || 'dark')
const { connected: wsConnected, lastEvent, connect } = useWebSocket()

const isAdmin = computed(() => auth.hasRole('admin'))

const userInitials = computed(() => {
  const name = auth.currentUser?.username || ''
  return name.slice(0, 2).toUpperCase()
})

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
    console.log('App mounted: connecting websocket and checking coordinator updates')
    connect()
    checkForUpdates()
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
  console.log('Checking coordinator update status via /api/update/check')
  checkUpdate()
    .then(data => {
      console.log('Update check response:', data)
      updateStore.current = data.current
      updateStore.latest = data.latest
      updateStore.available = data.update_available
      updateStore.releaseUrl = data.release_url
      updateStore.assetUrl = data.asset_url
      console.log('Update availability:', {
        current: updateStore.current,
        latest: updateStore.latest,
        available: updateStore.available,
        releaseUrl: updateStore.releaseUrl,
        assetUrl: updateStore.assetUrl
      })
    })
    .catch(err => {
      console.error('Failed to check for updates:', err)
    })
}

function showUpdateModal() {
  updateModalOpen.value = true
}

async function handleLogout() {
  await auth.logout()
  router.push('/login')
}
</script>

<style scoped>
/* â”€â”€ App shell â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€ */
.app {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: var(--bg-base);
}

/* â”€â”€ Nav â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€ */
.nav {
  display: flex;
  align-items: center;
  gap: 0;
  height: var(--nav-height);
  padding: 0 1.25rem;
  background: var(--nav-bg);
  border-bottom: 1px solid var(--nav-border);
  position: sticky;
  top: 0;
  z-index: 100;
  backdrop-filter: blur(12px);
}

/* Brand */
.nav-brand {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  text-decoration: none;
  flex-shrink: 0;
}

.nav-logo {
  width: 20px;
  height: 20px;
  color: var(--accent);
}

.nav-brand-text {
  font-family: var(--font-display);
  font-weight: 800;
  font-size: 1rem;
  letter-spacing: 0.04em;
  color: var(--text-primary);
}

.nav-version {
  font-family: var(--font-body);
  font-size: 0.7rem;
  font-weight: 500;
  color: var(--text-muted);
  opacity: 0.7;
  margin-left: 0.2rem;
  align-self: flex-end;
  padding-bottom: 0.05rem;
}

.nav-divider {
  width: 1px;
  height: 20px;
  background: var(--border-default);
  margin: 0 1.1rem;
  flex-shrink: 0;
}

/* Nav links */
.nav-links {
  display: flex;
  align-items: center;
  gap: 0.1rem;
  flex: 1;
}

.nav-links a {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.35rem 0.65rem;
  border-radius: 5px;
  color: var(--text-secondary);
  text-decoration: none;
  font-family: var(--font-body);
  font-size: 0.85rem;
  font-weight: 500;
  transition: color 0.15s, background 0.15s;
  white-space: nowrap;
}

.nav-links a svg {
  flex-shrink: 0;
  opacity: 0.7;
  transition: opacity 0.15s;
}

.nav-links a:hover {
  color: var(--text-primary);
  background: var(--bg-elevated);
}

.nav-links a:hover svg {
  opacity: 1;
}

.nav-links a.router-link-active {
  color: var(--text-primary);
  background: var(--bg-elevated);
  font-weight: 600;
}

.nav-links a.router-link-active svg {
  opacity: 1;
  color: var(--accent);
}

.nav-admin-menu {
  position: relative;
}

.nav-admin-trigger {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.35rem 0.65rem;
  border-radius: 5px;
  color: var(--text-secondary);
  font-family: var(--font-body);
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: color 0.15s, background 0.15s;
}

.nav-admin-menu:hover .nav-admin-trigger {
  color: var(--text-primary);
  background: var(--bg-elevated);
}

.nav-admin-trigger svg {
  flex-shrink: 0;
  opacity: 0.7;
}

.nav-admin-dropdown {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: 7px;
  padding: 0.3rem;
  min-width: 130px;
  box-shadow: 0 4px 16px rgba(0,0,0,0.25);
  z-index: 100;
}

.nav-admin-dropdown a {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.4rem 0.6rem;
  border-radius: 5px;
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 0.85rem;
  font-weight: 500;
  transition: color 0.15s, background 0.15s;
  white-space: nowrap;
}

.nav-admin-dropdown a:hover,
.nav-admin-dropdown a.router-link-active {
  color: var(--text-primary);
  background: var(--bg-elevated);
}

.nav-links a.disabled {
  opacity: 0.35;
  cursor: not-allowed;
  pointer-events: none;
}

/* Right side */
.nav-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 0.4rem;
  flex-shrink: 0;
}

/* Theme toggle */
.theme-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  background: none;
  border: 1px solid transparent;
  border-radius: 5px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: color 0.15s, background 0.15s, border-color 0.15s;
}

.theme-toggle:hover {
  color: var(--text-primary);
  background: var(--bg-elevated);
  border-color: var(--border-default);
}

/* WS indicator */
.ws-indicator {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0.6rem;
  border-radius: 999px;
  border: 1px solid var(--border-default);
  cursor: default;
  user-select: none;
}

.ws-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-error);
  flex-shrink: 0;
}

.ws-indicator.connected .ws-dot {
  background: var(--color-success);
  animation: ws-pulse 2s ease-in-out infinite;
}

@keyframes ws-pulse {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 var(--color-success); }
  50%       { opacity: 0.7; box-shadow: 0 0 0 3px transparent; }
}

.ws-label {
  font-family: var(--font-body);
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--text-secondary);
}

.ws-indicator.connected .ws-label {
  color: var(--color-success);
}

/* User menu */
.user-menu {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.2rem 0.2rem 0.2rem 0.5rem;
  border: 1px solid var(--border-default);
  border-radius: 6px;
  background: var(--bg-card);
}

.user-avatar {
  width: 24px;
  height: 24px;
  border-radius: 4px;
  background: var(--accent-dim);
  border: 1px solid var(--accent-border);
  color: var(--accent);
  font-family: var(--font-display);
  font-size: 0.65rem;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  letter-spacing: 0.02em;
}

.user-info {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
}

.user-name {
  font-family: var(--font-body);
  font-size: 0.8rem;
  font-weight: 500;
  color: var(--text-primary);
}

.user-role {
  font-family: var(--font-body);
  font-size: 0.7rem;
  color: var(--text-muted);
  text-transform: capitalize;
}

.user-actions {
  display: flex;
  align-items: center;
  gap: 0.1rem;
  margin-left: 0.2rem;
}

.icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  background: none;
  border: none;
  border-radius: 4px;
  color: var(--text-muted);
  cursor: pointer;
  transition: color 0.15s, background 0.15s;
}

.icon-btn:hover {
  color: var(--text-primary);
  background: var(--bg-elevated);
}

.icon-btn--logout:hover {
  color: var(--color-error);
  background: var(--bg-error);
}

/* Main content */
main {
  padding: 1.5rem 1.75rem;
  flex: 1;
}
</style>