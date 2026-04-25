import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useImageCache } from '@/composables/useImageCache'

describe('useImageCache', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.getItem = vi.fn(() => null)
    localStorage.setItem = vi.fn()
    localStorage.removeItem = vi.fn()
  })

  describe('getCachedImage', () => {
    it('当缓存中不存在时应该返回 null', () => {
      const { getCachedImage } = useImageCache()
      const result = getCachedImage('http://example.com/image.png')
      expect(result).toBeNull()
    })
  })

  describe('clearCache', () => {
    it('应该清除缓存和 localStorage', () => {
      const { clearCache } = useImageCache()
      clearCache()
      expect(localStorage.removeItem).toHaveBeenCalledWith('image_cache')
    })
  })

  describe('preloadImage', () => {
    it('应该调用 cacheImage 开始预加载', () => {
      const { preloadImage } = useImageCache()
      global.fetch = vi.fn().mockResolvedValue({
        blob: () => Promise.resolve(new Blob(['test'], { type: 'image/png' })),
      })
      
      preloadImage('http://example.com/test.png')
      
      expect(global.fetch).toHaveBeenCalledWith('http://example.com/test.png')
    })
  })
})
