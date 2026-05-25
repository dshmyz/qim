export interface ConnectionQuality {
  latency: number
  packetLoss: number
  stability: 'excellent' | 'good' | 'fair' | 'poor'
  lastCheckTime: number
}

export interface ConnectionMonitorConfig {
  checkInterval: number
  timeoutThreshold: number
  historySize: number
}

const DEFAULT_CONFIG: ConnectionMonitorConfig = {
  checkInterval: 30000,
  timeoutThreshold: 5000,
  historySize: 10
}

class ConnectionMonitor {
  private config: ConnectionMonitorConfig
  private latencyHistory: number[] = []
  private lastCheckTime: number = 0
  private checkTimer: number | null = null
  private sendTime: number = 0
  private pendingPong: boolean = false

  constructor(config: Partial<ConnectionMonitorConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config }
  }

  start(sendPing: () => void): void {
    if (this.checkTimer) {
      this.stop()
    }

    this.checkTimer = window.setInterval(() => {
      if (!this.pendingPong) {
        this.sendTime = Date.now()
        this.pendingPong = true
        sendPing()
      }
    }, this.config.checkInterval)

    console.log('[ConnectionMonitor] 开始监控')
  }

  stop(): void {
    if (this.checkTimer) {
      clearInterval(this.checkTimer)
      this.checkTimer = null
    }
    this.pendingPong = false
    console.log('[ConnectionMonitor] 停止监控')
  }

  recordPong(): void {
    if (!this.pendingPong) return

    const latency = Date.now() - this.sendTime
    this.latencyHistory.push(latency)
    
    if (this.latencyHistory.length > this.config.historySize) {
      this.latencyHistory.shift()
    }

    this.pendingPong = false
    this.lastCheckTime = Date.now()

    console.log(`[ConnectionMonitor] 延迟: ${latency}ms`)
  }

  recordTimeout(): void {
    this.latencyHistory.push(this.config.timeoutThreshold)
    
    if (this.latencyHistory.length > this.config.historySize) {
      this.latencyHistory.shift()
    }

    this.pendingPong = false
    this.lastCheckTime = Date.now()

    console.warn('[ConnectionMonitor] 心跳超时')
  }

  getQuality(): ConnectionQuality {
    if (this.latencyHistory.length === 0) {
      return {
        latency: 0,
        packetLoss: 0,
        stability: 'excellent',
        lastCheckTime: this.lastCheckTime
      }
    }

    const avgLatency = this.latencyHistory.reduce((a, b) => a + b, 0) / this.latencyHistory.length
    const timeoutCount = this.latencyHistory.filter(l => l >= this.config.timeoutThreshold).length
    const packetLoss = (timeoutCount / this.latencyHistory.length) * 100

    let stability: ConnectionQuality['stability']
    if (avgLatency < 100 && packetLoss === 0) {
      stability = 'excellent'
    } else if (avgLatency < 300 && packetLoss < 10) {
      stability = 'good'
    } else if (avgLatency < 1000 && packetLoss < 30) {
      stability = 'fair'
    } else {
      stability = 'poor'
    }

    return {
      latency: Math.round(avgLatency),
      packetLoss: Math.round(packetLoss * 10) / 10,
      stability,
      lastCheckTime: this.lastCheckTime
    }
  }

  getLatencyHistory(): number[] {
    return [...this.latencyHistory]
  }

  reset(): void {
    this.latencyHistory = []
    this.lastCheckTime = 0
    this.pendingPong = false
    console.log('[ConnectionMonitor] 重置监控数据')
  }
}

export const connectionMonitor = new ConnectionMonitor()
