<template>
  <div class="users-container">
    <div class="users-header">
      <h1>User Management</h1>
      <div class="users-header-actions">
        <button class="btn btn-secondary" @click="handleDownloadInstaller" :disabled="downloadingInstaller" title="Download ArcVault Windows setup installer">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" style="margin-right:4px;vertical-align:-2px"><path d="M7 1v8m0 0l-3-3m3 3l3-3" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/><rect x="1" y="11" width="12" height="2" rx="1" stroke="currentColor" stroke-width="1.3"/></svg>
          Download Agent Installer
        </button>
        <button class="btn btn-primary" @click="showCreateUserModal = true">
          + Create User
        </button>
      </div>
    </div>

    <div v-if="loading" class="skeleton-group" aria-busy="true">
      <div class="skeleton skeleton-line" style="width: 38%"></div>
      <div class="skeleton skeleton-block"></div>
      <div class="skeleton skeleton-line" style="width: 62%"></div>
    </div>

    <div v-else-if="error" class="error-message">
      {{ error }}
      <button @click="fetchUsers" class="btn btn-secondary btn-small">Retry</button>
    </div>

    <div v-else class="users-table-wrapper">
      <table class="users-table">
        <thead>
          <tr>
            <th>Username</th>
            <th>Role</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in users" :key="user.id">
            <td>{{ user.username }}</td>
            <td>
              <span class="role-badge" :class="user.role">{{ user.role }}</span>
            </td>
            <td>
              <button
                class="btn btn-secondary btn-small"
                @click="editUser(user)"
                title="Edit role"
              >
                Edit
              </button>
              <button
                class="btn btn-danger btn-small"
                @click="deleteUser(user)"
                title="Delete user"
              >
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="users.length === 0" class="empty-state">
        No users found
      </div>

      <div class="pagination">
        <button
          :disabled="page === 1"
          @click="page--"
          class="btn btn-secondary btn-small"
        >
          ← Prev
        </button>
        <span class="page-info">Page {{ page }}</span>
        <button
          :disabled="users.length < limit"
          @click="page++"
          class="btn btn-secondary btn-small"
        >
          Next →
        </button>
      </div>
    </div>

    <!-- Create User Modal -->
    <div v-if="showCreateUserModal" class="modal-overlay" @click.self="showCreateUserModal = false">
      <div class="modal-dialog">
        <div class="modal-header">
          <h2>Create User</h2>
          <button class="modal-close" @click="showCreateUserModal = false">×</button>
        </div>
        <form @submit.prevent="handleCreateUser" class="modal-body">
          <div class="form-group">
            <label for="new-username">Username</label>
            <input
              id="new-username"
              v-model="newUser.username"
              type="text"
              placeholder="Enter username"
              :disabled="creatingUser"
              autofocus
            />
          </div>

          <div class="form-group">
            <label for="new-password">Password</label>
            <input
              id="new-password"
              v-model="newUser.password"
              type="password"
              placeholder="Enter password (min 8 characters)"
              :disabled="creatingUser"
            />
          </div>

          <div class="form-group">
            <label for="new-role">Role</label>
            <select
              id="new-role"
              v-model="newUser.role"
              :disabled="creatingUser"
            >
              <option value="viewer">Viewer</option>
              <option value="operator">Operator</option>
              <option value="admin">Admin</option>
            </select>
          </div>

          <div v-if="createError" class="error-message">
            {{ createError }}
          </div>

          <div class="modal-footer">
            <button
              type="button"
              class="btn btn-secondary"
              @click="showCreateUserModal = false"
              :disabled="creatingUser"
            >
              Cancel
            </button>
            <button
              type="submit"
              class="btn btn-primary"
              :disabled="creatingUser || !newUser.username || !newUser.password"
            >
              {{ creatingUser ? 'Creating...' : 'Create User' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Edit Role Modal -->
    <div v-if="showEditRoleModal" class="modal-overlay" @click.self="showEditRoleModal = false">
      <div class="modal-dialog">
        <div class="modal-header">
          <h2>Edit User Role</h2>
          <button class="modal-close" @click="showEditRoleModal = false">×</button>
        </div>
        <form @submit.prevent="handleEditRole" class="modal-body">
          <div class="form-group">
            <label>Username</label>
            <div class="readonly-field">{{ editingUser?.username }}</div>
          </div>

          <div class="form-group">
            <label for="edit-role">Role</label>
            <select
              id="edit-role"
              v-model="editingUser.role"
              :disabled="editingRole"
            >
              <option value="viewer">Viewer</option>
              <option value="operator">Operator</option>
              <option value="admin">Admin</option>
            </select>
          </div>

          <div v-if="editError" class="error-message">
            {{ editError }}
          </div>

          <div class="modal-footer">
            <button
              type="button"
              class="btn btn-secondary"
              @click="showEditRoleModal = false"
              :disabled="editingRole"
            >
              Cancel
            </button>
            <button
              type="submit"
              class="btn btn-primary"
              :disabled="editingRole"
            >
              {{ editingRole ? 'Updating...' : 'Update Role' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete Confirmation Modal -->
    <div v-if="showDeleteConfirmModal" class="modal-overlay" @click.self="showDeleteConfirmModal = false">
      <div class="modal-dialog">
        <div class="modal-header">
          <h2>Delete User</h2>
          <button class="modal-close" @click="showDeleteConfirmModal = false">×</button>
        </div>
        <div class="modal-body">
          <p>Are you sure you want to delete user <strong>{{ deletingUser?.username }}</strong>?</p>
          <p class="warning">This action cannot be undone.</p>

          <div class="modal-footer">
            <button
              type="button"
              class="btn btn-secondary"
              @click="showDeleteConfirmModal = false"
              :disabled="deletingUser.inProgress"
            >
              Cancel
            </button>
            <button
              type="button"
              class="btn btn-danger"
              @click="confirmDelete"
              :disabled="deletingUser.inProgress"
            >
              {{ deletingUser.inProgress ? 'Deleting...' : 'Delete User' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { getUsers, createUser, updateUserRole, deleteUser as deleteUserApi, downloadInstaller } from '../api'
import { useAuth } from '../composables/useAuth.js'

const auth = useAuth()

// Main state
const users = ref([])
const loading = ref(false)
const error = ref('')
const page = ref(1)
const limit = 25

// Create user modal
const showCreateUserModal = ref(false)
const creatingUser = ref(false)
const createError = ref('')
const newUser = ref({ username: '', password: '', role: 'viewer' })

// Edit role modal
const showEditRoleModal = ref(false)
const editingRole = ref(false)
const editError = ref('')
const editingUser = ref({ id: '', username: '', role: 'viewer' })

// Delete confirmation modal
const showDeleteConfirmModal = ref(false)
const deletingUser = ref({ id: '', username: '', inProgress: false })

onMounted(() => {
  // Redirect if not admin
  if (!auth.hasRole('admin')) {
    window.location.hash = '/agents'
    return
  }
  fetchUsers()
})

async function fetchUsers() {
  loading.value = true
  error.value = ''

  try {
    const data = await getUsers({ page: page.value, limit })
    users.value = data.data || []
  } catch (err) {
    error.value = err.message || 'Failed to load users'
  } finally {
    loading.value = false
  }
}

async function handleCreateUser() {
  if (!newUser.value.username || !newUser.value.password) {
    createError.value = 'Username and password are required'
    return
  }

  if (newUser.value.password.length < 8) {
    createError.value = 'Password must be at least 8 characters'
    return
  }

  creatingUser.value = true
  createError.value = ''

  try {
    await createUser(newUser.value.username, newUser.value.password, newUser.value.role)
    showCreateUserModal.value = false
    newUser.value = { username: '', password: '', role: 'viewer' }
    await fetchUsers()
  } catch (err) {
    createError.value = err.message || 'Failed to create user'
  } finally {
    creatingUser.value = false
  }
}

function editUser(user) {
  editingUser.value = { ...user }
  showEditRoleModal.value = true
  editError.value = ''
}

async function handleEditRole() {
  editingRole.value = true
  editError.value = ''

  try {
    await updateUserRole(editingUser.value.id, editingUser.value.role)
    showEditRoleModal.value = false
    await fetchUsers()
  } catch (err) {
    editError.value = err.message || 'Failed to update user role'
    editingRole.value = false
  }
}

function deleteUser(user) {
  deletingUser.value = { id: user.id, username: user.username, inProgress: false }
  showDeleteConfirmModal.value = true
}

async function confirmDelete() {
  deletingUser.value.inProgress = true

  try {
    await deleteUserApi(deletingUser.value.id)
    showDeleteConfirmModal.value = false
    await fetchUsers()
  } catch (err) {
    error.value = err.message || 'Failed to delete user'
    showDeleteConfirmModal.value = false
  }
}

// Re-fetch when page changes
watch(page, () => fetchUsers())

const downloadingInstaller = ref(false)

async function handleDownloadInstaller() {
  downloadingInstaller.value = true
  try {
    await downloadInstaller()
  } catch (err) {
    alert(`Failed to download installer: ${err.message}`)
  } finally {
    downloadingInstaller.value = false
  }
}
</script>

<style scoped>
/* modal-overlay, modal-dialog, modal-header, modal-body, modal-footer,
   form-group, readonly-field all covered by global design system */

.users-container {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.users-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-subtle);
}

.users-header h1 {
  margin: 0;
  font-family: var(--font-display);
  font-size: 1.3rem;
  color: var(--text-primary);
}

.users-header-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.loading,
.empty-state {
  padding: 2rem;
  text-align: center;
  color: var(--text-muted);
  font-family: var(--font-body);
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
}

.error-message {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.6rem 0.9rem;
  background: var(--bg-error);
  border: 1px solid rgba(255,92,122,0.3);
  border-radius: 6px;
  color: var(--color-error);
  font-family: var(--font-body);
  font-size: 0.85rem;
}

.users-table-wrapper {
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  overflow: hidden;
}

.users-table {
  width: 100%;
  border-collapse: collapse;
  font-family: var(--font-body);
}

.users-table th {
  padding: 0.55rem 0.85rem;
  text-align: left;
  font-size: 0.72rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border-default);
}

.users-table td {
  padding: 0.65rem 0.85rem;
  border-bottom: 1px solid var(--border-subtle);
  color: var(--text-primary);
  font-size: 0.88rem;
}

.users-table tbody tr:last-child td { border-bottom: none; }
.users-table tbody tr:hover td { background: var(--bg-elevated); }

.role-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.2rem 0.6rem;
  border-radius: 4px;
  font-family: var(--font-body);
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.role-badge.admin    { background: var(--bg-error);   color: var(--color-error); }
.role-badge.operator { background: var(--bg-warning);  color: var(--color-warning); }
.role-badge.viewer   { background: var(--bg-elevated); color: var(--text-secondary); }

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  padding: 0.85rem;
  border-top: 1px solid var(--border-subtle);
}

.page-info {
  font-family: var(--font-body);
  font-size: 0.85rem;
  color: var(--text-muted);
  min-width: 60px;
  text-align: center;
}

.warning {
  color: var(--color-error);
  font-family: var(--font-body);
  font-size: 0.88rem;
  font-weight: 500;
}

/* Button system for this view */
.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.45rem 1rem;
  border: 1px solid transparent;
  border-radius: 5px;
  font-family: var(--font-body);
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}
.btn:disabled { opacity: 0.45; cursor: not-allowed; }

.btn-primary {
  background: var(--accent);
  color: var(--bg-base);
  border-color: var(--accent);
  font-weight: 600;
}
.btn-primary:hover:not(:disabled) { filter: brightness(1.1); }

.btn-secondary {
  background: var(--bg-elevated);
  color: var(--text-primary);
  border-color: var(--border-default);
}
.btn-secondary:hover:not(:disabled) { background: var(--bg-card); border-color: var(--border-strong); }

.btn-danger {
  background: var(--bg-error);
  color: var(--color-error);
  border-color: rgba(255,92,122,0.35);
}
.btn-danger:hover:not(:disabled) { background: var(--color-error); color: var(--bg-base); }

.btn-small { padding: 0.25rem 0.7rem; font-size: 0.78rem; }
</style>
