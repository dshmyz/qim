export interface DBPoolStats {
  maxOpenConnections: number
  openConnections: number
  inUse: number
  idle: number
  waitCount: number
  waitDuration: number
  maxIdleClosed: number
  maxLifetimeClosed: number
}

export interface ServerMetrics {
  cpu: number
  memory: number
  disk: number
  network: {
    in: number
    out: number
  }
  dbPool?: DBPoolStats
  timestamp: string
  uptime: number
  goRoutines: number
}

export interface ServiceStatus {
  name: string
  status: 'healthy' | 'unhealthy' | 'warning'
  message: string
  lastCheck: string
  latency: number
}

export interface AlertRule {
  id: number
  name: string
  metric: 'cpu' | 'memory' | 'disk' | 'network'
  condition: 'gt' | 'lt' | 'eq'
  threshold: number
  duration: number
  notifyMethods: string[]
  notifyTargets: string[]
  enabled: boolean
  createdAt: string
}

export interface AlertHistory {
  id: number
  ruleId: number
  metric: string
  value: number
  status: 'firing' | 'resolved'
  handledAt?: string
  handlerId?: number
  createdAt: string
}
