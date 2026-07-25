<template>
  <div>
    <div class="page-header">
      <h1>Templates</h1>
      <button @click="openCreate">+ New Template</button>
      <button @click="load" :disabled="loading">{{ loading ? 'Loading...' : 'Refresh' }}</button>
    </div>

    <div v-if="error" class="error">{{ error }}</div>

    <div class="filters">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="Search templates by name or agent ID..."
        class="search-input"
        @input="onSearch"
      />
    </div>

    <div v-if="result.total === 0 && !loading" class="empty">
      {{ searchQuery ? 'No templates match your search.' : 'No templates yet. Create one to get started.' }}
    </div>

    <table v-else class="table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Agent</th>
          <th>Schedule</th>
          <th>Enabled</th>
          <th>Next Run</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="t in result.data" :key="t.id">
          <td>{{ t.name }}</td>
          <td class="mono">{{ t.agent_id }}</td>
          <td>
            <span class="mono">{{ t.schedule }}</span>
            <span v-if="preview(t.schedule)" class="cron-hint">{{ preview(t.schedule) }}</span>
          </td>
          <td>
            <button
              class="toggle-btn"
              :class="t.enabled ? 'enabled' : 'disabled'"
              @click="toggleEnabled(t)"
            >
              {{ t.enabled ? 'On' : 'Off' }}
            </button>
          </td>
          <td class="dim">{{ t.next_run ? formatDate(t.next_run) : '—' }}</td>
          <td class="actions">
            <button class="action-btn" @click="openEdit(t)">Edit</button>
            <button class="action-btn run-btn" @click="runNow(t)" title="Run immediately">▶ Run</button>
            <button class="danger-sm" @click="remove(t.id)">Delete</button>
          </td>
        </tr>
      </tbody>
    </table>

    <Pagination
      :page="page"
      :pages="result.pages"
      :total="result.total"
      :limit="limit"
      @page-change="goToPage"
    />

    <!-- Create / Edit modal -->
    <div v-if="showModal" class="modal-backdrop" @click.self="closeModal">
      <div class="modal">
        <h3>{{ editTarget ? 'Edit Template' : 'New Template' }}</h3>

        <div class="form-grid">
          <label>ID</label>
          <input
            v-model="form.id"
            placeholder="nightly-docs-backup"
            :disabled="!!editTarget"
            :class="{ dimmed: !!editTarget }"
          />

          <label>Name</label>
          <input v-model="form.name" placeholder="Nightly Docs Backup" />

          <label>Agent ID</label>
          <input v-model="form.agent_id" placeholder="agent-01" />

          <label>Command</label>
          <input v-model="form.command" placeholder='robocopy D:\Docs E:\Backup /MIR' />

          <label>Schedule</label>
          <ScheduleBuilder v-model="form.schedule" />

          <template v-if="editTarget">
            <label>Enabled</label>
            <label class="switch-label">
              <input type="checkbox" v-model="form.enabled" />
              <span>{{ form.enabled ? 'Yes' : 'No' }}</span>
            </label>
          </template>
        </div>

        <div v-if="modalError" class="error">{{ modalError }}</div>

        <div class="form-actions">
          <button class="primary" @click="saveTemplate" :disabled="saving">
            {{ saving ? 'Saving...' : (editTarget ? 'Save Changes' : 'Create') }}
          </button>
          <button @click="closeModal">Cancel</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getTemplates, createTemplate, updateTemplate, deleteTemplate, runTemplate, cronPreview } from '../api'
import Pagination from '../components/Pagination.vue'
import ScheduleBuilder from '../components/ScheduleBuilder.vue'

const result = ref({ data: [], total: 0, page: 1, pages: 0, limit: 25 })
const page = ref(1)
const limit = 25
const loading = ref(false)
const error = ref(null)
const searchQuery = ref('')

const showModal = ref(false)
const editTarget = ref(null)
const saving = ref(false)
const modalError = ref(null)

const form = ref({ id: '', name: '', agent_id: '', command: '', schedule: '', enabled: true })

function preview(expr) {
  return cronPreview(expr)
}

