import { ref, watch } from 'vue'
import QMessage from '../utils/qmessage'

export interface SettingsProfile {
  nickname: string
  signature: string
}

export interface MessageSettings {
  notificationsEnabled: boolean
  soundEnabled: boolean
  desktopNotificationsEnabled: boolean
  dndMode: 'none' | 'all_day' | 'custom'
  dndStartTime: string
  dndEndTime: string
  defaultSaveDirectory: string
  // C1: 发送方式
  sendShortcut: 'enter' | 'ctrl_enter'
  // C2: 通知细化
  mentionAlert: boolean
  notificationPreview: 'content' | 'simple'
  dndExceptions: string[]
  nightDndEnabled: boolean
  nightDndStart: string
  nightDndEnd: string
}

export interface AppearanceSettings {
  theme: string
  fontSize: number
}

export function useSettings(currentUser: any, serverUrl: any, request: any) {
  const currentTheme = ref(localStorage.getItem('theme') || 'modern-light')

  const settingsProfile = ref<SettingsProfile>({
    nickname: currentUser.value?.nickname || currentUser.value?.username || '我的账号',
    signature: currentUser.value?.signature || '这个人很懒，什么都没留下'
  })

  const messageSettings = ref<MessageSettings>({
    notificationsEnabled: true,
    soundEnabled: true,
    desktopNotificationsEnabled: true,
    dndMode: 'none',
    dndStartTime: '22:00',
    dndEndTime: '08:00',
    defaultSaveDirectory: '',
    // C1: 发送方式
    sendShortcut: 'enter',
    // C2: 通知细化
    mentionAlert: true,
    notificationPreview: 'content',
    dndExceptions: [],
    nightDndEnabled: false,
    nightDndStart: '23:00',
    nightDndEnd: '07:00'
  })

  const appearanceSettings = ref<AppearanceSettings>({
    theme: currentTheme.value,
    fontSize: 14
  })

  const loadSettings = () => {
    // 1. 先读取 localStorage（同步），判断用户是否已设置自定义目录
    const savedMessageSettings = localStorage.getItem('messageSettings')
    let userSavedDir = ''
    if (savedMessageSettings) {
      try {
        const parsed = JSON.parse(savedMessageSettings)
        messageSettings.value = { ...messageSettings.value, ...parsed }
        userSavedDir = parsed.defaultSaveDirectory || ''
      } catch (e) {
        console.error('Failed to load message settings:', e)
      }
    }

    // 2. 只有用户没有自定义目录（空或 ~/Downloads）时，才去获取系统默认下载路径
    if (!userSavedDir || userSavedDir === '~/Downloads') {
      if (window.electron?.ipcRenderer?.invoke) {
        window.electron.ipcRenderer.invoke('get-default-download-path').then(path => {
          if (path) messageSettings.value.defaultSaveDirectory = path
        }).catch(() => {
          // IPC 失败则用默认路径
        })
      }
    }

    const savedAppearanceSettings = localStorage.getItem('appearanceSettings')
    if (savedAppearanceSettings) {
      try {
        appearanceSettings.value = { ...appearanceSettings.value, ...JSON.parse(savedAppearanceSettings) }
        if (appearanceSettings.value.theme !== currentTheme.value) {
          currentTheme.value = appearanceSettings.value.theme
        }
      } catch (e) {
        console.error('Failed to load appearance settings:', e)
      }
    }

    const savedFontSize = localStorage.getItem('fontSize')
    if (savedFontSize) {
      appearanceSettings.value.fontSize = parseInt(savedFontSize)
    }
  }

  const saveSettings = async () => {
    try {
      const profileResponse = await request('/api/v1/users/me', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          nickname: settingsProfile.value.nickname,
          signature: settingsProfile.value.signature
        })
      })

      if (appearanceSettings.value.theme !== currentTheme.value) {
        localStorage.setItem('theme', appearanceSettings.value.theme)
      }

      localStorage.setItem('fontSize', appearanceSettings.value.fontSize.toString())
      localStorage.setItem('messageSettings', JSON.stringify(messageSettings.value))
      localStorage.setItem('appearanceSettings', JSON.stringify(appearanceSettings.value))

      if (profileResponse.code === 0) {
        if (currentUser.value) {
          currentUser.value.username = settingsProfile.value.nickname
        }
        QMessage.success('保存成功')
        return true
      } else {
        QMessage.error('保存失败: ' + profileResponse.message)
        return false
      }
    } catch (error) {
      console.error('保存设置失败:', error)
      const errorMessage = error instanceof Error ? error.message : '未知错误'
      QMessage.error('保存失败: ' + errorMessage)
      return false
    }
  }

  const browseDefaultSaveDirectory = (callback?: (path: string) => void) => {
    if (window.electron && window.electron.ipcRenderer) {
      window.electron.ipcRenderer.send('open-file-dialog', { properties: ['openDirectory'] })

      const handleResult = (event: any, result: any) => {
        window.electron.ipcRenderer.removeListener('file-dialog-result', handleResult)
        if (!result.canceled && result.filePaths && result.filePaths.length > 0) {
          messageSettings.value.defaultSaveDirectory = result.filePaths[0]
          QMessage.success('目录已选择')
          if (callback) {
            callback(result.filePaths[0])
          }
        }
      }

      window.electron.ipcRenderer.on('file-dialog-result', handleResult)
    } else {
      messageSettings.value.defaultSaveDirectory = ''
      QMessage.info('使用默认下载目录')
    }
  }

  const applyFontSize = (fontSize: number) => {
    document.documentElement.style.setProperty('--font-size-base', `${fontSize}px`)
    document.body.style.fontSize = `${fontSize}px`
    const container = document.querySelector('.im-container') as HTMLElement
    if (container) {
      container.style.fontSize = fontSize + 'px'
    }
  }

  const setTheme = (theme: string) => {
    currentTheme.value = theme
    localStorage.setItem('theme', theme)
    document.documentElement.setAttribute('data-theme', theme)
  }

  const initTheme = () => {
    const savedTheme = localStorage.getItem('theme') || 'modern-light'
    currentTheme.value = savedTheme
    document.documentElement.setAttribute('data-theme', savedTheme)

    const savedFontSize = localStorage.getItem('fontSize')
    if (savedFontSize) {
      appearanceSettings.value.fontSize = parseInt(savedFontSize)
    }
    applyFontSize(appearanceSettings.value.fontSize)
  }

  const updateSettingsProfile = () => {
    if (currentUser.value) {
      settingsProfile.value.nickname = currentUser.value.nickname || currentUser.value.username || '我的账号'
      settingsProfile.value.signature = currentUser.value.signature || '这个人很懒，什么都没留下'
    }
  }

  watch(currentUser, () => {
    updateSettingsProfile()
  }, { immediate: true })

  return {
    currentTheme,
    settingsProfile,
    messageSettings,
    appearanceSettings,
    loadSettings,
    saveSettings,
    browseDefaultSaveDirectory,
    applyFontSize,
    setTheme,
    initTheme,
    updateSettingsProfile
  }
}
