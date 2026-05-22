<template>
  <div class="users-container">
    <div class="users-header">
      <h1>User Management</h1>
      <button class="btn btn-primary" @click="showCreateUserModal = true">
        + Create User
      </button>
    </div>

    <div v-if="loading" class="loading">Loading users...</div>

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
import { ref, onMounted } from 'vue'
import { getUsers, createUser, updateUserRole, deleteUser as deleteUserApi } from '../api.js'
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
    users.value = data.users || []
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
</script>

<script>
import { watch } from 'vue'
</script>

<style scoped>
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
}

.users-header h1 {
  margin: 0;
  font-size: 1.75rem;
  color: var(--text-primary);
}

.loading,
.empty-state {
  padding: 2rem;
  text-align: center;
  color: var(--text-secondary);
  background: var(--bg-secondary);
  border-radius: 8px;
}

.error-message {
  padding: 12px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 4px;
  color: #ef4444;
  font-size: 14px;
}

.users-table-wrapper {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
}

.users-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.95rem;
}

.users-table thead {
  background: var(--bg-primary);
  border-bottom: 2px solid var(--border-color);
}

.users-table th {
  padding: 12px;
  text-align: left;
  font-weight: 600;
  color: var(--text-primary);
}

.users-table td {
  padding: 12px;
  border-bottom: 1px solid var(--border-color);
  color: var(--text-primary);
}

.users-table tbody tr:hover {
  background: var(--bg-primary);
}

.role-badge {
  display: inline-block;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 0.85rem;
  font-weight: 500;
}

.role-badge.admin {
  background: rgba(239, 68, 68, 0.2);
  color: #ef4444;
}

.role-badge.operator {
  background: rgba(245, 158, 11, 0.2);
  color: #f59e0b;
}

.role-badge.viewer {
  background: rgba(107, 114, 128, 0.2);
  color: #6b7280;
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  padding: 1rem;
  border-top: 1px solid var(--border-color);
  background: var(--bg-primary);
}

.page-info {
  font-size: 0.9rem;
  color: var(--text-secondary);
  min-width: 60px;
  text-align: center;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-dialog {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  width: 100%;
  max-width: 450px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.3);
  max-height: 90vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.modal-close {
  background: none;
  border: none;
  font-size: 24px;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: background 0.2s;
}

.modal-close:hover {
  background: var(--bg-secondary);
}

.modal-body {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.modal-footer {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.form-group input,
.form-group select {
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  font-size: 14px;
  transition: border-color 0.2s;
}

.form-group input:focus,
.form-group select:focus {
  outline: none;
  border-color: var(--accent-color);
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
}

.form-group input:disabled,
.form-group select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.readonly-field {
  padding: 10px 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 14px;
}

.warning {
  color: #ef4444;
  font-weight: 500;
}

.btn {
  padding: 10px 16px;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-primary {
  background: var(--accent-color);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: var(--accent-color-hover);
}

.btn-secondary {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--border-color);
}

.btn-danger {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.3);
}

.btn-danger:hover:not(:disabled) {
  background: rgba(239, 68, 68, 0.2);
}

.btn-small {
  padding: 6px 12px;
  font-size: 12px;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Dark mode */
[data-theme='dark'] {
  --bg-primary: #1f2937;
  --bg-secondary: #374151;
  --border-color: #4b5563;
  --text-primary: #f3f4f6;
  --text-secondary: #9ca3af;
  --accent-color: #6366f1;
  --accent-color-hover: #4f46e5;
}

/* Light mode */
[data-theme='light'] {
  --bg-primary: #ffffff;
  --bg-secondary: #f9fafb;
  --border-color: #e5e7eb;
  --text-primary: #111827;
  --text-secondary: #6b7280;
  --accent-color: #6366f1;
  --accent-color-hover: #4f46e5;
}
</style>
