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
  it('keeps ~/Downloads placeholder when electron IPC unavailable', () => {
    storage.set('messageSettings', JSON.stringify({ defaultSaveDirectory: '~/Downloads' }))

    const settings = useSettings(ref(null), ref('http://localhost:8080'), vi.fn())
    settings.loadSettings()

    // 在无 electron 环境（测试环境）下，~/Downloads 不会被替换为系统默认路径
    // 实际运行时会通过 IPC 获取系统下载目录并替换
    expect(settings.messageSettings.value.defaultSaveDirectory).toBe('~' + '/Downloads')
  })
})
