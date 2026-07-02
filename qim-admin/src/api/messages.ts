import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'
import type { ApiResponse } from '@/types'
import type { MessageSearchParams, MessageSearchResult } from '@/types/message'

export function searchMessages(params: MessageSearchParams): Promise<AxiosResponse<ApiResponse<MessageSearchResult>>> {
  return request({
    url: '/v1/admin/messages/search',
    method: 'get',
    params,
  })
}
