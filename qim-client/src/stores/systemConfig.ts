import { defineStore } from 'pinia'
import { ref } from 'vue'
import { request } from '../api/core'

export const useSystemConfigStore = defineStore('systemConfig', () => {
  const enableAI = ref(true)
  const enableReadReceipt = ref(true)
  const messageRecallTime = ref(120)
  // 发送提醒触发门槛（秒）：消息发送满该时长才允许提醒。0=禁止提醒。
  const messageRemindTime = ref(3600)
  // 向量库是否可用（服务端运行时注入，决定机器人知识库开关是否生效）
  const vectorEnabled = ref(false)
  const loaded = ref(false)

  async function fetchPublicConfig() {
    try {
      const res = await request<any>('/api/v1/system/public-config')
      const data = res.data ?? res
      if (data.enableAI !== undefined) enableAI.value = data.enableAI
      if (data.enableReadReceipt !== undefined) enableReadReceipt.value = data.enableReadReceipt
      if (data.messageRecallTime !== undefined) messageRecallTime.value = data.messageRecallTime
      if (data.messageRemindTime !== undefined) messageRemindTime.value = data.messageRemindTime
      if (data.vector_enabled !== undefined) vectorEnabled.value = data.vector_enabled === true
      loaded.value = true
    } catch (e) {
      console.warn('获取公开系统配置失败:', e)
    }
  }

  function updateFromServer(data: any) {
    if (data?.enableAI !== undefined) enableAI.value = data.enableAI
    if (data?.enableReadReceipt !== undefined) enableReadReceipt.value = data.enableReadReceipt
    if (data?.messageRecallTime !== undefined) messageRecallTime.value = data.messageRecallTime
    if (data?.messageRemindTime !== undefined) messageRemindTime.value = data.messageRemindTime
    if (data?.vector_enabled !== undefined) vectorEnabled.value = data.vector_enabled === true
  }

  function canRecall(messageCreatedAt: number | string | Date): boolean {
    if (messageRecallTime.value === 0) return false
    const created = new Date(messageCreatedAt).getTime()
    const now = Date.now()
    return (now - created) <= messageRecallTime.value * 1000
  }

  /**
   * canSendReminder 判定一条自发送消息是否可发送提醒。单一事实源：
   * 右键菜单的显示条件与消息上铃铛的显示条件共用此函数，避免两处逻辑漂移。
   * 阈值由系统配置 messageRemindTime 控制（0=禁止提醒），其它规则与撤回/群聊一致。
   */
  function canRemind(message: any, conversationType?: string): boolean {
    if (messageRemindTime.value === 0) return false
    if (!message || !message.timestamp || message.isRead) return false
    // 群聊/讨论组不支持提醒，避免给太多人发提醒
    if (conversationType === 'group' || conversationType === 'discussion') return false
    if (message.sender?.isBot) return false

    const messageTime = new Date(message.timestamp).getTime()
    if (Number.isNaN(messageTime)) return false

    return Date.now() - messageTime > messageRemindTime.value * 1000
  }

  return {
    enableAI,
    enableReadReceipt,
    messageRecallTime,
    messageRemindTime,
    vectorEnabled,
    loaded,
    fetchPublicConfig,
    updateFromServer,
    canRecall,
    canRemind,
  }
})