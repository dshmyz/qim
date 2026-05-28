import type { ApiResponse, DashboardData, RecentRegistration, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export const getDashboardStats = (): Promise<AxiosResponse<ApiResponse<DashboardData>>> => {
  return request({
    url: '/v1/admin/dashboard/stats',
    method: 'get',
  })
}

export const getDashboardTrend = (): Promise<AxiosResponse<ApiResponse<any>>> => {
  return request({
    url: '/v1/admin/dashboard/trend',
    method: 'get',
  })
}

export const getRecentRegistrations = (params?: PaginationParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<RecentRegistration>>>> => {
  return request({
    url: '/v1/admin/recent-registrations',
    method: 'get',
    params,
  })
}

export const getStatistics = (): Promise<AxiosResponse<ApiResponse<any>>> => {
  return request({
    url: '/v1/admin/statistics',
    method: 'get',
  })
}

export const getUserStatistics = (startDate: string, endDate: string): Promise<AxiosResponse<ApiResponse<any>>> => {
  return request({
    url: '/v1/admin/statistics/users',
    method: 'get',
    params: { startDate, endDate },
  })
}

export const getGroupStatistics = (startDate: string, endDate: string): Promise<AxiosResponse<ApiResponse<any>>> => {
  return request({
    url: '/v1/admin/statistics/groups',
    method: 'get',
    params: { startDate, endDate },
  })
}

export const getMessageStatistics = (startDate: string, endDate: string): Promise<AxiosResponse<ApiResponse<any>>> => {
  return request({
    url: '/v1/admin/statistics/messages',
    method: 'get',
    params: { startDate, endDate },
  })
}
