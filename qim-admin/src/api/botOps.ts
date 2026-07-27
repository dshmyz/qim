import type { ApiResponse, PaginationParams } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

// 外部 agent webhook bot（脱敏：无 webhook_secret）
export interface ExternalBot {
  id: number
  name: string
  creator_name: string
  is_active: boolean
  mode: string
  webhook_url: string
  pending_count: number
  dead_count: number
  created_at: string
}

export interface GetExternalBotsParams extends PaginationParams {
  keyword?: string
}

export const getExternalBots = (params: GetExternalBotsParams): Promise<AxiosResponse<ApiResponse<{ list: ExternalBot[]; total: number }>>> => {
  return request({ url: '/v1/admin/bots/external', method: 'get', params })
}

// webhook 投递记录
export interface WebhookDelivery {
  id: number
  bot_id: number
  bot_name?: string
  event: string // bot.message | bot.card_action
  payload: string
  payload_preview?: string
  webhook_url: string
  status: string // pending | done | dead
  attempts: number
  last_error: string
  next_retry_at: string | null
  created_at: string
  updated_at: string
  delivered_at: string | null
}

export interface GetDeliveriesParams extends PaginationParams {
  bot_id?: number
  event?: string
  status?: string
}

export const getWebhookDeliveries = (params: GetDeliveriesParams): Promise<AxiosResponse<ApiResponse<{ list: WebhookDelivery[]; total: number }>>> => {
  return request({ url: '/v1/admin/webhook-deliveries', method: 'get', params })
}

export const getWebhookDelivery = (id: number): Promise<AxiosResponse<ApiResponse<{ delivery: WebhookDelivery; bot_name: string }>>> => {
  return request({ url: `/v1/admin/webhook-deliveries/${id}`, method: 'get' })
}

export interface RedeliverResult {
  status: string
  attempts: number
  last_error: string
  next_retry_at: string | null
}

export const redeliverWebhook = (id: number): Promise<AxiosResponse<ApiResponse<RedeliverResult>>> => {
  return request({ url: `/v1/admin/webhook-deliveries/${id}/redeliver`, method: 'post' })
}
