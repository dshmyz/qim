import type { ApiResponse, Group, PaginationParams, PaginatedResponse, ConversationMember } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export const getGroups = (params: PaginationParams & { keyword?: string }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<Group>>>> => {
  return request({
    url: '/v1/admin/groups',
    method: 'get',
    params,
  })
}

export const getGroupById = (id: number): Promise<AxiosResponse<ApiResponse<Group>>> => {
  return request({
    url: `/v1/groups/${id}`,
    method: 'get',
  })
}

export const getGroupMembers = (groupId: number, params?: PaginationParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<ConversationMember>>>> => {
  return request({
    url: `/v1/groups/${groupId}/members`,
    method: 'get',
    params,
  })
}

export const removeGroupMember = (groupId: number, userId: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/groups/${groupId}/members/${userId}`,
    method: 'delete',
  })
}

export const deleteGroup = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/groups/${id}`,
    method: 'delete',
  })
}
