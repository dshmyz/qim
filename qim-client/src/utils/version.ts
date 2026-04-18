import axios from 'axios'
import packageJson from '../../package.json'

const API_BASE_URL = 'http://localhost:8080/api/v1'

// 获取当前版本
export const getCurrentVersion = (): string => {
  return packageJson.version
}

// 检查版本更新
export const checkVersionUpdate = async (): Promise<{
  latestVersion: string
  currentVersion: string
  needUpdate: boolean
  forceUpdate: boolean
  updateUrl: string
  releaseNotes: string
} | null> => {
  try {
    const response = await axios.post(`${API_BASE_URL}/auth/check-version`, {
      version: getCurrentVersion()
    })

    if (response.data.code === 0 && response.data.data) {
      return {
        latestVersion: response.data.data.latest_version,
        currentVersion: response.data.data.current_version,
        needUpdate: response.data.data.need_update,
        forceUpdate: response.data.data.force_update,
        updateUrl: response.data.data.update_url,
        releaseNotes: response.data.data.release_notes
      }
    }
    return null
  } catch (error) {
    console.error('版本检查失败:', error)
    return null
  }
}

// 打开更新链接
export const openUpdateLink = (updateUrl: string): void => {
  window.open(updateUrl, '_blank')
}
