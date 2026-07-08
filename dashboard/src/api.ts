// fallow-ignore-file security-sink
import { z } from 'zod'
import type * as Types from './types/api'
import { describeCron } from './utils/cron'
import { AgentListSchema } from './schemas/agents'
import { JobListSchema } from './schemas/jobs'
import { GroupListSchema } from './schemas/groups'
import { LoginResponseSchema, RefreshTokenResponseSchema } from './schemas/auth'
import { VersionResponseSchema } from './schemas/status'

const BASE_URL = ''

// --- API Contract Error ---
class ApiContractError extends Error {
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
const login = async (username: string, password: string): Promise<Types.LoginResponse> => {
  const res = await fetch(`${BASE_URL}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  }).then(r => r.ok ? r.json() : r.text().then(t => { throw new Error(t) }))
  return validateResponse('/api/auth/login', LoginResponseSchema, res)
}

const logout = () =>
  request('POST', '/api/auth/logout')

const refreshToken = async (): Promise<Types.RefreshTokenResponse> => {
  const res = await request('POST', '/api/auth/refresh')
  return validateResponse('/api/auth/refresh', RefreshTokenResponseSchema, res)
}

const changePassword = (currentPassword: string, newPassword: string) =>
  request('PUT', '/api/auth/change-password', {
    old_password: currentPassword,
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
export const getAgents = async ({ page = 1, limit = 25, search = '', status = '' } = {}) => {
  const res = await request('GET', `/api/agents${buildQuery({ page, limit, search, status })}`)
  return res
}

// --- Jobs ---
export const getJobs = async ({ page = 1, limit = 25, search = '', status = '', agentID = '' } = {}) => {
  const res = await request('GET', `/api/jobs${buildQuery({ page, limit, search, status, agent_id: agentID })}`)
  return res
}

const getJob = (id: string) => request('GET', `/api/jobs/${id}`)

export const createJob = (job: Partial<Types.Job>) => request('POST', '/api/jobs', job)

export const deleteJob = (id: string) => request('DELETE', `/api/jobs/${id}`)

export const cancelJob = (id: string) => request('POST', `/api/jobs/${id}/cancel`)

const updateJobStatus = (id: string, status: string) =>
  request('PATCH', `/api/jobs/${id}/status`, { status })

const triggerJob = (id: string, siteID: string | null = null) =>
  request('POST', `/api/jobs/${id}/trigger${siteID ? `?site=${siteID}` : ''}`)

// --- Job Runs ---
export const getJobRuns = ({ page = 1, limit = 25, jobID = '', agentID = '', status = '' } = {}) =>
  request('GET', `/api/job-runs${buildQuery({ page, limit, job_id: jobID, agent_id: agentID, status: status || undefined })}`)

// --- Updates ---
export const checkUpdate = () =>
  request('GET', '/api/update/check')

const getVersion = async (): Promise<Types.VersionResponse> => {
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

const getTemplate = (id: string) => request('GET', `/api/templates/${id}`)

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

const getFederation = (id: string) =>
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
// fallow-ignore-next-line unused-export
export function saveToken(token: string) {
  localStorage.setItem('arcvault_jwt', token)
  localStorage.setItem('arcvault_token', token)
}

function clearToken() {
  localStorage.removeItem('arcvault_jwt')
  localStorage.removeItem('arcvault_token')
}

function hasToken() {
  return !!getToken()
}

// --- Cron Preview ---
export function cronPreview(expr: string): string {
  return describeCron(expr)
}

// --- Alerts (Phase 17) ---
export const getAlertRules = () =>
  request('GET', '/api/alert-rules')

export const createAlertRule = (rule: Partial<Types.AlertRule>) =>
  request('POST', '/api/alert-rules', rule)

const updateAlertRule = (id: number, rule: Partial<Types.AlertRule>) =>
  request('PUT', `/api/alert-rules/${id}`, rule)

export const deleteAlertRule = (id: number) =>
  request('DELETE', `/api/alert-rules/${id}`)

export const getAlertHistory = () =>
  request('GET', '/api/alert-history')

export const retryAlert = (id: string) =>
  request('POST', `/api/alert-history/${id}/retry`)

// --- Installer Download ---
export async function downloadInstaller() {
  const token = getToken()
  if (!token) {
    throw new Error('No authentication token')
  }

  const res = await fetch(`${BASE_URL}/downloads/installer`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  })

  if (res.status === 401) {
    handle401()
    throw new Error('Session expired. Please log in again.')
  }

  if (!res.ok) {
    const text = await res.text()
    throw new Error(`Failed to download installer: ${res.status} ${text}`)
  }

  // Extract filename from Content-Disposition header
  const disposition = res.headers.get('Content-Disposition') || ''
  const match = disposition.match(/filename=(.+)$/)
  const filename = match ? match[1] : 'ArcVault-Setup.exe'

  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

// --- Agent Bootstrap ---
// fallow-ignore-next-line unused-export
export async function downloadBootstrapScript() {
  const token = getToken()
  if (!token) {
    throw new Error('No authentication token')
  }

  const res = await fetch(`${BASE_URL}/api/admin/bootstrap.ps1`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  })

  if (res.status === 401) {
    handle401()
    throw new Error('Session expired. Please log in again.')
  }

  if (!res.ok) {
    const text = await res.text()
    throw new Error(`Failed to download bootstrap script: ${res.status} ${text}`)
  }

  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'bootstrap.ps1'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