async function load() {
  loading.value = true
  error.value = null
  try {
    result.value = await getTemplates({ page: page.value, limit, search: searchQuery.value })
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function onSearch() {
  page.value = 1
  load()
}

function goToPage(n) {
  page.value = n
  load()
}

function openCreate() {
  editTarget.value = null
  form.value = { id: '', name: '', agent_id: '', command: '', schedule: '', enabled: true }
  modalError.value = null
  showModal.value = true
}

function openEdit(t) {
  editTarget.value = t
  form.value = { id: t.id, name: t.name, agent_id: t.agent_id, command: t.command, schedule: t.schedule, enabled: t.enabled }
  modalError.value = null
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  editTarget.value = null
}

async function saveTemplate() {
  modalError.value = null
  saving.value = true
  try {
    if (editTarget.value) {
      await updateTemplate(editTarget.value.id, {
        name: form.value.name,
        agent_id: form.value.agent_id,
        command: form.value.command,
        schedule: form.value.schedule,
        enabled: form.value.enabled,
      })
    } else {
      await createTemplate({
        id: form.value.id,
        name: form.value.name,
        agent_id: form.value.agent_id,
        command: form.value.command,
        schedule: form.value.schedule,
      })
    }
    closeModal()
    page.value = 1
    await load()
  } catch (e) {
    modalError.value = e.message
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(t) {
  try {
    await updateTemplate(t.id, { enabled: !t.enabled })
    await load()
  } catch (e) {
    error.value = e.message
  }
}

async function runNow(t) {
  try {
    await runTemplate(t.id)
    // brief visual feedback then reload to show new job in History
    await load()
  } catch (e) {
    error.value = e.message
  }
}

async function remove(id) {
  if (!confirm('Delete this template?')) return
  try {
    await deleteTemplate(id)
    await load()
  } catch (e) {
    error.value = e.message
  }
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleString()
}

onMounted(load)
</script>

<style scoped>
/* All flat #4f8ef7 / #2a2a3e / #0f0f1a / #1e1e2e / #4caf50 colors
   replaced with Royal Purple design-system tokens (theme-royal-purple.css).
   Layout rules (flex, grid, padding, etc.) are unchanged. */

/* .page-header, .filters, .empty, .error, .table handled by global style.css
   — scoped overrides below only for Templates-specific elements */

.search-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border-radius: var(--radius-ctrl);
  border: 1px solid var(--border-default);
  background: var(--bg-input);
  color: var(--text-primary);
  font-size: 0.95rem;
  font-family: var(--font-body);
  outline: none;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.search-input:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px var(--accent-dim);
}

.table { width: 100%; border-collapse: collapse; }
.table th, .table td {
  text-align: left;
  padding: 0.6rem 0.75rem;
  border-bottom: 1px solid var(--border-default);
}
.table th {
  color: var(--text-muted);
  font-weight: 600;
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}
.table tbody tr:hover td { background: var(--bg-elevated); }
.mono  { font-family: var(--font-mono); font-size: 0.85rem; }
.dim   { color: var(--text-muted); font-size: 0.85rem; }
.cron-hint { display: block; color: var(--text-muted); font-size: 0.75rem; margin-top: 0.1rem; }

.actions { display: flex; gap: 0.4rem; align-items: center; }

.action-btn {
  padding: 0.2rem 0.6rem;
  background: var(--accent-dim);
  color: var(--accent);
  border: 1px solid var(--accent-border);
  border-radius: 5px;
  cursor: pointer;
  font-size: 0.8rem;
  font-family: var(--font-body);
  transition: background 0.12s, color 0.12s;
}
.action-btn:hover { background: var(--accent); color: var(--bg-base); }

.run-btn {
  background: var(--bg-success);
  color: var(--color-success);
  border-color: var(--color-success);
}
.run-btn:hover { background: var(--color-success); color: var(--bg-base); }

.toggle-btn {
  padding: 0.2rem 0.7rem;
  border-radius: 999px;
  border: 1px solid;
  font-size: 0.8rem;
  cursor: pointer;
  font-weight: 600;
  font-family: var(--font-body);
}
.toggle-btn.enabled  { background: var(--bg-success);  color: var(--color-success); border-color: var(--color-success); }
.toggle-btn.disabled { background: var(--bg-elevated);  color: var(--text-muted);   border-color: var(--border-default); }

button.danger-sm {
  padding: 0.2rem 0.6rem;
  background: var(--bg-error);
  color: var(--color-error);
  border: 1px solid rgba(255, 92, 122, 0.35);
  border-radius: 5px;
  cursor: pointer;
  font-size: 0.8rem;
  font-family: var(--font-body);
}

/* Modal */
.modal-backdrop {
  position: fixed; inset: 0;
  background: rgba(0, 0, 0, 0.68);
  backdrop-filter: blur(4px);
  display: flex; align-items: center; justify-content: center;
  z-index: 100;
}
.modal {
  background: var(--bg-elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-card);
  padding: 2rem;
  width: 520px;
  max-width: 95vw;
  box-shadow: var(--shadow-lg);
}
.modal h3 { margin: 0 0 1.25rem; color: var(--text-primary); font-family: var(--font-display); }

.form-grid {
  display: grid;
  grid-template-columns: 140px 1fr;
  gap: 0.6rem 1rem;
  align-items: start;
  margin-bottom: 1rem;
}
.form-grid label { color: var(--text-muted); font-size: 0.9rem; padding-top: 0.45rem; font-family: var(--font-body); }
.form-grid input {
  padding: 0.4rem 0.6rem;
  border-radius: 6px;
  border: 1px solid var(--border-default);
  background: var(--bg-input);
  color: var(--text-primary);
  font-family: var(--font-body);
  font-size: 0.88rem;
  width: 100%;
  box-sizing: border-box;
  outline: none;
  transition: border-color 0.15s;
}
.form-grid input:focus { border-color: var(--accent); box-shadow: 0 0 0 2px var(--accent-dim); }
.form-grid input:disabled, .form-grid input.dimmed { opacity: 0.45; cursor: not-allowed; }

.schedule-field { display: flex; flex-direction: column; gap: 0.25rem; }
.cron-preview { color: var(--accent); font-size: 0.8rem; font-family: var(--font-mono); }

.switch-label { display: flex; align-items: center; gap: 0.5rem; color: var(--text-secondary); font-size: 0.9rem; cursor: pointer; font-family: var(--font-body); }

.form-actions { display: flex; gap: 0.5rem; margin-top: 1rem; }
button.primary {
  padding: 0.4rem 1.2rem;
  background: var(--accent);
  color: var(--bg-base);
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-family: var(--font-body);
  font-weight: 600;
}
button.primary:hover { filter: brightness(1.1); }
.form-actions button:not(.primary) {
  padding: 0.4rem 1rem;
  background: transparent;
  color: var(--text-muted);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  cursor: pointer;
  font-family: var(--font-body);
}
.form-actions button:not(.primary):hover { color: var(--text-primary); border-color: var(--border-strong); }
</style>