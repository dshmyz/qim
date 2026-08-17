// src/composables/useUserToken.ts

import { useRequest } from './useRequest'

export interface UserTokenInfo {
  id: number
  name: string
  created_at: string
  last_used_at: string | null
}

export interface IssuedUserToken {
  token: string
  token_id: number
  name: string
  created_at: string
}

/**
 * 用户长期令牌管理 composable
 * 供 qim CLI / qim-mcp 以本人身份调用用户 API（Authorization: Bearer qusr_...）。
 * 模式镜像 useBotConfig 的 bot 令牌管理。
 */
export function useUserToken() {
  const { get, post, delete: del, isRequesting: loading, lastError: error } = useRequest()

  const listTokens = async (): Promise<UserTokenInfo[]> => {
    const response = await get<any>('/api/v1/user-tokens')
    return response?.data?.tokens || []
  }

  const issueToken = async (name: string): Promise<IssuedUserToken | null> => {
    const response = await post<any>('/api/v1/user-tokens', { name })
    return response?.data || null
  }

  // 返回是否撤销成功：del 经 safeRequest 吞错返回 null，调用方必须据此区分成败，
  // 否则撤销失败也会被当成已撤销（令牌实际仍有效）。
  const revokeToken = async (tokenId: number): Promise<boolean> => {
    const response = await del<any>(`/api/v1/user-tokens/${tokenId}`)
    return response !== null && response?.code === 0
  }

  return { loading, error, listTokens, issueToken, revokeToken }
}