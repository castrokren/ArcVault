<template>
  <div class="app">
    <header class="nav">
      <span class="nav-brand">ArcVault</span>
      <nav>
        <router-link to="/agents">Agents</router-link>
        <router-link to="/jobs">Jobs</router-link>
        <router-link to="/history">History</router-link>
      </nav>
      <div class="nav-right">
        <span class="ws-indicator" :class="{ connected: wsConnected }">
          {{ wsConnected ? '● Live' : '○ Disconnected' }}
        </span>
      </div>
    </header>

    <UpdateBanner v-if="tokenSet" :onUpdate="showUpdateModal" />

    <div v-if="!tokenSet" class="token-gate">
      <div class="token-box">
        <h2>Enter Admin Token</h2>
        <input v-model="tokenInput" type="password" placeholder="Bearer token" @keyup.enter="saveToken" />
        <button @click="saveToken">Connect</button>
      </div>
    </div>

    <main v-else>
      <router-view :lastEvent="lastEvent" />
    </main>

    <UpdateModal :isOpen="updateModalOpen" :lastEvent="lastEvent" @close="updateModalOpen = false" />
  </div>
</template>

<script setup>
import { ref, onMounted, provide, reactive } from 'vue'
import { saveToken as persistToken, hasToken } from './api.js'
import { useWebSocket } from './composables/useWebSocket.js'
import UpdateBanner from './components/UpdateBanner.vue'
import UpdateModal from './components/UpdateModal.vue'

const tokenInput = ref('')
const tokenSet = ref(false)
const updateModalOpen = ref(false)
const { connected: wsConnected, lastEvent, connect } = useWebSocket()

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
  if (hasToken()) {
    tokenSet.value = true
    connect()
    checkForUpdates()
  }
})

function saveToken() {
  if (!tokenInput.value.trim()) return
  persistToken(tokenInput.value.trim())
  tokenSet.value = true
  connect()
  checkForUpdates()
}

function checkForUpdates() {
  const token = localStorage.getItem('arcvault_token')
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

function showUpdateModal() {
  updateModalOpen.value = true
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
}
.nav a.router-link-active { color: #fff; border-bottom: 2px solid #4f8ef7; padding-bottom: 2px; }

.nav-right { margin-left: auto; }

.ws-indicator { font-size: 0.85rem; color: #e55; }
.ws-indicator.connected { color: #4caf50; }

.token-gate {
  display: flex;
  justify-content: center;
  align-items: center;
  flex: 1;
  padding: 2rem;
}

.token-box {
  background: #1e1e2e;
  border: 1px solid #333;
  border-radius: 8px;
  padding: 2rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  min-width: 320px;
}

.token-box h2 { margin: 0; color: #fff; }

.token-box input {
  padding: 0.5rem 0.75rem;
  border-radius: 4px;
  border: 1px solid #444;
  background: #111;
  color: #fff;
  font-size: 1rem;
}

.token-box button {
  padding: 0.5rem;
  background: #4f8ef7;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 1rem;
}

main { padding: 1.5rem; flex: 1; }
</style>
