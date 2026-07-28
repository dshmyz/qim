import { unref } from 'vue'
import type { Ref } from 'vue'
import { useChatRequest } from './useChatRequest'
import QMessage from '../utils/qmessage'
import { logger } from '../utils/logger'

/** 卡片动作提交结果。alreadyHandled=true 表示后端命中幂等（已处理过），不重复触发 webhook。 */
export interface CardActionResult {
  ok: boolean
  alreadyHandled: boolean
}

/**
 * useBotCardAction 封装“提交卡片按钮动作”的 API 调用与结果提示。
 * 让 CardMessage 叶子组件直接调用，省去事件逐层透传（对齐 useMessageReminder 范式）。
 *
 * 用户点击 bot 发来的卡片按钮 -> POST /api/v1/messages/:id/card-action
 * -> 服务端以 event="bot.card_action" 转发到 agent webhook -> agent 回复。
 *
 * 幂等：同一卡片+同一用户后端只处理一次。重复点击返回 alreadyHandled=true，
 * 前端据此给“已处理”提示而非静默成功（兼顾清缓存/换设备/直连后端的场景）。
 */
export function useBotCardAction(serverUrl: string | Ref<string>) {
  const { request } = useChatRequest(unref(serverUrl))

  const submitCardAction = async (
    messageId: string,
    actionId: string,
    value?: string
  ): Promise<CardActionResult> => {
    if (!messageId || !actionId) return { ok: false, alreadyHandled: false }
    try {
      const response = await request(`/api/v1/messages/${messageId}/card-action`, {
        method: 'POST',
        body: JSON.stringify({ action_id: actionId, value: value ?? '' })
      })
      if (response.code === 0) {
        const alreadyHandled = !!response.data?.already_handled
        if (alreadyHandled) {
          QMessage.info(response.message || '该卡片已处理')
        }
        return { ok: true, alreadyHandled }
      }
      QMessage.error('提交失败: ' + (response.message || '未知错误'))
      return { ok: false, alreadyHandled: false }
    } catch (error) {
      logger.error('提交卡片动作失败:', error)
      QMessage.error('提交失败: ' + (error as Error).message)
      return { ok: false, alreadyHandled: false }
    }
  }

  return { submitCardAction }
}
