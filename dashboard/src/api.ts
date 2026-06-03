import { z } from 'zod'
import type * as Types from './types/api'
import { AgentListSchema } from './schemas/agents'
import { JobListSchema } from './schemas/jobs'
import { GroupListSchema } from './schemas/groups'
import { LoginResponseSchema, RefreshTokenResponseSchema } from './schemas/auth'
import { VersionResponseSchema } from './schemas/status'

const BASE_URL = ''

// --- API Contract Error ---
export class ApiContractError extends Error {
  constructor(
    public endpoint: string,
    public zodError: z.ZodError<any>
  ) {
    super(`[API Contract] ${endpoint} shape mismatch`)
    this.name = 'ApiContractError'
  }
}

// --- Token Management ---
export function getToken() {
  return localStorage.getItem('arcvault_jwt') || localStorage.getItem('arcvault_token') || ''
}

function getAuthHeaders() {
  const token = getToken()
  if (!token) return {}
  return {
    Authorization: `Bearer ${token}`,
  }
}

function handle401() {
  localStorage.removeItem('arcvault_jwt')
  localStorage.removeItem('arcvault_token')
  localStorage.removeItem('arcvault_user')
  localStorage.removeItem('arcvault_remember_me')
  window.location.hash = '#/login'
}

async function request(method: string, path: string, body: any = null) {
  const opts: RequestInit = {
    method,
    headers: {
      ...getAuthHeaders(),
      'Content-Type': 'application/json',
    },
  }
  if (body) opts.body = JSON.stringify(body)

  const res = await fetch(`${BASE_URL}${path}`, opts)

  if (res.status === 401) {
    handle401()
    throw new Error('Session expired. Please log in again.')
  }

  if (!res.ok) {
    const text = await res.text()
    throw new Error(`${method} ${path} → ${res.status}: ${text}`)
  }
  if (res.status === 204 || res.status === 202) return null
  return res.json()
}

function buildQuery(params: Record<string, any>) {
  const q = Object.entries(params)
    .filter(([, v]) => v !== null && v !== undefined && v !== '' && v !== 0)
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`)
    .join('&')
  return q ? `?${q}` : ''
}

// --- Validation Wrapper ---
function validateResponse<T>(endpoint: string, schema: z.Schema<T>, data: any): T {
  const result = schema.safeParse(data)
  if (!result.success) {
    console.error(`[API Contract] ${endpoint} validation failed:`, result.error.format())
    throw new ApiContractError(endpoint, result.error)
  }
  return result.data
}

// --- Auth ---
export const login = async (username: string, password: string): Promise<Types.LoginResponse> => {
  const res = await fetch(`${BASE_URL}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  }).then(r => r.ok ? r.json() : r.text().then(t => { throw new Error(t) }))
  return validateResponse('/api/auth/login', LoginResponseSchema, res)
}

export const logout = () =>
  request('POST', '/api/auth/logout')

export const refreshToken = async (): Promise<Types.RefreshTokenResponse> => {
  const res = await request('POST', '/api/auth/refresh')
  return validateResponse('/api/auth/refresh', RefreshTokenResponseSchema, res)
}

export const changePassword = (currentPassword: string, newPassword: string) =>
  request('PUT', '/api/auth/change-password', {
    current_password: currentPassword,
    new_password: newPassword,
  })

// --- Users ---
export const getUsers = ({ page = 1, limit = 25 } = {}) =>
  request('GET', `/api/users${buildQuery({ page, limit })}`)

export const createUser = (username: string, password: string, role: string) =>
  request('POST', '/api/users', { username, password, role })

export const updateUserRole = (id: number, role: string) =>
  request('PUT', `/api/users/${id}/role`, { role })

export const deleteUser = (id: number) =>
  request('DELETE', `/api/users/${id}`)

// --- Groups ---
export const getGroups = async (): Promise<Types.Group[]> => {
  const res = await request('GET', '/api/groups')
  return validateResponse('/api/groups', GroupListSchema, res)
}

export const createGroup = (name: string, description: string) =>
  request('POST', '/api/groups', { name, description })

export const updateGroup = (id: number, name: string, description: string) =>
  request('PUT', `/api/groups/${id}`, { name, description })

export const deleteGroup = (id: number) =>
  request('DELETE', `/api/groups/${id}`)

