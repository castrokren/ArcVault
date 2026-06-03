<template>
  <div class="federation-view">
    <div class="view-header">
      <h1>Federation</h1>
      <div class="header-actions">
        <router-link to="/federation/health" class="btn-secondary">Health Status</router-link>
        <button class="btn-primary" @click="openCreate">+ Add Site</button>
      </div>
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
} from '../api'

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
/* modal-overlay, form-group covered globally */

.federation-view { max-width: 1000px; }

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
  color: var(--text-primary);
}

.header-actions { display: flex; gap: 0.6rem; align-items: center; }

.loading, .empty {
  color: var(--text-muted);
  font-family: var(--font-body);
  padding: 2rem 0;
}

/* data-table reuses .table styles, but has its own class name */
.data-table {
  width: 100%;
  border-collapse: collapse;
  font-family: var(--font-body);
  font-size: 0.88rem;
}
.data-table th {
  text-align: left;
  padding: 0.55rem 0.85rem;
  font-size: 0.72rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border-default);
}
.data-table td {
  padding: 0.65rem 0.85rem;
  border-bottom: 1px solid var(--border-subtle);
  color: var(--text-primary);
}
.data-table tbody tr:last-child td { border-bottom: none; }
.data-table tbody tr:hover td { background: var(--bg-elevated); }

.url-cell {
  font-family: var(--font-mono);
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  padding: 0.2rem 0.6rem;
  border-radius: 4px;
  font-size: 0.72rem;
  font-weight: 600;
  text-transform: capitalize;
}
.status-badge.online   { background: var(--bg-success);  color: var(--color-success); }
.status-badge.offline  { background: var(--bg-error);    color: var(--color-error); }
.status-badge.degraded { background: var(--bg-warning);  color: var(--color-warning); }

.actions-cell { display: flex; gap: 0.4rem; }

.btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
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
.btn-primary:hover:not(:disabled) { filter: brightness(1.1); }
.btn-primary:disabled { opacity: 0.45; cursor: default; }

.btn-secondary {
  display: inline-flex;
  align-items: center;
  padding: 0.4rem 1rem;
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
  border-radius: 5px;
  cursor: pointer;
  font-family: var(--font-body);
  font-size: 0.85rem;
  text-decoration: none;
  transition: all 0.15s;
}
.btn-secondary:hover { color: var(--text-primary); border-color: var(--border-strong); background: var(--bg-elevated); }

.btn-sm {
  padding: 0.22rem 0.6rem;
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
  border-radius: 4px;
  cursor: pointer;
  font-family: var(--font-body);
  font-size: 0.78rem;
  transition: all 0.15s;
}
.btn-sm:hover { color: var(--text-primary); border-color: var(--border-strong); background: var(--bg-elevated); }
.btn-sm:disabled { opacity: 0.45; cursor: default; }

.btn-danger {
  padding: 0.22rem 0.6rem;
  background: transparent;
  color: var(--color-error);
  border: 1px solid rgba(255,77,109,0.4);
  border-radius: 4px;
  cursor: pointer;
  font-family: var(--font-body);
  font-size: 0.78rem;
  transition: background 0.15s;
}
.btn-danger:hover { background: var(--bg-error); }
.btn-danger:disabled { opacity: 0.45; cursor: default; }

/* Modal (uses global .modal-overlay) */
.modal {
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: 10px;
  padding: 1.5rem;
  min-width: 360px;
  max-width: 92vw;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  box-shadow: var(--shadow-lg);
}

.modal h2 {
  margin: 0;
  font-family: var(--font-display);
  font-size: 1rem;
  font-weight: 700;
  color: var(--text-primary);
}

.modal p {
  margin: 0;
  color: var(--text-secondary);
  font-family: var(--font-body);
  font-size: 0.88rem;
}

.error {
  color: var(--color-error);
  font-family: var(--font-body);
  font-size: 0.85rem;
}

.modal-actions { display: flex; gap: 0.65rem; }
</style>
