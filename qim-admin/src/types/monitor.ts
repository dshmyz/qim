export interface ServerMetrics {
  cpu: number
  memory: number
  disk: number
  network: {
    in: number
    out: number
  }
  timestamp: string
}

export interface ServiceStatus {
  name: string
  status: 'healthy' | 'unhealthy' | 'warning'
  message: string
  lastCheck: string
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
