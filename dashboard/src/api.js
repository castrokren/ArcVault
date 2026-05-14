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

// --- agents ---
export const getAgents = () => request('GET', '/api/agents')

// --- jobs ---
export const getJobs = (agentID = null) =>
  request('GET', `/api/jobs${agentID ? `?agent_id=${agentID}` : ''}`)

export const getJob = (id) => request('GET', `/api/jobs/${id}`)

export const createJob = (job) => request('POST', '/api/jobs', job)

export const deleteJob = (id) => request('DELETE', `/api/jobs/${id}`)

export const updateJobStatus = (id, status) =>
  request('PATCH', `/api/jobs/${id}/status`, { status })

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
