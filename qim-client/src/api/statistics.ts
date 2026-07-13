import { http } from './core'
import type { ApiResponse } from '../composables/useRequest'

/**
 * 统计周期类型
 */
export type StatisticsPeriod = 'day' | 'week' | 'month' | 'year'

/**
 * 统计数据结构（与后端返回对齐）
 */
export interface StatisticsData {
  totalMessages?: number
  totalFiles?: number
  totalNotes?: number
  totalTasks?: number
  completedTasks?: number
  pendingTasks?: number
  taskCompletionRate?: number
  maxMessages?: number
  messageTrend?: Array<Record<string, any>>
  fileTypes?: Array<Record<string, any>>
  [key: string]: any
}

class StatisticsAPI {
  /**
   * 获取统计数据
   * @param period 统计周期（day/week/month/year）
   */
  async get(period: StatisticsPeriod): Promise<StatisticsData> {
    const response = await http.get<ApiResponse<StatisticsData>>(
      '/api/v1/statistics',
      { params: { period } }
    )
    return response.data
  }
}

export const statisticsApi = new StatisticsAPI()