export const getGroupMembers = (groupId: number) =>
  request('GET', `/api/groups/${groupId}/agents`)

export const addAgentToGroup = (groupId: number, agentId: string) =>
  request('POST', `/api/groups/${groupId}/agents`, { agent_id: agentId })

export const removeAgentFromGroup = (groupId: number, agentId: string) =>
  request('DELETE', `/api/groups/${groupId}/agents/${agentId}`)

// --- Agents ---
export const getAgents = async ({ page = 1, limit = 25, search = '', status = '' } = {}): Promise<Types.Agent[]> => {
  const res = await request('GET', `/api/agents${buildQuery({ page, limit, search, status })}`)
  return validateResponse('/api/agents', AgentListSchema, res)
}

// --- Jobs ---
export const getJobs = async ({ page = 1, limit = 25, search = '', status = '', agentID = '' } = {}): Promise<Types.Job[]> => {
  const res = await request('GET', `/api/jobs${buildQuery({ page, limit, search, status, agent_id: agentID })}`)
  return validateResponse('/api/jobs', JobListSchema, res)
}

export const getJob = (id: string) => request('GET', `/api/jobs/${id}`)

export const createJob = (job: Partial<Types.Job>) => request('POST', '/api/jobs', job)

export const deleteJob = (id: string) => request('DELETE', `/api/jobs/${id}`)

export const updateJobStatus = (id: string, status: string) =>
  request('PATCH', `/api/jobs/${id}/status`, { status })

export const triggerJob = (id: string, siteID: string | null = null) =>
  request('POST', `/api/jobs/${id}/trigger${siteID ? `?site=${siteID}` : ''}`)

// --- Job Runs ---
export const getJobRuns = ({ page = 1, limit = 25, jobID = '', agentID = '' } = {}) =>
  request('GET', `/api/job-runs${buildQuery({ page, limit, job_id: jobID, agent_id: agentID })}`)

// --- Rollback ---
export const getRollbackAvailable = () =>
  request('GET', '/api/rollback-available')

export const applyRollback = () =>
  request('POST', '/api/rollback')

export const applyAgentRollback = (agentId: string) =>
  request('POST', `/api/agents/${agentId}/rollback`)

// --- Updates ---
export const checkUpdate = () =>
  request('GET', '/api/update/check')

export const getVersion = async (): Promise<Types.VersionResponse> => {
  const res = await request('GET', '/api/version')
  return validateResponse('/api/version', VersionResponseSchema, res)
}

export const applyCoordinatorUpdate = () =>
  request('POST', '/api/update/apply')

export const applyAgentUpdate = (agentId: string) =>
  request('POST', `/api/agents/${agentId}/update`)

// --- Templates ---
export const getTemplates = ({ page = 1, limit = 25, search = '' } = {}) =>
  request('GET', `/api/templates${buildQuery({ page, limit, search })}`)

export const getTemplate = (id: string) => request('GET', `/api/templates/${id}`)

export const createTemplate = (template: Partial<Types.Template>) => request('POST', '/api/templates', template)

export const updateTemplate = (id: string, data: Partial<Types.Template>) => request('PUT', `/api/templates/${id}`, data)

export const deleteTemplate = (id: string) => request('DELETE', `/api/templates/${id}`)

export const runTemplate = (id: string, siteID: string | null = null) =>
  request('POST', `/api/templates/${id}/run${siteID ? `?site=${siteID}` : ''}`)

// --- Federation ---
export const listFederation = () =>
  request('GET', '/api/federation')

export const createFederation = (data: Partial<Types.FederationNode>) =>
  request('POST', '/api/federation', data)

export const getFederation = (id: string) =>
  request('GET', `/api/federation/${id}`)

export const updateFederation = (id: string, data: Partial<Types.FederationNode>) =>
  request('PUT', `/api/federation/${id}`, data)

export const deleteFederation = (id: string) =>
  request('DELETE', `/api/federation/${id}`)

export const syncFederation = (id: string) =>
  request('POST', `/api/federation/${id}/sync`)

export const getFederationAgents = (id: string) =>
  request('GET', `/api/federation/${id}/agents`)

export const getFederationJobs = (id: string) =>
  request('GET', `/api/federation/${id}/jobs`)

export const getFederationHistory = (id: string) =>
  request('GET', `/api/federation/${id}/history`)

