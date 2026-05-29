export interface AuthProvider {
  id: number
  name: string
  protocol: 'ldap' | 'oauth' | 'cas'
  type: 'direct' | 'redirect'
  enabled: boolean
  priority: number
  config: string
  display_name: string
  icon: string
  created_at: string
  updated_at: string
}

export interface OrgSyncConfig {
  id: number
  name: string
  enabled: boolean
  sync_type: string
  schedule: string
  config: string
  last_sync_at: string | null
  last_sync_status: string
  created_at: string
  updated_at: string
}

export interface OrgSyncLog {
  id: number
  config_id: number
  status: 'running' | 'success' | 'failed'
  started_at: string
  finished_at: string | null
  stats: string
  error_message: string
  created_at: string
}
