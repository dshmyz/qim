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
  dndMode: 'none' | 'mute' | 'custom'
}

export interface AppearanceSettings {
  theme: string
  fontSize: number
}

export interface AdvancedSettings {
  twoFactorEnabled: boolean
}

export interface FileSettings {
  defaultSaveDirectory: string
  autoDownload: boolean
  maxFileSize: number
  allowedFileTypes: string
  autoPreviewImages: boolean
  enableFileHistory: boolean
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
    dndMode: 'none'
  })

  const appearanceSettings = ref<AppearanceSettings>({
    theme: currentTheme.value,
    fontSize: 14
  })

  const advancedSettings = ref<AdvancedSettings>({
    twoFactorEnabled: currentUser.value?.two_factor_enabled || false
  })

  const fileSettings = ref<FileSettings>({
    defaultSaveDirectory: '~/Downloads',
    autoDownload: false,
    maxFileSize: 50,
    allowedFileTypes: 'jpg,png,gif,pdf,doc,docx,xls,xlsx,ppt,pptx,zip,rar',
    autoPreviewImages: true,
    enableFileHistory: true
  })

  const loadSettings = () => {
    // 通过 Electron IPC 获取系统默认下载路径
    if (window.electron?.ipcRenderer?.invoke) {
      window.electron.ipcRenderer.invoke('get-default-download-path').then(path => {
        if (path) fileSettings.value.defaultSaveDirectory = path
      }).catch(() => {
        // IPC 失败则用默认路径
      })
    }

    const savedMessageSettings = localStorage.getItem('messageSettings')
    if (savedMessageSettings) {
      try {
        messageSettings.value = { ...messageSettings.value, ...JSON.parse(savedMessageSettings) }
      } catch (e) {
        console.error('Failed to load message settings:', e)
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

    const savedFileSettings = localStorage.getItem('fileSettings')
    if (savedFileSettings) {
      try {
        fileSettings.value = { ...fileSettings.value, ...JSON.parse(savedFileSettings) }
      } catch (e) {
        console.error('Failed to load file settings:', e)
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
      localStorage.setItem('fileSettings', JSON.stringify(fileSettings.value))

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

  const clearCache = async () => {
    if (confirm('确定要清除缓存吗？')) {
      localStorage.removeItem('messageSettings')
      localStorage.removeItem('appearanceSettings')
      localStorage.removeItem('fileSettings')
      QMessage.success('缓存已清除')
    }
  }

  const saveTwoFactorSetting = async () => {
    try {
      const token = localStorage.getItem('token')
      if (!token) {
        QMessage.error('请先登录')
        return false
      }
      const response = await fetch(`${serverUrl.value}/api/v1/users/me`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          two_factor_enabled: advancedSettings.value.twoFactorEnabled
        })
      })
      const data = await response.json()
      if (data.code === 0) {
        QMessage.success('设置保存成功')
        return true
      } else {
        QMessage.error(data.message || '设置保存失败')
        return false
      }
    } catch (error) {
      console.error('保存双因素认证设置失败:', error)
      QMessage.error('保存设置失败')
      return false
    }
  }

  const browseDefaultSaveDirectory = (callback?: (path: string) => void) => {
    if (window.electron && window.electron.ipcRenderer) {
      window.electron.ipcRenderer.send('open-file-dialog', { properties: ['openDirectory'] })

      const handleResult = (result: any) => {
        window.electron.ipcRenderer.removeListener('file-dialog-result', handleResult)
        if (!result.canceled && result.filePaths && result.filePaths.length > 0) {
          fileSettings.value.defaultSaveDirectory = result.filePaths[0]
          QMessage.success('目录已选择')
          if (callback) {
            callback(result.filePaths[0])
          }
        }
      }

      window.electron.ipcRenderer.on('file-dialog-result', handleResult)
    } else {
      fileSettings.value.defaultSaveDirectory = '~/Downloads'
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
      advancedSettings.value.twoFactorEnabled = currentUser.value.two_factor_enabled || false
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
    advancedSettings,
    fileSettings,
    loadSettings,
    saveSettings,
    clearCache,
    saveTwoFactorSetting,
    browseDefaultSaveDirectory,
    applyFontSize,
    setTheme,
    initTheme,
    updateSettingsProfile
  }
}