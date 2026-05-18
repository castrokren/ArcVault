<template>
  <div class="federation-view">
    <div class="view-header">
      <h1>Federation</h1>
      <button class="btn-primary" @click="openCreate">+ Add Site</button>
    </div>

    <div v-if="loading" class="loading">Loading...</div>

    <div v-else-if="subs.length === 0" class="empty">
      No sub-coordinators registered. Add one to get started.
    </div>

    <table v-else class="data-table">
      <thead>
        <tr>
          <th>Name</th>
          <th>URL</th>
          <th>Status</th>
          <th>Last Seen</th>
          <th>Version</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="sub in subs" :key="sub.id">
          <td>{{ sub.name }}</td>
          <td class="url-cell">{{ sub.url }}</td>
          <td>
            <span class="status-badge" :class="sub.status">{{ sub.status }}</span>
          </td>
          <td>{{ sub.last_seen ? fmtDate(sub.last_seen) : '—' }}</td>
          <td>{{ sub.version || '—' }}</td>
          <td class="actions-cell">
            <button class="btn-sm" @click="openEdit(sub)">Edit</button>
            <button class="btn-sm" @click="forceSync(sub)" :disabled="syncing === sub.id">
              {{ syncing === sub.id ? 'Syncing…' : 'Sync' }}
            </button>
            <button class="btn-sm btn-danger" @click="confirmDelete(sub)">Delete</button>
          </td>
        </tr>
      </tbody>
    </table>

    <!-- Create / Edit Modal -->
    <div v-if="modal.open" class="modal-overlay" @click.self="closeModal">
      <div class="modal">
        <h2>{{ modal.editing ? 'Edit Site' : 'Add Site' }}</h2>
        <div class="form-group">
          <label>Name</label>
          <input v-model="modal.name" placeholder="NYC Office" />
        </div>
        <div class="form-group">
          <label>URL</label>
          <input v-model="modal.url" placeholder="http://sub.internal:8080" />
        </div>
        <div class="form-group">
          <label>Token</label>
          <input v-model="modal.token" type="password" placeholder="Federation token" />
        </div>
        <div v-if="modal.error" class="error">{{ modal.error }}</div>
        <div class="modal-actions">
          <button class="btn-primary" @click="submitModal" :disabled="modal.saving">
            {{ modal.saving ? 'Saving…' : 'Save' }}
          </button>
          <button class="btn-secondary" @click="closeModal">Cancel</button>
        </div>
      </div>
    </div>

    <!-- Delete Confirm Modal -->
    <div v-if="deleteTarget" class="modal-overlay" @click.self="deleteTarget = null">
      <div class="modal">
        <h2>Delete {{ deleteTarget.name }}?</h2>
        <p>This will remove the sub-coordinator and close its connection. This cannot be undone.</p>
        <div class="modal-actions">
          <button class="btn-danger" @click="doDelete" :disabled="deleting">
            {{ deleting ? 'Deleting…' : 'Delete' }}
          </button>
          <button class="btn-secondary" @click="deleteTarget = null">Cancel</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import {
  listFederation,
  createFederation,
  updateFederation,
  deleteFederation,
  syncFederation,
} from '../api.js'

const subs = ref([])
const loading = ref(true)
const syncing = ref(null)
const deleteTarget = ref(null)
const deleting = ref(false)

const modal = ref({
  open: false,
  editing: null,
  name: '',
  url: '',
  token: '',
  error: '',
  saving: false,
})

