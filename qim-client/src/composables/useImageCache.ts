import { ref } from 'vue'

interface CacheItem {
  url: string
  dataUrl: string
  timestamp: number
}

const imageCache = ref<Map<string, CacheItem>>(new Map())
const maxCacheSize = 100 // 最多缓存100张图片

export function useImageCache() {

  const getCacheKey = (url: string): string => {
    return btoa(url).replace(/[^a-zA-Z0-9]/g, '')
  }

  const saveToStorage = (cache: Map<string, CacheItem>) => {
    try {
      const data = JSON.stringify(Array.from(cache.entries()))
      localStorage.setItem('image_cache', data)
    } catch (e) {
      console.error('Failed to save image cache:', e)
      if (cache.size > maxCacheSize / 2) {
        clearOldest(cache)
        saveToStorage(cache)
      }
    }
  }

  const loadFromStorage = () => {
    try {
      const data = localStorage.getItem('image_cache')
      if (data) {
        const entries = JSON.parse(data) as [string, CacheItem][]
        imageCache.value = new Map(entries)
      }
    } catch (e) {
      console.error('Failed to load image cache:', e)
    }
  }

  const clearOldest = (cache: Map<string, CacheItem>) => {
    const items = Array.from(cache.values()).sort((a, b) => a.timestamp - b.timestamp)
    const toRemove = items.slice(0, Math.floor(maxCacheSize / 4))
    toRemove.forEach(item => {
      const key = getCacheKey(item.url)
      cache.delete(key)
    })
  }

  const getCachedImage = (url: string): string | null => {
    const key = getCacheKey(url)
    const cached = imageCache.value.get(key)
    if (cached) {
      cached.timestamp = Date.now()
      return cached.dataUrl
    }
    return null
  }

  const cacheImage = async (url: string): Promise<string | null> => {
    const cached = getCachedImage(url)
    if (cached) {
      return cached
    }

    try {
      const response = await fetch(url)
      const blob = await response.blob()

      return new Promise((resolve) => {
        const reader = new FileReader()
        reader.onloadend = () => {
          const dataUrl = reader.result as string

          if (imageCache.value.size >= maxCacheSize) {
            clearOldest(imageCache.value)
          }

          const key = getCacheKey(url)
          imageCache.value.set(key, {
            url,
            dataUrl,
            timestamp: Date.now()
          })
          saveToStorage(imageCache.value)
          resolve(dataUrl)
        }
        reader.onerror = () => {
          resolve(null)
        }
        reader.readAsDataURL(blob)
      })
    } catch (e) {
      console.error('Failed to cache image:', e)
      return null
    }
  }

  const preloadImage = (url: string) => {
    if (!getCachedImage(url)) {
      cacheImage(url)
    }
  }

  const clearCache = () => {
    imageCache.value.clear()
    localStorage.removeItem('image_cache')
  }

  loadFromStorage()

  return {
    getCachedImage,
    cacheImage,
    preloadImage,
    clearCache
  }
}