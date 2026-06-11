<template>
  <div class="groups-container">
    <div class="groups-header">
      <h1>Group Management</h1>
      <button class="btn btn-primary" @click="showCreateGroupModal = true">
        + Create Group
      </button>
    </div>

    <div v-if="loading" class="skeleton-group" aria-busy="true">
      <div class="skeleton skeleton-line" style="width: 38%"></div>
      <div class="skeleton skeleton-block"></div>
      <div class="skeleton skeleton-line" style="width: 62%"></div>
    </div>

    <div v-else-if="error" class="error-message">
      {{ error }}
      <button @click="fetchGroups" class="btn btn-secondary btn-small">Retry</button>
    </div>

    <div v-else class="groups-list">
      <div v-for="group in groups" :key="group.id" class="group-card">
        <div class="group-header">
          <div>
            <h3>{{ group.name }}</h3>
            <p v-if="group.description" class="group-description">{{ group.description }}</p>
          </div>
          <div class="group-actions">
            <button
              class="btn btn-secondary btn-small"
              @click="editGroup(group)"
              title="Edit group"
            >
              Edit
            </button>
            <button
              class="btn btn-danger btn-small"
              @click="deleteGroup(group)"
              title="Delete group"
            >
              Delete
            </button>
          </div>
        </div>

        <div class="group-members">
          <div class="members-header">
            <h4>Agents ({{ group.agentCount || 0 }})</h4>
            <button
              class="btn btn-secondary btn-small"
              @click="manageMembers(group)"
            >
              Manage
            </button>
          </div>
          <div v-if="group.agents && group.agents.length > 0" class="members-list">
            <div v-for="agent in group.agents" :key="agent.id" class="member-item">
              {{ agent.name }}
            </div>
          </div>
          <div v-else class="empty-members">
            No agents in this group
          </div>
        </div>
      </div>

      <div v-if="groups.length === 0" class="empty-state">
        No groups found. Create one to get started.
      </div>
    </div>

    <!-- Create Group Modal -->
    <div v-if="showCreateGroupModal" class="modal-overlay" @click.self="showCreateGroupModal = false">
      <div class="modal-dialog">
        <div class="modal-header">
          <h2>Create Group</h2>
          <button class="modal-close" @click="showCreateGroupModal = false">×</button>
        </div>
        <form @submit.prevent="handleCreateGroup" class="modal-body">
          <div class="form-group">
            <label for="new-group-name">Group Name</label>
            <input
              id="new-group-name"
              v-model="newGroup.name"
              type="text"
              placeholder="Enter group name"
              :disabled="creatingGroup"
              autofocus
            />
          </div>

          <div class="form-group">
            <label for="new-group-description">Description</label>
            <textarea
              id="new-group-description"
              v-model="newGroup.description"
              placeholder="Enter group description (optional)"
              :disabled="creatingGroup"
              rows="4"
            ></textarea>
          </div>

          <div v-if="createError" class="error-message">
            {{ createError }}
          </div>

          <div class="modal-footer">
            <button
              type="button"
              class="btn btn-secondary"
              @click="showCreateGroupModal = false"
              :disabled="creatingGroup"
            >
              Cancel
            </button>
            <button
              type="submit"
              class="btn btn-primary"
              :disabled="creatingGroup || !newGroup.name"
            >
              {{ creatingGroup ? 'Creating...' : 'Create Group' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Edit Group Modal -->
    <div v-if="showEditGroupModal" class="modal-overlay" @click.self="showEditGroupModal = false">
      <div class="modal-dialog">
        <div class="modal-header">
          <h2>Edit Group</h2>
          <button class="modal-close" @click="showEditGroupModal = false">×</button>
        </div>
        <form @submit.prevent="handleEditGroup" class="modal-body">
          <div class="form-group">
            <label for="edit-group-name">Group Name</label>
            <input
              id="edit-group-name"
              v-model="editingGroup.name"
              type="text"
              placeholder="Enter group name"
              :disabled="editingGroup.inProgress"
            />
          </div>

          <div class="form-group">
            <label for="edit-group-description">Description</label>
            <textarea
              id="edit-group-description"
              v-model="editingGroup.description"
              placeholder="Enter group description (optional)"
              :disabled="editingGroup.inProgress"
              rows="4"
            ></textarea>
          </div>

          <div v-if="editError" class="error-message">
            {{ editError }}
          </div>

          <div class="modal-footer">
            <button
              type="button"
              class="btn btn-secondary"
              @click="showEditGroupModal = false"
              :disabled="editingGroup.inProgress"
            >
              Cancel
            </button>
            <button
              type="submit"
              class="btn btn-primary"
              :disabled="editingGroup.inProgress || !editingGroup.name"
            >
              {{ editingGroup.inProgress ? 'Updating...' : 'Update Group' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Manage Members Modal -->
    <div v-if="showMembersModal" class="modal-overlay" @click.self="showMembersModal = false">
      <div class="modal-dialog modal-large">
        <div class="modal-header">
          <h2>Manage Group Members: {{ membersGroup.name }}</h2>
          <button class="modal-close" @click="showMembersModal = false">×</button>
        </div>
        <div class="modal-body">
          <div class="members-section">
            <h4>Current Members</h4>
            <div v-if="membersGroup.agents && membersGroup.agents.length > 0" class="members-list">
              <div v-for="agent in membersGroup.agents" :key="agent.id" class="member-row">
                <span>{{ agent.name }}</span>
                <button
                  class="btn btn-danger btn-small"
                  @click="removeMember(agent.id)"
                  :disabled="managingMembers"
                >
                  Remove
                </button>
              </div>
            </div>
            <div v-else class="empty-members">
              No agents in this group
            </div>
          </div>

          <div class="members-section">
            <h4>Add Members</h4>
            <div class="add-member-form">
              <select
                v-model="agentToAdd"
                class="agent-select"
                :disabled="managingMembers || availableAgents.length === 0"
              >
                <option value="">Select an agent...</option>
                <option v-for="agent in availableAgents" :key="agent.id" :value="agent.id">
                  {{ agent.name }}
                </option>
              </select>
              <button
                class="btn btn-primary btn-small"
                @click="addMember"
                :disabled="managingMembers || !agentToAdd"
              >
                {{ managingMembers ? 'Adding...' : 'Add Agent' }}
              </button>
            </div>
          </div>

          <div v-if="membersError" class="error-message">
            {{ membersError }}
          </div>

          <div class="modal-footer">
            <button
              type="button"
              class="btn btn-primary"
              @click="showMembersModal = false"
              :disabled="managingMembers"
            >
              Done
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation Modal -->
    <div v-if="showDeleteConfirmModal" class="modal-overlay" @click.self="showDeleteConfirmModal = false">
      <div class="modal-dialog">
        <div class="modal-header">
          <h2>Delete Group</h2>
          <button class="modal-close" @click="showDeleteConfirmModal = false">×</button>
        </div>
        <div class="modal-body">
          <p>Are you sure you want to delete group <strong>{{ deletingGroup.name }}</strong>?</p>
          <p class="warning">This action cannot be undone.</p>

          <div class="modal-footer">
            <button
              type="button"
              class="btn btn-secondary"
              @click="showDeleteConfirmModal = false"
              :disabled="deletingGroup.inProgress"
            >
              Cancel
            </button>
            <button
              type="button"
              class="btn btn-danger"
              @click="confirmDelete"
              :disabled="deletingGroup.inProgress"
            >
              {{ deletingGroup.inProgress ? 'Deleting...' : 'Delete Group' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import {
  getGroups,
  createGroup,
  updateGroup,
  deleteGroup as deleteGroupApi,
  getGroupMembers,
  addAgentToGroup,
  removeAgentFromGroup,
  getAgents,
} from '../api'
import { useAuth } from '../composables/useAuth.js'

const auth = useAuth()

// Main state
const groups = ref([])
const loading = ref(false)
const error = ref('')

// Create group modal
const showCreateGroupModal = ref(false)
const creatingGroup = ref(false)
const createError = ref('')
const newGroup = ref({ name: '', description: '' })

// Edit group modal
const showEditGroupModal = ref(false)
const editError = ref('')
const editingGroup = ref({ id: '', name: '', description: '', inProgress: false })

// Manage members modal
const showMembersModal = ref(false)
const membersGroup = ref({ id: '', name: '', agents: [] })
const allAgents = ref([])
const agentToAdd = ref('')
const managingMembers = ref(false)
const membersError = ref('')

const availableAgents = computed(() => {
  const currentIds = new Set(membersGroup.value.agents?.map(a => a.id) || [])
  return allAgents.value.filter(a => !currentIds.has(a.id))
})

// Delete confirmation modal
const showDeleteConfirmModal = ref(false)
const deletingGroup = ref({ id: '', name: '', inProgress: false })

onMounted(() => {
  // Redirect if not admin
  if (!auth.hasRole('admin')) {
    window.location.hash = '/agents'
    return
  }
  fetchGroups()
  fetchAgents()
})

async function fetchGroups() {
  loading.value = true
  error.value = ''

  try {
    const data = await getGroups()
    groups.value = data.groups || []
  } catch (err) {
    error.value = err.message || 'Failed to load groups'
  } finally {
    loading.value = false
  }
}

async function fetchAgents() {
  try {
    const data = await getAgents()
    allAgents.value = data.agents || []
  } catch (err) {
    console.error('Failed to fetch agents:', err)
  }
}

async function handleCreateGroup() {
  if (!newGroup.value.name) {
    createError.value = 'Group name is required'
    return
  }

  creatingGroup.value = true
  createError.value = ''

  try {
    await createGroup(newGroup.value.name, newGroup.value.description)
    showCreateGroupModal.value = false
    newGroup.value = { name: '', description: '' }
    await fetchGroups()
  } catch (err) {
    createError.value = err.message || 'Failed to create group'
  } finally {
    creatingGroup.value = false
  }
}

function editGroup(group) {
  editingGroup.value = {
    id: group.id,
    name: group.name,
    description: group.description || '',
    inProgress: false,
  }
  showEditGroupModal.value = true
  editError.value = ''
}

async function handleEditGroup() {
  if (!editingGroup.value.name) {
    editError.value = 'Group name is required'
    return
  }

  editingGroup.value.inProgress = true
  editError.value = ''

  try {
    await updateGroup(editingGroup.value.id, editingGroup.value.name, editingGroup.value.description)
    showEditGroupModal.value = false
    await fetchGroups()
  } catch (err) {
    editError.value = err.message || 'Failed to update group'
    editingGroup.value.inProgress = false
  }
}

async function manageMembers(group) {
  membersGroup.value = { ...group, agents: group.agents || [] }
  agentToAdd.value = ''
  membersError.value = ''
  showMembersModal.value = true

  try {
    const data = await getGroupMembers(group.id)
    membersGroup.value.agents = data.agents || []
  } catch (err) {
    membersError.value = err.message || 'Failed to load group members'
  }
}

async function addMember() {
  if (!agentToAdd.value) return

  managingMembers.value = true
  membersError.value = ''

  try {
    await addAgentToGroup(membersGroup.value.id, agentToAdd.value)
    agentToAdd.value = ''
    const data = await getGroupMembers(membersGroup.value.id)
    membersGroup.value.agents = data.agents || []
    await fetchGroups()
  } catch (err) {
    membersError.value = err.message || 'Failed to add agent to group'
  } finally {
    managingMembers.value = false
  }
}

async function removeMember(agentId) {
  managingMembers.value = true
  membersError.value = ''

  try {
    await removeAgentFromGroup(membersGroup.value.id, agentId)
    const data = await getGroupMembers(membersGroup.value.id)
    membersGroup.value.agents = data.agents || []
    await fetchGroups()
  } catch (err) {
    membersError.value = err.message || 'Failed to remove agent from group'
  } finally {
    managingMembers.value = false
  }
}

function deleteGroup(group) {
  deletingGroup.value = { id: group.id, name: group.name, inProgress: false }
  showDeleteConfirmModal.value = true
}

async function confirmDelete() {
  deletingGroup.value.inProgress = true

  try {
    await deleteGroupApi(deletingGroup.value.id)
    showDeleteConfirmModal.value = false
    await fetchGroups()
  } catch (err) {
    error.value = err.message || 'Failed to delete group'
    showDeleteConfirmModal.value = false
  }
}
</script>

<style scoped>
/* modal-overlay, modal-dialog, modal-header, modal-body, modal-footer,
   form-group covered by global design system */

.groups-container {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.groups-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-subtle);
}

.groups-header h1 {
  margin: 0;
  font-family: var(--font-display);
  font-size: 1.3rem;
  color: var(--text-primary);
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
  padding: 0.6rem 0.9rem;
  background: var(--bg-error);
  border: 1px solid rgba(255,92,122,0.3);
  border-radius: 6px;
  color: var(--color-error);
  font-family: var(--font-body);
  font-size: 0.85rem;
}

.groups-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
  gap: 1.25rem;
}

.group-card {
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
  transition: border-color 0.15s;
}
.group-card:hover { border-color: var(--border-strong); }

.group-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}

.group-header h3 {
  margin: 0;
  font-family: var(--font-display);
  color: var(--text-primary);
  font-size: 0.95rem;
  font-weight: 700;
}

.group-description {
  margin: 0.3rem 0 0;
  color: var(--text-secondary);
  font-family: var(--font-body);
  font-size: 0.85rem;
}

.group-actions { display: flex; gap: 0.4rem; }

.group-members {
  border-top: 1px solid var(--border-subtle);
  padding-top: 0.85rem;
}

.members-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.65rem;
}

.members-header h4 {
  margin: 0;
  font-family: var(--font-body);
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.members-list {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin-bottom: 0.35rem;
}

.member-item {
  padding: 0.35rem 0.65rem;
  background: var(--bg-elevated);
  border-radius: 4px;
  font-family: var(--font-mono);
  font-size: 0.8rem;
  color: var(--text-primary);
}

.empty-members {
  color: var(--text-muted);
  font-family: var(--font-body);
  font-size: 0.85rem;
}

.members-section {
  border: 1px solid var(--border-default);
  padding: 0.85rem;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.members-section h4 {
  margin: 0;
  font-family: var(--font-body);
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-primary);
}

.member-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.45rem 0.65rem;
  background: var(--bg-elevated);
  border-radius: 4px;
  font-family: var(--font-mono);
  font-size: 0.8rem;
  color: var(--text-primary);
}

.add-member-form { display: flex; gap: 0.5rem; }

.agent-select {
  flex: 1;
  padding: 0.45rem 0.7rem;
  border: 1px solid var(--border-default);
  border-radius: 5px;
  background: var(--bg-input);
  color: var(--text-primary);
  font-family: var(--font-body);
  font-size: 0.88rem;
  transition: border-color 0.15s;
}
.agent-select:focus { outline: none; border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
.agent-select:disabled { opacity: 0.5; cursor: not-allowed; }

.warning {
  color: var(--color-error);
  font-family: var(--font-body);
  font-size: 0.88rem;
  font-weight: 500;
}

/* Button system */
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
