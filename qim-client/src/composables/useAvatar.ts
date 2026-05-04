import { ref } from 'vue'
import { avatarAPI } from '../api/avatar'
import type {
  AvatarConfig,
  AvatarConfigWithApproval,
  AvatarSession,
  CreateAvatarConfigRequest
} from '../types/avatar'

export function useAvatar() {
  const config = ref<AvatarConfigWithApproval | null>(null)
  const sessions = ref<AvatarSession[]>([])
  const loading = ref(false)
  const error = ref('')

  async function fetchConfig() {
    loading.value = true
    error.value = ''
    try {
      config.value = await avatarAPI.getConfig()
    } catch (e: any) {
      error.value = e.response?.data?.message || '加载分身配置失败'
    } finally {
      loading.value = false
    }
  }

  async function createConfig(data: CreateAvatarConfigRequest) {
    loading.value = true
    error.value = ''
    try {
      const result = await avatarAPI.createConfig(data)
      // 创建后重新获取配置以包含审批状态
      await fetchConfig()
      return config.value
    } catch (e: any) {
      error.value = e.response?.data?.message || '创建分身配置失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function updateConfig(updates: Partial<AvatarConfig>) {
    loading.value = true
    error.value = ''
    try {
      config.value = await avatarAPI.updateConfig(updates)
      return config.value
    } catch (e: any) {
      error.value = e.response?.data?.message || '更新分身配置失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function deleteConfig() {
    loading.value = true
    error.value = ''
    try {
      await avatarAPI.deleteConfig()
      config.value = null
    } catch (e: any) {
      error.value = e.response?.data?.message || '删除分身配置失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function toggleEnabled(enabled: boolean) {
    if (!config.value) return
    await updateConfig({ enabled })
  }

  async function fetchSessions() {
    loading.value = true
    error.value = ''
    try {
      sessions.value = await avatarAPI.getSessions()
    } catch (e: any) {
      error.value = e.response?.data?.message || '加载会话分身状态失败'
    } finally {
      loading.value = false
    }
  }

  async function toggleSession(convId: string | number, enabled: boolean) {
    loading.value = true
    error.value = ''
    try {
      const session = await avatarAPI.updateSession(Number(convId), enabled)
      const idx = sessions.value.findIndex(s => s.conversationId === Number(convId))
      if (idx >= 0) {
        sessions.value[idx] = session
      } else {
        sessions.value.push(session)
      }
      return session
    } catch (e: any) {
      error.value = e.response?.data?.message || '切换会话分身失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function takeoverSession(convId: string | number) {
    loading.value = true
    error.value = ''
    try {
      const session = await avatarAPI.takeoverSession(Number(convId))
      const idx = sessions.value.findIndex(s => s.conversationId === Number(convId))
      if (idx >= 0) {
        sessions.value[idx] = session
      }
      return session
    } catch (e: any) {
      error.value = e.response?.data?.message || '接管分身失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  function getSession(convId: string | number): AvatarSession | undefined {
    return sessions.value.find(s => s.conversationId === Number(convId))
  }

  function isAvatarActive(convId: string | number): boolean {
    const session = getSession(convId)
    if (!session || !session.avatarEnabled) return false
    if (session.takeoverUntil && new Date(session.takeoverUntil) > new Date()) return false
    return true
  }

  // 审批相关方法
  async function applyForApproval() {
    loading.value = true
    error.value = ''
    try {
      config.value = await avatarAPI.applyForApproval()
      return config.value
    } catch (e: any) {
      error.value = e.response?.data?.message || '申请失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function cancelApplication() {
    loading.value = true
    error.value = ''
    try {
      config.value = await avatarAPI.cancelApplication()
      return config.value
    } catch (e: any) {
      error.value = e.response?.data?.message || '取消申请失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    config,
    sessions,
    loading,
    error,
    fetchConfig,
    createConfig,
    updateConfig,
    deleteConfig,
    toggleEnabled,
    fetchSessions,
    toggleSession,
    takeoverSession,
    getSession,
    isAvatarActive,
    applyForApproval,
    cancelApplication
  }
}
