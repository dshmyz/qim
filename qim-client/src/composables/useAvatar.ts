import { ref, computed } from 'vue'
import { avatarAPI } from '../api/avatar'
import { useCurrentUser } from './useCurrentUser'
import type {
  AvatarConfig,
  AvatarConfigWithApproval,
  AvatarSession,
  CreateAvatarConfigRequest
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
  const loading = ref(true)
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
      error.value = e.message || '加载分身配置失败'
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
      await fetchConfig(true)
      return config.value
    } catch (e: any) {
      error.value = e.message || '创建分身配置失败'
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
      // enabled 只允许关闭（false），不允许自行开启（true）
      const editableFields: (keyof AvatarConfig)[] = [
        'name',
        'useSystemConfig',
        'modelConfigId',
        'triggerRules',
        'knowledgeScope',
        'replyStrategy',
        'takeoverCooldown',
        'selfMessagePause',
        'activateByDefault',
        'customPersonaAddon'
      ]

      const sanitizedUpdates: Partial<AvatarConfig> = {}
      for (const key of editableFields) {
        if (key in updates) {
          (sanitizedUpdates as any)[key] = (updates as any)[key]
        }
      }

      // enabled 特殊处理：只允许关闭，不允许开启
      if ('enabled' in updates && updates.enabled === false) {
        sanitizedUpdates.enabled = false
      }
      
      config.value = await avatarAPI.updateConfig(sanitizedUpdates)
      return config.value
    } catch (e: any) {
      error.value = e.message || '更新分身配置失败'
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
      error.value = e.message || '删除分身配置失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function toggleEnabled(enabled: boolean) {
    if (!config.value) return
    if (enabled) {
      // 启用分身需要走审批流程
      await applyForApproval()
    } else {
      // 关闭分身：用户可以自行关闭，直接调用 API 绕过 editableFields 过滤
      loading.value = true
      error.value = ''
      try {
        config.value = await avatarAPI.updateConfig({ enabled: false })
      } catch (e: any) {
        error.value = e.message || '关闭分身失败'
        throw e
      } finally {
        loading.value = false
      }
    }
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
      error.value = e.message || '加载会话分身状态失败'
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
      error.value = e.message || '切换会话分身失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  // 重置所有会话设置为跟随全局默认：删除全部 session 行，sessions 列表清空并强制下次重新拉取。
  async function clearSessions() {
    loading.value = true
    error.value = ''
    try {
      await avatarAPI.clearSessions()
      sessions.value = []
      sessionsLoaded.value = false
    } catch (e: any) {
      error.value = e.message || '重置会话设置失败'
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
      error.value = e.message || '接管分身失败'
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
    if (session) {
      // 有 session 行 = 用户显式设置过：以该行为准
      if (!session.avatarEnabled) return false // 显式 opt-out
      if (session.takeoverUntil && new Date(session.takeoverUntil) > new Date()) return false
      return true // 显式 opt-in
    }
    // 无 session 行 = 跟随全局默认（与后端候选判定 enabled && activate_by_default 对齐）
    return config.value?.activateByDefault ?? false
  }

  // 审批相关方法
  async function applyForApproval() {
    loading.value = true
    error.value = ''
    try {
      config.value = await avatarAPI.applyForApproval()
      return config.value
    } catch (e: any) {
      error.value = e.message || '申请失败'
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
      error.value = e.message || '取消申请失败'
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
    loading,
    error,
    fetchConfig,
    createConfig,
    updateConfig,
    deleteConfig,
    toggleEnabled,
    fetchSessions,
    toggleSession,
    clearSessions,
    takeoverSession,
    getSession,
    isAvatarActive,
    applyForApproval,
    cancelApplication
  }
}
