import { unref } from 'vue'
import type { Ref } from 'vue'
import { useChatRequest } from './useChatRequest'
import QMessage from '../utils/qmessage'
import { logger } from '../utils/logger'

/**
 * canRemind 判定一条自发送消息是否可发送提醒。
 * 单一事实源：右键菜单的显示条件与消息上铃铛的显示条件共用此函数，避免两处逻辑漂移。
 */
export function canRemind(message: any, conversationType?: string): boolean {
  if (!message || !message.timestamp || message.isRead) return false
  // 群聊/讨论组不支持提醒，避免给太多人发提醒
  if (conversationType === 'group' || conversationType === 'discussion') return false
  if (message.sender?.isBot) return false

  const messageTime = new Date(message.timestamp).getTime()
  if (Number.isNaN(messageTime)) return false

  return Date.now() - messageTime > 60 * 60 * 1000
}

/**
 * useMessageReminder 封装"发送消息提醒"的 API 调用与结果提示。
 * 让 UI 层（铃铛、右键菜单）直接调用，省去事件逐层转发，避免漏层 bug。
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
