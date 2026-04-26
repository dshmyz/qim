import type { ApiResponse, Notification, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export interface GetNotificationsParams extends PaginationParams {
  type?: string
  isRead?: boolean
}

export const getNotifications = (params: GetNotificationsParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<Notification>>>> => {
  return request({
    url: '/v1/notifications',
    method: 'get',
    params,
  })
}

export const markAsRead = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/notifications/${id}/read`,
    method: 'put',
  })
}

export const markAllAsRead = (): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: '/v1/notifications/read-all',
    method: 'put',
  })
}

export const deleteNotification = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/notifications/${id}`,
    method: 'delete',
  })
}
