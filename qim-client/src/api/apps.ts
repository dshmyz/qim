import { http } from './core'
import type { ApiResponse } from '../composables/useRequest'

/**
 * 应用数据结构（与后端 apps 表对齐）
 */
export interface AppItem {
  id: number
  name: string
  icon: string
  category: string
  url: string
  status: string
  open_type?: string
  openType?: string
}

/**
 * 创建/更新应用时的 payload
 * 注意：openType 在前端使用，提交后端时需转换为 open_type
 */
export type AppPayload = Partial<Omit<AppItem, 'id' | 'openType'>> & {
  open_type?: string
}

/**
 * 后端 list 接口返回结构可能是 { list: [...] } 或直接为数组，本封装统一返回数组
 */
function extractApps(data: unknown): AppItem[] {
  if (Array.isArray(data)) return data
  if (data && typeof data === 'object' && Array.isArray((data as any).list)) {
    return (data as any).list
  }
  return []
}

class AppsAPI {
  /**
   * 获取用户应用列表
   */
  async list(): Promise<AppItem[]> {
    const response = await http.get<ApiResponse<unknown>>('/api/v1/apps')
    return extractApps(response.data)
  }

  /**
   * 创建应用
   */
  async create(payload: AppPayload): Promise<AppItem> {
    const response = await http.post<ApiResponse<AppItem>>('/api/v1/apps', payload)
    return response.data
  }

  /**
   * 更新应用
   */
  async update(id: number, payload: AppPayload): Promise<AppItem> {
    const response = await http.put<ApiResponse<AppItem>>(`/api/v1/apps/${id}`, payload)
    return response.data
  }

  /**
   * 删除应用
   */
  async remove(id: number): Promise<void> {
    await http.delete(`/api/v1/apps/${id}`)
  }
}

export const appsApi = new AppsAPI()
