<template>
  <div class="credentials-container">
    <div class="credentials-header">
      <h1>Credential Profiles</h1>
      <button class="btn btn-primary" @click="showCreateModal = true">
        + New Credential
      </button>
    </div>

    <div v-if="loading" class="skeleton-group" aria-busy="true">
      <div class="skeleton skeleton-line" style="width: 38%"></div>
      <div class="skeleton skeleton-block"></div>
      <div class="skeleton skeleton-line" style="width: 62%"></div>
    </div>

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
/* Credentials.vue was written for a light theme with hard-coded colors.
   This block replaces every flat value with Royal Purple tokens so it
   works in both dark (D) and light (E) automatically. Layout unchanged. */

.credentials-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 1.5rem;
}

.credentials-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
}
.credentials-header h1 {
  margin: 0;
  font-family: var(--font-display);
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.01em;
}

.credentials-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.25rem;
}

.credential-card {
  border: 1px solid var(--border-default);
  border-radius: var(--radius-card);
  padding: 1.25rem;
  background: var(--bg-card);
  box-shadow: var(--shadow-sm), var(--edge-highlight);
  transition: box-shadow 0.18s, border-color 0.18s;
}
.credential-card:hover {
  box-shadow: var(--shadow-md);
  border-color: var(--border-strong);
}

.credential-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.credential-info h3 {
  margin: 0 0 0.6rem;
  font-family: var(--font-display);
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.credential-type {
  display: inline-block;
  padding: 3px 10px;
  border-radius: 5px;
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.07em;
  text-transform: uppercase;
}
.credential-type.smb      { background: var(--bg-info);    color: var(--color-info); }
.credential-type.ssh      { background: var(--accent-2-dim);  color: var(--accent-2); }
.credential-type.aws      { background: var(--bg-warning);  color: var(--color-warning); }
.credential-type.database { background: var(--bg-success);  color: var(--color-success); }

.credential-meta small { color: var(--text-muted); font-size: 0.8rem; }

.empty-state {
  grid-column: 1 / -1;
  text-align: center;
  padding: 2.5rem;
  color: var(--text-muted);
  font-family: var(--font-body);
}

.loading {
  text-align: center;
  padding: 2.5rem;
  color: var(--text-muted);
  font-family: var(--font-body);
}

.error-message {
  background: var(--bg-error);
  border: 1px solid rgba(255, 92, 122, 0.35);
  color: var(--color-error);
  padding: 0.9rem 1rem;
  border-radius: 6px;
  margin-bottom: 1.25rem;
  font-family: var(--font-body);
  font-size: 0.88rem;
}

.credential-fields {
  border-top: 1px solid var(--border-subtle);
  padding-top: 1.25rem;
  margin-top: 1.25rem;
}

.radio-group {
  display: flex;
  gap: 1.25rem;
}
.radio-group label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  color: var(--text-secondary);
  font-family: var(--font-body);
  font-size: 0.88rem;
}

.form-actions {
  display: flex;
  gap: 0.65rem;
  justify-content: flex-end;
  margin-top: 1.25rem;
  padding-top: 1.25rem;
  border-top: 1px solid var(--border-subtle);
}

.toast {
  position: fixed;
  bottom: 1.25rem;
  right: 1.25rem;
  padding: 0.9rem 1.25rem;
  border-radius: var(--radius-ctrl);
  box-shadow: var(--shadow-lg);
  display: flex;
  align-items: center;
  gap: 1rem;
  max-width: 400px;
  font-family: var(--font-body);
  font-size: 0.88rem;
  z-index: 9000;
}
.toast-error {
  background: var(--bg-error);
  color: var(--color-error);
  border: 1px solid rgba(255, 92, 122, 0.3);
}
.toast-close {
  background: none;
  border: none;
  color: inherit;
  font-size: 1.2rem;
  cursor: pointer;
  padding: 0;
  opacity: 0.7;
}
.toast-close:hover { opacity: 1; }

.modal-large { max-width: 600px; }
</style>