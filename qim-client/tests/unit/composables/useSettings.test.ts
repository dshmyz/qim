import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useSettings } from '@/composables/useSettings'

const storage = new Map<string, string>()

beforeEach(() => {
  storage.clear()
  vi.mocked(localStorage.getItem).mockImplementation((key) => storage.get(key) ?? null)
  vi.mocked(localStorage.setItem).mockImplementation((key, value) => {
    storage.set(key, value)
  })
  vi.mocked(localStorage.removeItem).mockImplementation((key) => {
    storage.delete(key)
  })
  vi.mocked(localStorage.clear).mockImplementation(() => {
    storage.clear()
  })
  ;(window as any).electron = undefined
})

describe('useSettings file settings', () => {
  it('does not keep the ~/Downloads placeholder as a real download directory', () => {
    storage.set('fileSettings', JSON.stringify({ defaultSaveDirectory: '~/Downloads' }))

    const settings = useSettings(ref(null), ref('http://localhost:8080'), vi.fn())
    settings.loadSettings()

    expect(settings.fileSettings.value.defaultSaveDirectory).toBe('')
  })
})
