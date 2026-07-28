import { ref } from 'vue'
import { useRequest } from './useRequest'
import type { BotWebhookConfig, BotTokenInfo } from '../types/bot'

/**
 * Bot 配置 composable
 * 管理 bot 的 webhook 模式/地址/密钥配置与访问令牌（签发/列表/撤销）。
 * 镜像 useBots 的 useRequest 模式。
 */
export function useBotConfig() {
  const { get, post, put, delete: del, isRequesting: loading, lastError: error } = useRequest()

  /**
   * 签发新的 bot 访问令牌。明文 token 仅本次返回，需提示用户保存。
   */
  const issueToken = async (botId: number, name: string) => {
    const response = await post<any>(`/api/v1/bots/${botId}/token`, { name })
    return response?.data ?? null
  }

  /**
   * 列出 bot 的访问令牌（不含明文/hash）。
   */
  const listTokens = async (botId: number) => {
    const response = await get<{ tokens: BotTokenInfo[] }>(`/api/v1/bots/${botId}/tokens`)
    return response?.tokens || []
  }

  /**
   * 撤销令牌（软删除即时生效）。
   */
  const revokeToken = async (botId: number, tokenId: number) => {
    const response = await del(`/api/v1/bots/${botId}/token/${tokenId}`)
    if (!response) {
      throw new Error(error.value || '撤销令牌失败')
    }
    return response
  }

  /**
   * 更新 bot 的 webhook 配置（合并到既有 Config，保留未提供字段）。
   */
  const updateConfig = async (botId: number, config: BotWebhookConfig) => {
    const response = await put(`/api/v1/bots/${botId}/config`, config)
    if (!response) {
      throw new Error(error.value || '更新配置失败')
    }
    return response
  }

  return {
    loading,
    error,
    issueToken,
    listTokens,
    revokeToken,
    updateConfig,
  }
}
