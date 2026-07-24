import { unref } from 'vue'
import type { Ref } from 'vue'
import { useChatRequest } from './useChatRequest'
import QMessage from '../utils/qmessage'
import { logger } from '../utils/logger'

/**
 * useBotCardAction 封装“提交卡片按钮动作”的 API 调用与结果提示。
 * 让 CardMessage 叶子组件直接调用，省去事件逐层透传（对齐 useMessageReminder 范式）。
 *
 * 用户点击 bot 发来的卡片按钮 -> POST /api/v1/messages/:id/card-action
 * -> 服务端以 event="bot.card_action" 转发到 agent webhook -> agent 回复。
 */
export function useBotCardAction(serverUrl: string | Ref<string>) {
  const { request } = useChatRequest(unref(serverUrl))

  const submitCardAction = async (
    messageId: string,
    actionId: string,
    value?: string
  ): Promise<boolean> => {
    if (!messageId || !actionId) return false
    try {
      const response = await request(`/api/v1/messages/${messageId}/card-action`, {
        method: 'POST',
        body: JSON.stringify({ action_id: actionId, value: value ?? '' })
      })
      if (response.code === 0) {
        return true
      }
      QMessage.error('提交失败: ' + (response.message || '未知错误'))
      return false
    } catch (error) {
      logger.error('提交卡片动作失败:', error)
      QMessage.error('提交失败: ' + (error as Error).message)
      return false
    }
  }

  return { submitCardAction }
}
