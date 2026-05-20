import { ref, computed } from 'vue'
import { avatarAPI } from '../api/avatar'
import { useCurrentUser } from './useCurrentUser'
import type {
  AvatarConfig,
  AvatarConfigWithApproval,
  AvatarSession,
  CreateAvatarConfigRequest,
  AvatarWithTools
} from '../types/avatar'

function mapSessionFields(raw: any): AvatarSession {
  return {
    conversationId: raw.conversation_id ?? raw.conversationId,
    avatarEnabled: raw.avatar_enabled ?? raw.avatarEnabled,
    takeoverUntil: raw.takeover_until ?? raw.takeoverUntil,
    lastReplyAt: raw.last_reply_at ?? raw.lastReplyAt
  }
}

export function useAvatar() {
  const { currentUser } = useCurrentUser()
  const config = ref<AvatarConfigWithApproval | null>(null)
  const sessions = ref<AvatarSession[]>([])
  const avatarWithTools = ref<AvatarWithTools | null>(null)
  const loading = ref(false)
  const error = ref('')
  const configLoaded = ref(false)
  const sessionsLoaded = ref(false)

  async function fetchConfig(force = false) {
    if (configLoaded.value && !force) {
      return
    }
    loading.value = true
    error.value = ''
    try {
      config.value = await avatarAPI.getConfig()
      configLoaded.value = true
    } catch (e: any) {
      error.value = e.response?.data?.message || '加载分身配置失败'
      config.value = null
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
      // 只提交用户可编辑的字段，过滤掉只读字段
      const editableFields: (keyof AvatarConfig)[] = [
        'name',
        'enabled',
        'useSystemConfig',
        'modelConfigId',
        'triggerRules',
        'knowledgeScope',
        'replyStrategy',
        'takeoverCooldown',
        'customPersonaAddon'
      ]
      
      const sanitizedUpdates: Partial<AvatarConfig> = {}
      for (const key of editableFields) {
        if (key in updates) {
          (sanitizedUpdates as any)[key] = (updates as any)[key]
        }
      }
      
      config.value = await avatarAPI.updateConfig(sanitizedUpdates)
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

  async function fetchSessions(force = false) {
    if (sessionsLoaded.value && !force) {
      return
    }
    loading.value = true
    error.value = ''
    try {
      const data = await avatarAPI.getSessions()
      sessions.value = data.map(mapSessionFields)
      sessionsLoaded.value = true
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
      const session = mapSessionFields(await avatarAPI.updateSession(Number(convId), enabled))
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
      const session = mapSessionFields(await avatarAPI.takeoverSession(Number(convId)))
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

  // 工具相关方法
  async function fetchAvatarWithTools() {
    loading.value = true
    error.value = ''
    try {
      avatarWithTools.value = await avatarAPI.getAvatarWithTools()
      return avatarWithTools.value
    } catch (e: any) {
      error.value = e.response?.data?.message || '加载工具列表失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function toggleTool(toolId: string) {
    const tool = avatarWithTools.value?.availableTools.find(t => t.id === toolId)
    if (!tool) return

    loading.value = true
    error.value = ''
    try {
      if (tool.enabled) {
        await avatarAPI.unbindToolFromAvatar(toolId)
      } else {
        await avatarAPI.bindToolToAvatar(toolId)
      }
      await fetchAvatarWithTools()
    } catch (e: any) {
      error.value = e.response?.data?.message || '切换工具失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  const avatarConfig = config
  const avatarApprovalStatus = computed(() => config.value?.approvalStatus || 'none')

  return {
    config,
    avatarConfig,
    avatarApprovalStatus,
    sessions,
    avatarWithTools,
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
    cancelApplication,
    fetchAvatarWithTools,
    toggleTool
  }
}
