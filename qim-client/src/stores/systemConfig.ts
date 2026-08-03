import { defineStore } from 'pinia'
import { ref } from 'vue'
import { request } from '../api/core'

export const useSystemConfigStore = defineStore('systemConfig', () => {
  const enableAI = ref(true)
  const enableReadReceipt = ref(true)
  const messageRecallTime = ref(120)
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
    if (data?.vector_enabled !== undefined) vectorEnabled.value = data.vector_enabled === true
  }

  function canRecall(messageCreatedAt: number | string | Date): boolean {
    if (messageRecallTime.value === 0) return false
    const created = new Date(messageCreatedAt).getTime()
    const now = Date.now()
    return (now - created) <= messageRecallTime.value * 1000
  }

  return {
    enableAI,
    enableReadReceipt,
    messageRecallTime,
    vectorEnabled,
    loaded,
    fetchPublicConfig,
    updateFromServer,
    canRecall,
  }
})