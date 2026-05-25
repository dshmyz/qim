import { logger } from '../utils/logger'

interface PerformanceMetric {
  name: string
  value: number
  timestamp: number
  category: 'network' | 'render' | 'memory' | 'custom'
}

class PerformanceMonitor {
  private metrics: PerformanceMetric[] = []
  private readonly MAX_METRICS = 1000
  private observer: PerformanceObserver | null = null

  constructor() {
    this.setupPerformanceObserver()
  }

  private setupPerformanceObserver() {
    if (typeof window === 'undefined' || !window.PerformanceObserver) {
      return
    }

    try {
      this.observer = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          this.recordMetric({
            name: entry.name,
            value: entry.duration,
            timestamp: entry.startTime,
            category: 'render'
          })
        }
      })

      this.observer.observe({ entryTypes: ['measure', 'navigation', 'resource'] })
    } catch (error) {
      logger.error('Performance observer setup failed:', error)
    }
  }

  private recordMetric(metric: PerformanceMetric) {
    if (this.metrics.length >= this.MAX_METRICS) {
      this.metrics.shift()
    }
    this.metrics.push(metric)
  }

  measureNetworkRequest(name: string, duration: number) {
    this.recordMetric({
      name: `network:${name}`,
      value: duration,
      timestamp: Date.now(),
      category: 'network'
    })
  }

  measureRenderTime(name: string, duration: number) {
    this.recordMetric({
      name: `render:${name}`,
      value: duration,
      timestamp: Date.now(),
      category: 'render'
    })
  }

  measureMemoryUsage() {
    if (typeof window === 'undefined' || !(performance as any).memory) {
      return
    }

    const memory = (performance as any).memory
    this.recordMetric({
      name: 'memory:used',
      value: memory.usedJSHeapSize,
      timestamp: Date.now(),
      category: 'memory'
    })
  }

  startMeasure(name: string): () => void {
    const startTime = performance.now()
    
    return () => {
      const duration = performance.now() - startTime
      this.recordMetric({
        name: `custom:${name}`,
        value: duration,
        timestamp: Date.now(),
        category: 'custom'
      })
    }
  }

  getMetrics(category?: PerformanceMetric['category']): PerformanceMetric[] {
    if (!category) {
      return [...this.metrics]
    }
    return this.metrics.filter(m => m.category === category)
  }

  getAverageMetric(name: string): number {
    const matchingMetrics = this.metrics.filter(m => m.name === name)
    if (matchingMetrics.length === 0) return 0
    
    const sum = matchingMetrics.reduce((acc, m) => acc + m.value, 0)
    return sum / matchingMetrics.length
  }

  getMetricStats(name: string): {
    count: number
    avg: number
    min: number
    max: number
    p95: number
  } {
    const matchingMetrics = this.metrics
      .filter(m => m.name === name)
      .map(m => m.value)
      .sort((a, b) => a - b)

    if (matchingMetrics.length === 0) {
      return { count: 0, avg: 0, min: 0, max: 0, p95: 0 }
    }

    const count = matchingMetrics.length
    const sum = matchingMetrics.reduce((acc, val) => acc + val, 0)
    const avg = sum / count
    const min = matchingMetrics[0]
    const max = matchingMetrics[count - 1]
    const p95Index = Math.floor(count * 0.95)
    const p95 = matchingMetrics[p95Index]

    return { count, avg, min, max, p95 }
  }

  generateReport(): {
    network: any
    render: any
    memory: any
    custom: any
  } {
    return {
      network: this.getMetricStats('network:request'),
      render: this.getMetricStats('render:component'),
      memory: this.getMetricStats('memory:used'),
      custom: this.getMetricStats('custom:operation')
    }
  }

  clearMetrics() {
    this.metrics = []
  }

  destroy() {
    if (this.observer) {
      this.observer.disconnect()
      this.observer = null
    }
  }
}

export const performanceMonitor = new PerformanceMonitor()

export function measurePerformance(name: string) {
  return performanceMonitor.startMeasure(name)
}
