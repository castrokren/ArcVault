const BASE_URL = ''

function getToken() {
  // Try new JWT token first, fall back to old admin token
  return localStorage.getItem('arcvault_jwt') || localStorage.getItem('arcvault_token') || ''
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
  if (res.status === 204 || res.status === 202) return null
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

// --- auth ---
export const login = (username, password) =>
  fetch(`${BASE_URL}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  }).then(r => r.ok ? r.json() : r.text().then(t => { throw new Error(t) }))

export const logout = () =>
  request('POST', '/api/auth/logout')

export const refreshToken = () =>
  request('POST', '/api/auth/refresh')

export const changePassword = (currentPassword, newPassword) =>
  request('POST', '/api/auth/change-password', {
    current_password: currentPassword,
    new_password: newPassword,
  })

// --- users ---
export const getUsers = ({ page = 1, limit = 25 } = {}) =>
  request('GET', `/api/users${buildQuery({ page, limit })}`)

export const createUser = (username, password, role) =>
  request('POST', '/api/users', { username, password, role })

export const updateUserRole = (id, role) =>
  request('PUT', `/api/users/${id}`, { role })

export const deleteUser = (id) =>
  request('DELETE', `/api/users/${id}`)

// --- groups ---
export const getGroups = () =>
  request('GET', '/api/groups')

export const createGroup = (name, description) =>
  request('POST', '/api/groups', { name, description })

export const updateGroup = (id, name, description) =>
  request('PUT', `/api/groups/${id}`, { name, description })

export const deleteGroup = (id) =>
  request('DELETE', `/api/groups/${id}`)

export const getGroupMembers = (groupId) =>
  request('GET', `/api/groups/${groupId}/agents`)

export const addAgentToGroup = (groupId, agentId) =>
  request('POST', `/api/groups/${groupId}/agents`, { agent_id: agentId })

export const removeAgentFromGroup = (groupId, agentId) =>
  request('DELETE', `/api/groups/${groupId}/agents/${agentId}`)

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

// Trigger a job. Pass siteID to proxy through federation to a sub-coordinator.
export const triggerJob = (id, siteID = null) =>
  request('POST', `/api/jobs/${id}/trigger${siteID ? `?site=${siteID}` : ''}`)

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

// --- templates ---
export const getTemplates = ({ page = 1, limit = 25, search = '' } = {}) =>
  request('GET', `/api/templates${buildQuery({ page, limit, search })}`)

export const getTemplate = (id) => request('GET', `/api/templates/${id}`)

export const createTemplate = (template) => request('POST', '/api/templates', template)

export const updateTemplate = (id, data) => request('PUT', `/api/templates/${id}`, data)

export const deleteTemplate = (id) => request('DELETE', `/api/templates/${id}`)

// Run a template. Pass siteID to proxy through federation to a sub-coordinator.
export const runTemplate = (id, siteID = null) =>
  request('POST', `/api/templates/${id}/run${siteID ? `?site=${siteID}` : ''}`)

// --- federation ---
export const listFederation = () =>
  request('GET', '/api/federation')

export const createFederation = (data) =>
  request('POST', '/api/federation', data)

export const getFederation = (id) =>
  request('GET', `/api/federation/${id}`)

export const updateFederation = (id, data) =>
  request('PUT', `/api/federation/${id}`, data)

export const deleteFederation = (id) =>
  request('DELETE', `/api/federation/${id}`)

export const syncFederation = (id) =>
  request('POST', `/api/federation/${id}/sync`)

export const getFederationAgents = (id) =>
  request('GET', `/api/federation/${id}/agents`)

export const getFederationJobs = (id) =>
  request('GET', `/api/federation/${id}/jobs`)

export const getFederationHistory = (id) =>
  request('GET', `/api/federation/${id}/history`)

export const getFederationHealth = () =>
  request('GET', '/api/federation/health')

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

// --- cronPreview ---
// Returns a plain-English description of a cron expression.
// Covers the most common patterns; falls back to the raw expression.
//
// Field order: minute hour dom month dow
export function cronPreview(expr) {
  if (!expr || !expr.trim()) return ''

  const parts = expr.trim().split(/\s+/)
  if (parts.length !== 5) return expr

  const [min, hour, dom, month, dow] = parts

  const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
  const months = ['January', 'February', 'March', 'April', 'May', 'June',
                  'July', 'August', 'September', 'October', 'November', 'December']

  function fmtTime(h, m) {
    const hh = parseInt(h, 10)
    const mm = parseInt(m, 10)
    const suffix = hh >= 12 ? 'PM' : 'AM'
    const h12 = hh === 0 ? 12 : hh > 12 ? hh - 12 : hh
    const mmStr = mm === 0 ? '' : `:${String(mm).padStart(2, '0')}`
    return `${h12}${mmStr} ${suffix}`
  }

  // Every N minutes: */N * * * *
  if (min.startsWith('*/') && hour === '*' && dom === '*' && month === '*' && dow === '*') {
    const n = min.slice(2)
    return `Every ${n} minute${n === '1' ? '' : 's'}`
  }

  // Every hour at minute N: N * * * *
  if (!min.includes('*') && !min.includes('/') && hour === '*' && dom === '*' && month === '*' && dow === '*') {
    return `Every hour at minute ${min}`
  }

  // Daily at specific time: M H * * *
  if (!min.includes('*') && !min.includes('/') &&
      !hour.includes('*') && !hour.includes('/') &&
      dom === '*' && month === '*' && dow === '*') {
    return `Every day at ${fmtTime(hour, min)}`
  }

  // Weekly on a specific day: M H * * D
  if (!min.includes('*') && !min.includes('/') &&
      !hour.includes('*') && !hour.includes('/') &&
      dom === '*' && month === '*' &&
      !dow.includes('*') && !dow.includes('/') && !dow.includes(',')) {
    const d = parseInt(dow, 10)
    const dayName = d >= 0 && d <= 6 ? days[d] : `day ${dow}`
    return `Every ${dayName} at ${fmtTime(hour, min)}`
  }

  // Monthly on a specific day: M H D * *
  if (!min.includes('*') && !min.includes('/') &&
      !hour.includes('*') && !hour.includes('/') &&
      !dom.includes('*') && !dom.includes('/') &&
      month === '*' && dow === '*') {
    return `Monthly on day ${dom} at ${fmtTime(hour, min)}`
  }

  // Yearly on a specific date: M H D Mo *
  if (!min.includes('*') && !min.includes('/') &&
      !hour.includes('*') && !hour.includes('/') &&
      !dom.includes('*') && !dom.includes('/') &&
      !month.includes('*') && !month.includes('/') &&
      dow === '*') {
    const mo = parseInt(month, 10)
    const monthName = mo >= 1 && mo <= 12 ? months[mo - 1] : `month ${month}`
    return `Yearly on ${monthName} ${dom} at ${fmtTime(hour, min)}`
  }

  // Weekdays only: M H * * 1-5
  if (!min.includes('*') && !min.includes('/') &&
      !hour.includes('*') && !hour.includes('/') &&
      dom === '*' && month === '*' && dow === '1-5') {
    return `Weekdays at ${fmtTime(hour, min)}`
  }

  // Weekends only: M H * * 0,6 or 6,0
  if (!min.includes('*') && !min.includes('/') &&
      !hour.includes('*') && !hour.includes('/') &&
      dom === '*' && month === '*' && (dow === '0,6' || dow === '6,0')) {
    return `Weekends at ${fmtTime(hour, min)}`
  }

  // fallback — show the raw expression
  return expr
}
