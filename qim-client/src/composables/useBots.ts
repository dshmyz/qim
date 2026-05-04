import { ref } from 'vue'
import { useRequest } from './useRequest'

/**
 * Bot 管理 composable
 * 提供 Bot 的增删改查功能
 */
export function useBots() {
  const { get, post, put, delete: del, isRequesting: loading, lastError: error } = useRequest()
  const botCount = ref(0)

  /**
   * 获取所有 Bot
   */
  const fetchBots = async () => {
    const response = await get<any>('/api/v1/bots')
    return response?.data || []
  }

  /**
   * 获取 Bot 模板
   */
  const fetchTemplates = async () => {
    const response = await get<any>('/api/v1/bots/templates')
    return response?.data || []
  }

  /**
   * 获取当前用户的 Bot
   */
  const fetchMyBots = async () => {
    const response = await get<any>('/api/v1/bots/my')
    return response?.data || []
  }

  /**
   * 获取当前用户的 Bot 数量
   */
  const fetchMyBotCount = async () => {
    const response = await get<any>('/api/v1/bots/my-count')
    if (response?.data?.count !== undefined) {
      botCount.value = response.data.count
    }
    return botCount.value
  }

  /**
   * 创建 Bot
   */
  const createBot = async (data: Record<string, unknown>) => {
    const response = await post<any>('/api/v1/bots', data)
    if (!response) {
      throw new Error(error.value || '创建 Bot 失败')
    }
    return response
  }

  /**
   * 更新 Bot
   */
  const updateBot = async (id: number, data: Record<string, unknown>) => {
    const response = await put<any>(`/api/v1/bots/${id}`, data)
    if (!response) {
      throw new Error(error.value || '更新 Bot 失败')
    }
    return response
  }

  /**
   * 删除 Bot
   */
  const deleteBot = async (id: number) => {
    const response = await del<any>(`/api/v1/bots/${id}`)
    if (!response) {
      throw new Error(error.value || '删除 Bot 失败')
    }
    return response
  }

  return {
    loading,
    error,
    botCount,
    fetchBots,
    fetchTemplates,
    fetchMyBots,
    fetchMyBotCount,
    createBot,
    updateBot,
    deleteBot,
  }
}
