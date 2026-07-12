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

describe('useSettings 扩展字段', () => {
  it('消息设置包含 C1/C2 新字段默认值', () => {
    const settings = useSettings(ref(null), ref('http://localhost:8080'), vi.fn())
    // C1: 发送方式
    expect(settings.messageSettings.value.sendShortcut).toBe('enter')
    // C2: 通知细化
    expect(settings.messageSettings.value.mentionAlert).toBe(true)
    expect(settings.messageSettings.value.notificationPreview).toBe('content')
    expect(settings.messageSettings.value.notificationSound).toBe('default')
    expect(settings.messageSettings.value.dndExceptions).toEqual([])
    expect(settings.messageSettings.value.nightDndEnabled).toBe(false)
    expect(settings.messageSettings.value.nightDndStart).toBe('23:00')
    expect(settings.messageSettings.value.nightDndEnd).toBe('07:00')
  })

  it('从 localStorage 加载时恢复 C1/C2 新字段', () => {
    storage.set('messageSettings', JSON.stringify({
      sendShortcut: 'ctrl_enter',
      mentionAlert: false,
      notificationPreview: 'simple',
      notificationSound: 'chime',
      dndExceptions: ['user1', 'user2'],
      nightDndEnabled: true,
      nightDndStart: '22:00',
      nightDndEnd: '06:00',
    }))

    const settings = useSettings(ref(null), ref('http://localhost:8080'), vi.fn())
    settings.loadSettings()

    expect(settings.messageSettings.value.sendShortcut).toBe('ctrl_enter')
    expect(settings.messageSettings.value.mentionAlert).toBe(false)
    expect(settings.messageSettings.value.notificationPreview).toBe('simple')
    expect(settings.messageSettings.value.notificationSound).toBe('chime')
    expect(settings.messageSettings.value.dndExceptions).toEqual(['user1', 'user2'])
    expect(settings.messageSettings.value.nightDndEnabled).toBe(true)
    expect(settings.messageSettings.value.nightDndStart).toBe('22:00')
    expect(settings.messageSettings.value.nightDndEnd).toBe('06:00')
  })

  it('保存设置时 C1/C2 新字段写入 localStorage', async () => {
    const mockRequest = vi.fn().mockResolvedValue({ code: 0, message: 'success' })
    const settings = useSettings(
      ref({ nickname: 'test', username: 'test' }),
      ref('http://localhost:8080'),
      mockRequest
    )
    settings.messageSettings.value.sendShortcut = 'ctrl_enter'
    settings.messageSettings.value.mentionAlert = false
    settings.messageSettings.value.dndExceptions = ['user_a']

    await settings.saveSettings()

    const savedMessage = JSON.parse(storage.get('messageSettings') || '{}')
    expect(savedMessage.sendShortcut).toBe('ctrl_enter')
    expect(savedMessage.mentionAlert).toBe(false)
    expect(savedMessage.dndExceptions).toEqual(['user_a'])
  })
})
