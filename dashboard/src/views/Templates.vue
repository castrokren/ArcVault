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
.page-header { display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1.5rem; }
.page-header h1 { margin: 0; flex: 1; }
.page-header button {
  padding: 0.4rem 1rem;
  background: #4f8ef7;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.error { color: #e55; margin: 0.5rem 0; }
.empty { color: #888; margin: 2rem 0; }

.filters { margin-bottom: 1.5rem; }
.search-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border-radius: 4px;
  border: 1px solid #2a2a3e;
  background: #0f0f1a;
  color: #fff;
  font-size: 0.95rem;
}
.search-input:focus {
  outline: none;
  border-color: #4f8ef7;
  box-shadow: 0 0 0 2px rgba(79, 142, 247, 0.1);
}

.table { width: 100%; border-collapse: collapse; }
.table th, .table td {
  text-align: left;
  padding: 0.6rem 0.75rem;
  border-bottom: 1px solid #2a2a3e;
}
.table th { color: #888; font-weight: 600; font-size: 0.85rem; text-transform: uppercase; }
.mono { font-family: monospace; font-size: 0.85rem; }
.dim { color: #888; font-size: 0.85rem; }

.cron-hint {
  display: block;
  color: #888;
  font-size: 0.75rem;
  margin-top: 0.1rem;
}

.actions { display: flex; gap: 0.4rem; align-items: center; }

.action-btn {
  padding: 0.2rem 0.6rem;
  background: #1a2a3a;
  color: #4f8ef7;
  border: 1px solid #4f8ef7;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.8rem;
}
.action-btn:hover { background: #4f8ef7; color: #fff; }

.run-btn {
  background: #1a3a1a;
  color: #4caf50;
  border-color: #4caf50;
}
.run-btn:hover { background: #4caf50; color: #fff; }

.toggle-btn {
  padding: 0.2rem 0.7rem;
  border-radius: 999px;
  border: 1px solid;
  font-size: 0.8rem;
  cursor: pointer;
  font-weight: 600;
}
.toggle-btn.enabled  { background: #1a3a1a; color: #4caf50; border-color: #4caf50; }
.toggle-btn.disabled { background: #2a2a2a; color: #888;    border-color: #555; }

button.danger-sm {
  padding: 0.2rem 0.6rem;
  background: #3a1a1a;
  color: #e55;
  border: 1px solid #e55;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.8rem;
}

/* Modal */
.modal-backdrop {
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
  padding: 2rem;
  width: 520px;
  max-width: 95vw;
}
.modal h3 { margin: 0 0 1.25rem; }

.form-grid {
  display: grid;
  grid-template-columns: 140px 1fr;
  gap: 0.6rem 1rem;
  align-items: start;
  margin-bottom: 1rem;
}
.form-grid label { color: #aaa; font-size: 0.9rem; padding-top: 0.45rem; }
.form-grid input {
  padding: 0.4rem 0.6rem;
  border-radius: 4px;
  border: 1px solid #444;
  background: #111;
  color: #fff;
  width: 100%;
  box-sizing: border-box;
}
.form-grid input:disabled, .form-grid input.dimmed {
  opacity: 0.5;
  cursor: not-allowed;
}

.schedule-field { display: flex; flex-direction: column; gap: 0.25rem; }
.cron-preview { color: #4f8ef7; font-size: 0.8rem; }

.switch-label { display: flex; align-items: center; gap: 0.5rem; color: #ccc; font-size: 0.9rem; cursor: pointer; }

.form-actions { display: flex; gap: 0.5rem; margin-top: 1rem; }
button.primary {
  padding: 0.4rem 1.2rem;
  background: #4caf50;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}
.form-actions button:not(.primary) {
  padding: 0.4rem 1rem;
  background: transparent;
  color: #aaa;
  border: 1px solid #444;
  border-radius: 4px;
  cursor: pointer;
}
</style>
