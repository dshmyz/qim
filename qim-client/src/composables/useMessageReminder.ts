import { unref } from 'vue'
import type { Ref } from 'vue'
import { useChatRequest } from './useChatRequest'
import QMessage from '../utils/qmessage'
import { logger } from '../utils/logger'

/**
 * useMessageReminder 封装"发送消息提醒"的 API 调用与结果提示。
 * 让 UI 层（铃铛、右键菜单）直接调用，省去事件逐层转发，避免漏层 bug。
 * 是否可提醒的判定统一走 useSystemConfigStore().canRemind（单一事实源，阈值来自系统配置）。
 */
export function useMessageReminder(serverUrl: string | Ref<string>) {
  const { request } = useChatRequest(unref(serverUrl))

  const remind = async (message: any): Promise<void> => {
    if (!message) return
    try {
      const response = await request(`/api/v1/messages/${message.id}/remind`, {
        method: 'POST'
      })
      if (response.code === 0) {
        QMessage.info('提醒发送中，结果稍后通知')
      } else {
        QMessage.error('发送提醒失败: ' + (response.message || '未知错误'))
      }
    } catch (error) {
      logger.error('发送提醒失败:', error)
      QMessage.error('发送提醒失败: ' + (error as Error).message)
    }
  }

  return { remind }
}
