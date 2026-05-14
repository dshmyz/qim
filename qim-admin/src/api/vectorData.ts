import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'
import type { ApiResponse } from '@/types'

export interface CollectionInfo {
  name: string
  count: number
}

export interface VectorEntry {
  doc_id: string
  content: string
  metadata: Record<string, string>
}

export interface CollectionData {
  collection: string
  entries: VectorEntry[]
  total: number
}

export function listCollections(): Promise<AxiosResponse<ApiResponse<CollectionInfo[]>>> {
  return request({
    url: '/v1/admin/vector/collections',
    method: 'get',
  })
}

export function getCollectionData(name: string): Promise<AxiosResponse<ApiResponse<CollectionData>>> {
  return request({
    url: `/v1/admin/vector/collections/${name}`,
    method: 'get',
  })
}