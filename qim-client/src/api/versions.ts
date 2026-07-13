import { http } from './core'
import type { ApiResponse } from '../composables/useRequest'

/**
 * 客户端版本信息（与后端 client_versions 表对齐）
 */
export interface ClientVersion {
  id: number
  version: string
  releaseDate?: string
  updateNotes?: string
  downloadUrl?: string
  [key: string]: any
}

/**
 * 版本列表响应结构
 */
interface VersionListResponse {
  list: ClientVersion[]
}

class VersionsAPI {
  /**
   * 获取指定平台的最新版本
   *
   * 该接口为公开接口（无需鉴权），使用 8 秒超时
   * 以匹配"检查更新"场景的快速反馈需求。
   *
   * @param platform 平台标识（macos/windows/linux）
   * @returns 最新版本信息；无版本记录时返回 null
   */
  async getLatest(platform: string): Promise<ClientVersion | null> {
    const response = await http.get<ApiResponse<VersionListResponse>>(
      '/api/v1/client/versions',
      {
        params: { platform, pageSize: 1 },
        skipAuth: true,
        timeout: 8000,
      }
    )

    const list = response.data?.list
    if (!list || list.length === 0) return null
    return list[0]
  }
}

export const versionsApi = new VersionsAPI()
