const BASE_URL = 'http://localhost:443'

function getToken() {
  return localStorage.getItem('arcvault_token') || ''
}

async function request(method, path, body = null) {
  const opts = {
    method,
    headers: {
      'Authorization': `Bearer ${getToken()}`,
      'Content-Type': 'application/json',
    },
  }
  if (body) opts.body = JSON.stringify(body)

  const res = await fetch(`${BASE_URL}${path}`, opts)
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`${method} ${path} → ${res.status}: ${text}`)
  }
  if (res.status === 204) return null
  return res.json()
}

// Build a query string from a params object, omitting null/undefined/''/0 values.
function buildQuery(params) {
  const q = Object.entries(params)
    .filter(([, v]) => v !== null && v !== undefined && v !== '' && v !== 0)
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`)
    .join('&')
  return q ? `?${q}` : ''
}

// --- agents ---
export const getAgents = ({ page = 1, limit = 25, search = '', status = '' } = {}) =>
  request('GET', `/api/agents${buildQuery({ page, limit, search, status })}`)

// --- jobs ---
export const getJobs = ({ page = 1, limit = 25, search = '', status = '', agentID = '' } = {}) =>
  request('GET', `/api/jobs${buildQuery({ page, limit, search, status, agent_id: agentID })}`)

export const getJob = (id) => request('GET', `/api/jobs/${id}`)

export const createJob = (job) => request('POST', '/api/jobs', job)

export const deleteJob = (id) => request('DELETE', `/api/jobs/${id}`)

export const updateJobStatus = (id, status) =>
  request('PATCH', `/api/jobs/${id}/status`, { status })

// --- job runs ---
export const getJobRuns = ({ page = 1, limit = 25, jobID = '', agentID = '' } = {}) =>
  request('GET', `/api/job-runs${buildQuery({ page, limit, job_id: jobID, agent_id: agentID })}`)

// --- rollback ---
export const getRollbackAvailable = () =>
  request('GET', '/api/rollback-available')

export const applyRollback = () =>
  request('POST', '/api/rollback')

export const applyAgentRollback = (agentId) =>
  request('POST', `/api/agents/${agentId}/rollback`)

// --- token helpers ---
export function saveToken(token) {
  localStorage.setItem('arcvault_token', token)
}

export function clearToken() {
  localStorage.removeItem('arcvault_token')
}

export function hasToken() {
  return !!getToken()
}
