// API response types — mirrors Go coordinator response structs
// Last synced: 2026-06-03

export interface ProgressData {
  files_processed: number
  bytes_transferred: number
  total_files: number
  total_bytes: number
}

export interface Job {
  id: string
  agent_id: string
  name: string
  source_path: string
  dest_path: string
  schedule?: string
  sync_flags?: Record<string, unknown>
  status: string
  progress?: ProgressData
  created_at: string
}

export interface Agent {
  id: string
  hostname: string
  os: string
  arch: string
  version: string
  status: string
  last_seen?: string
  registered_at: string
  rollback_available: boolean
}

export interface Group {
  id: number
  name: string
  description: string
  agent_count: number
}

export interface LoginResponse {
  token: string
  role: string
  must_change_password: boolean
}

export interface VersionResponse {
  version: string
  build_time: string
  os: string
  arch: string
  go_version: string
  uptime: string
}

export interface User {
  id: number
  username: string
  role: string
}

export interface AlertRule {
  id: number
  name: string
  condition: string
  action: string
  enabled: boolean
}

export interface FederationNode {
  id: string
  name: string
  url: string
  status: string
}

export interface JobRun {
  id: string
  job_id: string
  agent_id: string
  status: string
  started_at: string
  completed_at?: string
  error?: string
}

export interface Template {
  id: string
  name: string
  description?: string
  source_path: string
  dest_path: string
}

export interface RefreshTokenResponse {
  token: string
  role: string
  must_change_password: boolean
}

export interface ErrorResponse {
  error: string
}
