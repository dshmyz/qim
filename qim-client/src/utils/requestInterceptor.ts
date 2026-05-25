interface CacheEntry<T = any> {
  data: T
  timestamp: number
  expiresIn: number
}

interface PendingRequest<T = any> {
  promise: Promise<T>
  timestamp: number
}

class RequestInterceptor {
  private cache: Map<string, CacheEntry> = new Map()
  private pendingRequests: Map<string, PendingRequest> = new Map()
  
  private readonly DEFAULT_CACHE_TTL = 5 * 60 * 1000
  private readonly MAX_CACHE_SIZE = 100
  private readonly PENDING_REQUEST_TTL = 30000

  private generateCacheKey(url: string, config?: any): string {
    const method = config?.method || 'GET'
    const data = config?.data ? JSON.stringify(config.data) : ''
    const params = config?.params ? JSON.stringify(config.params) : ''
    return `${method}:${url}:${data}:${params}`
  }

  private isCacheExpired(entry: CacheEntry): boolean {
    return Date.now() - entry.timestamp > entry.expiresIn
  }

  private cleanupCache(): void {
    if (this.cache.size <= this.MAX_CACHE_SIZE) return
    
    const entries = Array.from(this.cache.entries())
      .sort((a, b) => a[1].timestamp - b[1].timestamp)
    
    const toDelete = entries.slice(0, entries.length - this.MAX_CACHE_SIZE)
    toDelete.forEach(([key]) => this.cache.delete(key))
  }

  private cleanupPendingRequests(): void {
    const now = Date.now()
    this.pendingRequests.forEach((pending, key) => {
      if (now - pending.timestamp > this.PENDING_REQUEST_TTL) {
        this.pendingRequests.delete(key)
      }
    })
  }

  async request<T = any>(
    requestFn: () => Promise<T>,
    url: string,
    config?: any
  ): Promise<T> {
    const method = config?.method || 'GET'
    const cacheKey = this.generateCacheKey(url, config)
    const shouldCache = method === 'GET' && config?.cache !== false
    const ttl = config?.cacheTTL || this.DEFAULT_CACHE_TTL

    this.cleanupPendingRequests()

    if (shouldCache) {
      const cached = this.cache.get(cacheKey)
      if (cached && !this.isCacheExpired(cached)) {
        console.log(`[RequestInterceptor] 缓存命中: ${cacheKey}`)
        return cached.data as T
      }
    }

    const pending = this.pendingRequests.get(cacheKey)
    if (pending) {
      console.log(`[RequestInterceptor] 请求去重: ${cacheKey}`)
      return pending.promise as Promise<T>
    }

    const promise = requestFn()
    this.pendingRequests.set(cacheKey, {
      promise,
      timestamp: Date.now()
    })

    try {
      const result = await promise
      
      if (shouldCache) {
        this.cache.set(cacheKey, {
          data: result,
          timestamp: Date.now(),
          expiresIn: ttl
        })
        this.cleanupCache()
      }
      
      return result
    } finally {
      this.pendingRequests.delete(cacheKey)
    }
  }

  clearCache(pattern?: RegExp): void {
    if (!pattern) {
      this.cache.clear()
      console.log('[RequestInterceptor] 清空所有缓存')
      return
    }

    const keysToDelete: string[] = []
    this.cache.forEach((_, key) => {
      if (pattern.test(key)) {
        keysToDelete.push(key)
      }
    })
    
    keysToDelete.forEach(key => this.cache.delete(key))
    console.log(`[RequestInterceptor] 清空匹配缓存: ${keysToDelete.length} 条`)
  }

  getCacheStats(): { size: number; entries: Array<{ key: string; age: number }> } {
    const now = Date.now()
    const entries = Array.from(this.cache.entries())
      .map(([key, entry]) => ({
        key,
        age: now - entry.timestamp
      }))
    
    return {
      size: this.cache.size,
      entries
    }
  }

  getPendingCount(): number {
    return this.pendingRequests.size
  }
}

export const requestInterceptor = new RequestInterceptor()

export function createCachedRequest<T = any>(
  requestFn: () => Promise<T>,
  url: string,
  config?: any
): Promise<T> {
  return requestInterceptor.request(requestFn, url, config)
}
