<template>
  <div class="credentials-container">
    <div class="credentials-header">
      <h1>Credential Profiles</h1>
      <button class="btn btn-primary" @click="showCreateModal = true">
        + New Credential
      </button>
    </div>

    <div v-if="loading" class="loading">Loading credentials...</div>

    <div v-else-if="error" class="error-message">
      {{ error }}
      <button @click="fetchCredentials" class="btn btn-secondary btn-small">Retry</button>
    </div>

    <div v-else class="credentials-list">
      <div v-if="credentials.length === 0" class="empty-state">
        No credential profiles yet. Create one to get started.
      </div>

      <div v-for="cred in credentials" :key="cred.id" class="credential-card">
        <div class="credential-header">
          <div class="credential-info">
            <h3>{{ cred.name }}</h3>
            <span class="credential-type" :class="cred.type.toLowerCase()">{{ cred.type }}</span>
          </div>
          <button
            class="btn btn-danger btn-small"
            @click="deleteCredential(cred)"
            title="Delete credential"
          >
            Delete
          </button>
        </div>
        <div class="credential-meta">
          <small>Created: {{ formatDate(cred.created_at) }}</small>
        </div>
      </div>
    </div>

    <!-- Create Modal -->
    <div v-if="showCreateModal" class="modal-overlay" @click.self="showCreateModal = false">
      <div class="modal-dialog modal-large">
        <div class="modal-header">
          <h2>Create Credential Profile</h2>
          <button class="modal-close" @click="showCreateModal = false">×</button>
        </div>
        <form @submit.prevent="handleCreateCredential" class="modal-body">
          <div class="form-group">
            <label for="cred-name">Profile Name</label>
            <input
              id="cred-name"
              v-model="newCredential.name"
              type="text"
              placeholder="e.g., Production SMB Share"
              :disabled="creating"
              autofocus
              required
            />
          </div>

          <div class="form-group">
            <label for="cred-type">Credential Type</label>
            <select
              id="cred-type"
              v-model="newCredential.type"
              :disabled="creating"
              required
            >
              <option value="">-- Select Type --</option>
              <option value="SMB">SMB (Windows File Share)</option>
              <option value="SSH">SSH (Linux/Unix)</option>
              <option value="AWS">AWS</option>
              <option value="Database">Database</option>
            </select>
          </div>

          <!-- SMB Fields -->
          <div v-if="newCredential.type === 'SMB'" class="credential-fields">
            <div class="form-group">
              <label for="smb-host">Host</label>
              <input
                id="smb-host"
                v-model="newCredential.data.host"
                type="text"
                placeholder="e.g., \\server\share or server.local"
                :disabled="creating"
                required
              />
            </div>
            <div class="form-group">
              <label for="smb-username">Username</label>
              <input
                id="smb-username"
                v-model="newCredential.data.username"
                type="text"
                placeholder="e.g., domain\user or user@domain"
                :disabled="creating"
                required
              />
            </div>
            <div class="form-group">
              <label for="smb-password">Password</label>
              <input
                id="smb-password"
                v-model="newCredential.data.password"
                type="password"
                placeholder="Password"
                :disabled="creating"
                required
              />
            </div>
          </div>

          <!-- SSH Fields -->
          <div v-if="newCredential.type === 'SSH'" class="credential-fields">
            <div class="form-group">
              <label>SSH Authentication Type</label>
              <div class="radio-group">
                <label>
                  <input
                    type="radio"
                    v-model="sshAuthType"
                    value="key"
                    :disabled="creating"
                  />
                  Private Key
                </label>
                <label>
                  <input
                    type="radio"
                    v-model="sshAuthType"
                    value="password"
                    :disabled="creating"
                  />
                  Password
                </label>
              </div>
            </div>

            <div v-if="sshAuthType === 'key'" class="form-group">
              <label for="ssh-key">Private Key (PEM format)</label>
              <textarea
                id="ssh-key"
                v-model="newCredential.data.key"
                placeholder="-----BEGIN OPENSSH PRIVATE KEY-----&#10;..."
                :disabled="creating"
                rows="6"
                required
              />
            </div>

            <div v-if="sshAuthType === 'password'" class="form-group">
              <label for="ssh-password">Password</label>
              <input
                id="ssh-password"
                v-model="newCredential.data.password"
                type="password"
                placeholder="Password"
                :disabled="creating"
                required
              />
            </div>
          </div>

          <!-- AWS Fields -->
          <div v-if="newCredential.type === 'AWS'" class="credential-fields">
            <div class="form-group">
              <label for="aws-key">Access Key ID</label>
              <input
                id="aws-key"
                v-model="newCredential.data.access_key_id"
                type="text"
                placeholder="AKIA..."
                :disabled="creating"
                required
              />
            </div>
            <div class="form-group">
              <label for="aws-secret">Secret Access Key</label>
              <input
                id="aws-secret"
                v-model="newCredential.data.secret_access_key"
                type="password"
                placeholder="Secret key"
                :disabled="creating"
                required
              />
            </div>
          </div>

          <!-- Database Fields -->
          <div v-if="newCredential.type === 'Database'" class="credential-fields">
            <div class="form-group">
              <label for="db-host">Host</label>
              <input
                id="db-host"
                v-model="newCredential.data.host"
                type="text"
                placeholder="localhost or hostname"
                :disabled="creating"
                required
              />
            </div>
            <div class="form-group">
              <label for="db-port">Port</label>
              <input
                id="db-port"
                v-model="newCredential.data.port"
                type="number"
                placeholder="5432"
                :disabled="creating"
                required
              />
            </div>
            <div class="form-group">
              <label for="db-username">Username</label>
              <input
                id="db-username"
                v-model="newCredential.data.username"
                type="text"
                placeholder="Username"
                :disabled="creating"
                required
              />
            </div>
            <div class="form-group">
              <label for="db-password">Password</label>
              <input
                id="db-password"
                v-model="newCredential.data.password"
                type="password"
                placeholder="Password"
                :disabled="creating"
                required
              />
            </div>
          </div>

          <div class="form-actions">
            <button
              type="button"
              @click="showCreateModal = false"
              class="btn btn-secondary"
              :disabled="creating"
            >
              Cancel
            </button>
            <button type="submit" class="btn btn-primary" :disabled="creating">
              {{ creating ? 'Creating...' : 'Create Credential' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete Confirmation Toast -->
    <div v-if="deleteError" class="toast toast-error">
      {{ deleteError }}
      <button @click="deleteError = null" class="toast-close">×</button>
    </div>
  </div>
</template>

<script>
import { getToken } from '../../api'

export default {
  name: 'Credentials',
  data() {
    return {
      credentials: [],
      loading: false,
      error: null,
      showCreateModal: false,
      creating: false,
      deleteError: null,
      sshAuthType: 'key',
      newCredential: {
        name: '',
        type: '',
        data: {},
      },
    };
  },
  mounted() {
    this.fetchCredentials();
  },
  methods: {
    async fetchCredentials() {
      this.loading = true;
      this.error = null;
      try {
        const response = await fetch('/api/credential-profiles', {
          headers: {
            Authorization: `Bearer ${getToken()}`,
          },
        });

        if (!response.ok) {
          throw new Error(`Failed to fetch credentials: ${response.status}`);
        }

        const data = await response.json();
        this.credentials = Array.isArray(data.data) ? data.data : (Array.isArray(data) ? data : []);
      } catch (err) {
        this.error = err.message;
      } finally {
        this.loading = false;
      }
    },

    async handleCreateCredential() {
      if (!this.newCredential.name || !this.newCredential.type) {
        alert('Please fill in all required fields');
        return;
      }

      this.creating = true;
      try {
        const response = await fetch('/api/credential-profiles', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${getToken()}`,
          },
          body: JSON.stringify({
            name: this.newCredential.name,
            type: this.newCredential.type,
            data: this.newCredential.data,
          }),
        });

        if (!response.ok) {
          const text = await response.text();
          throw new Error(text || `Failed to create credential: ${response.status}`);
        }

        alert('Credential created successfully');
        this.showCreateModal = false;
        this.resetForm();
        await this.fetchCredentials();
      } catch (err) {
        alert(`Error: ${err.message}`);
      } finally {
        this.creating = false;
      }
    },

    async deleteCredential(cred) {
      if (!confirm(`Delete credential "${cred.name}"? This will fail if jobs reference it.`)) {
        return;
      }

      try {
        const response = await fetch(`/api/credential-profiles/${cred.id}`, {
          method: 'DELETE',
          headers: {
            Authorization: `Bearer ${getToken()}`,
          },
        });

        if (response.status === 409) {
          this.deleteError = `Cannot delete "${cred.name}" — it's referenced by one or more jobs.`;
          setTimeout(() => {
            this.deleteError = null;
          }, 5000);
          return;
        }

        if (!response.ok) {
          throw new Error(`Failed to delete credential: ${response.status}`);
        }

        await this.fetchCredentials();
      } catch (err) {
        alert(`Error: ${err.message}`);
      }
    },

    resetForm() {
      this.newCredential = { name: '', type: '', data: {} };
      this.sshAuthType = 'key';
    },

    formatDate(dateStr) {
      return new Date(dateStr).toLocaleString();
    },
  },
};
</script>

<style scoped>
.credentials-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

.credentials-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
}

.credentials-header h1 {
  margin: 0;
  font-size: 28px;
  color: #333;
}

.credentials-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

.credential-card {
  border: 1px solid #ddd;
  border-radius: 8px;
  padding: 20px;
  background: white;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  transition: box-shadow 0.2s;
}

.credential-card:hover {
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

.credential-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 15px;
}

.credential-info h3 {
  margin: 0 0 10px 0;
  font-size: 18px;
  color: #333;
}

.credential-type {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
}

.credential-type.smb {
  background: #e3f2fd;
  color: #1976d2;
}

.credential-type.ssh {
  background: #f3e5f5;
  color: #7b1fa2;
}

.credential-type.aws {
  background: #fff3e0;
  color: #e65100;
}

.credential-type.database {
  background: #e8f5e9;
  color: #2e7d32;
}

.credential-meta small {
  color: #666;
}

.empty-state {
  grid-column: 1 / -1;
  text-align: center;
  padding: 40px;
  color: #999;
}

.loading {
  text-align: center;
  padding: 40px;
  color: #999;
}

.error-message {
  background: #ffebee;
  border: 1px solid #ef5350;
  color: #c62828;
  padding: 15px;
  border-radius: 4px;
  margin-bottom: 20px;
}

.modal-large {
  max-width: 600px;
}

.credential-fields {
  border-top: 1px solid #eee;
  padding-top: 20px;
  margin-top: 20px;
}

.radio-group {
  display: flex;
  gap: 20px;
}

.radio-group label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.form-actions {
  display: flex;
  gap: 10px;
  justify-content: flex-end;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #eee;
}

.toast {
  position: fixed;
  bottom: 20px;
  right: 20px;
  padding: 15px 20px;
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  display: flex;
  align-items: center;
  gap: 15px;
  max-width: 400px;
}

.toast-error {
  background: #ffebee;
  color: #c62828;
  border: 1px solid #ef5350;
}

.toast-close {
  background: none;
  border: none;
  color: inherit;
  font-size: 20px;
  cursor: pointer;
  padding: 0;
}
</style>