async function load() {
  try {
    subs.value = await listFederation() || []
  } catch {
    subs.value = []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  modal.value = { open: true, editing: null, name: '', url: '', token: '', error: '', saving: false }
}

function openEdit(sub) {
  modal.value = { open: true, editing: sub, name: sub.name, url: sub.url, token: '', error: '', saving: false }
}

function closeModal() {
  modal.value.open = false
}

async function submitModal() {
  const m = modal.value
  if (!m.name.trim()) { m.error = 'Name is required'; return }
  if (!m.url.trim()) { m.error = 'URL is required'; return }
  if (!m.editing && !m.token.trim()) { m.error = 'Token is required'; return }

  m.saving = true
  m.error = ''
  try {
    if (m.editing) {
      const data = { name: m.name, url: m.url }
      if (m.token.trim()) data.token = m.token.trim()
      await updateFederation(m.editing.id, data)
    } else {
      await createFederation({ name: m.name.trim(), url: m.url.trim(), token: m.token.trim() })
    }
    closeModal()
    await load()
  } catch (e) {
    m.error = e.message || 'Failed to save'
  } finally {
    m.saving = false
  }
}

async function forceSync(sub) {
  syncing.value = sub.id
  try {
    await syncFederation(sub.id)
    setTimeout(load, 1500)
  } catch {
    // ignore
  } finally {
    syncing.value = null
  }
}

function confirmDelete(sub) {
  deleteTarget.value = sub
}

async function doDelete() {
  deleting.value = true
  try {
    await deleteFederation(deleteTarget.value.id)
    deleteTarget.value = null
    await load()
  } catch {
    // ignore
  } finally {
    deleting.value = false
  }
}

function fmtDate(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  return d.toLocaleString()
}

onMounted(load)
</script>

<style scoped>
.federation-view { max-width: 1000px; }

.view-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.5rem;
}

.view-header h1 { margin: 0; font-size: 1.4rem; }

.loading, .empty { color: var(--text-secondary, #888); padding: 2rem 0; }

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9rem;
}

.data-table th {
  text-align: left;
  padding: 0.6rem 0.75rem;
  border-bottom: 2px solid var(--border-color, #333);
  color: var(--text-secondary, #888);
  font-weight: 600;
}

.data-table td {
  padding: 0.6rem 0.75rem;
  border-bottom: 1px solid var(--border-color, #2a2a3a);
}

.url-cell {
  font-size: 0.82rem;
  color: var(--text-secondary, #888);
  font-family: monospace;
}

.status-badge {
  display: inline-block;
  padding: 0.2rem 0.55rem;
  border-radius: 99px;
  font-size: 0.78rem;
  font-weight: 600;
  text-transform: capitalize;
}

.status-badge.online  { background: rgba(76, 175, 80, 0.15); color: #4caf50; }
.status-badge.offline { background: rgba(229, 57, 53, 0.15);  color: #e53935; }
.status-badge.degraded { background: rgba(255, 167, 38, 0.15); color: #ffa726; }

.actions-cell { display: flex; gap: 0.4rem; }

.btn-primary {
  padding: 0.4rem 1rem;
  background: #4f8ef7;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.88rem;
}
.btn-primary:hover { background: #3a7de0; }
.btn-primary:disabled { opacity: 0.5; cursor: default; }

.btn-secondary {
  padding: 0.4rem 1rem;
  background: transparent;
  color: #aaa;
  border: 1px solid #555;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.88rem;
}
.btn-secondary:hover { background: rgba(255,255,255,0.05); }

.btn-sm {
  padding: 0.25rem 0.6rem;
  background: transparent;
  color: #aaa;
  border: 1px solid #444;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.8rem;
}
.btn-sm:hover { background: rgba(255,255,255,0.05); }
.btn-sm:disabled { opacity: 0.5; cursor: default; }

.btn-danger {
  padding: 0.25rem 0.6rem;
  background: transparent;
  color: #e53935;
  border: 1px solid #e53935;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.8rem;
}
.btn-danger:hover { background: rgba(229, 57, 53, 0.1); }
.btn-danger:disabled { opacity: 0.5; cursor: default; }

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal {
  background: #1e1e2e;
  border: 1px solid #333;
  border-radius: 8px;
  padding: 1.5rem;
  min-width: 360px;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.modal h2 { margin: 0; font-size: 1.1rem; }
.modal p { margin: 0; color: #aaa; font-size: 0.9rem; }

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.form-group label { font-size: 0.82rem; color: #aaa; }

.form-group input {
  padding: 0.45rem 0.65rem;
  border-radius: 4px;
  border: 1px solid #444;
  background: #111;
  color: #fff;
  font-size: 0.9rem;
}

.form-group input:focus { outline: none; border-color: #4f8ef7; }

.error { color: #e53935; font-size: 0.85rem; }

.modal-actions { display: flex; gap: 0.75rem; }

[data-theme="light"] .modal { background: #fafafa; border-color: #e0e0e0; }
[data-theme="light"] .form-group input { background: #fff; border-color: #ddd; color: #1a1a1a; }
[data-theme="light"] .btn-sm { color: #555; border-color: #ccc; }
[data-theme="light"] .btn-secondary { color: #555; border-color: #ccc; }
</style>