export const getFederationHealth = () =>
  request('GET', '/api/federation/health')

// --- Token Helpers ---
export function saveToken(token: string) {
  localStorage.setItem('arcvault_jwt', token)
  localStorage.setItem('arcvault_token', token)
}

export function clearToken() {
  localStorage.removeItem('arcvault_jwt')
  localStorage.removeItem('arcvault_token')
}

export function hasToken() {
  return !!getToken()
}

// --- Cron Preview ---
export function cronPreview(expr: string): string {
  if (!expr || !expr.trim()) return ''

  const parts = expr.trim().split(/\s+/)
  if (parts.length !== 5) return expr

  const [min, hour, dom, month, dow] = parts

  const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
  const months = ['January', 'February', 'March', 'April', 'May', 'June',
                  'July', 'August', 'September', 'October', 'November', 'December']

  function fmtTime(h: string, m: string) {
    const hh = parseInt(h, 10)
    const mm = parseInt(m, 10)
    const suffix = hh >= 12 ? 'PM' : 'AM'
    const h12 = hh === 0 ? 12 : hh > 12 ? hh - 12 : hh
    const mmStr = mm === 0 ? '' : `:${String(mm).padStart(2, '0')}`
    return `${h12}${mmStr} ${suffix}`
  }

  if (min.startsWith('*/') && hour === '*' && dom === '*' && month === '*' && dow === '*') {
    const n = min.slice(2)
    return `Every ${n} minute${n === '1' ? '' : 's'}`
  }

  if (!min.includes('*') && !min.includes('/') && hour === '*' && dom === '*' && month === '*' && dow === '*') {
    return `Every hour at minute ${min}`
  }

  if (!min.includes('*') && !min.includes('/') &&
      !hour.includes('*') && !hour.includes('/') &&
      dom === '*' && month === '*' && dow === '*') {
    return `Every day at ${fmtTime(hour, min)}`
  }

  if (!min.includes('*') && !min.includes('/') &&
      !hour.includes('*') && !hour.includes('/') &&
      dom === '*' && month === '*' &&
      !dow.includes('*') && !dow.includes('/') && !dow.includes(',')) {
    const d = parseInt(dow, 10)
    const dayName = d >= 0 && d <= 6 ? days[d] : `day ${dow}`
    return `Every ${dayName} at ${fmtTime(hour, min)}`
  }

  if (!min.includes('*') && !min.includes('/') &&
      !hour.includes('*') && !hour.includes('/') &&
      !dom.includes('*') && !dom.includes('/') &&
      month === '*' && dow === '*') {
    return `Monthly on day ${dom} at ${fmtTime(hour, min)}`
  }

  if (!min.includes('*') && !min.includes('/') &&
      !hour.includes('*') && !hour.includes('/') &&
      !dom.includes('*') && !dom.includes('/') &&
      !month.includes('*') && !month.includes('/') &&
      dow === '*') {
    const mo = parseInt(month, 10)
    const monthName = mo >= 1 && mo <= 12 ? months[mo - 1] : `month ${month}`
    return `Yearly on ${monthName} ${dom} at ${fmtTime(hour, min)}`
  }

  if (!min.includes('*') && !min.includes('/') &&
      !hour.includes('*') && !hour.includes('/') &&
      dom === '*' && month === '*' && dow === '1-5') {
    return `Weekdays at ${fmtTime(hour, min)}`
  }

  if (!min.includes('*') && !min.includes('/') &&
      !hour.includes('*') && !hour.includes('/') &&
      dom === '*' && month === '*' && (dow === '0,6' || dow === '6,0')) {
    return `Weekends at ${fmtTime(hour, min)}`
  }

  return expr
}

// --- Alerts (Phase 17) ---
export const getAlertRules = () =>
  request('GET', '/api/alert-rules')

export const createAlertRule = (rule: Partial<Types.AlertRule>) =>
  request('POST', '/api/alert-rules', rule)

export const updateAlertRule = (id: number, rule: Partial<Types.AlertRule>) =>
  request('PUT', `/api/alert-rules/${id}`, rule)

export const deleteAlertRule = (id: number) =>
  request('DELETE', `/api/alert-rules/${id}`)

export const getAlertHistory = () =>
  request('GET', '/api/alert-history')

export const retryAlert = (id: string) =>
  request('POST', `/api/alert-history/${id}/retry`)
