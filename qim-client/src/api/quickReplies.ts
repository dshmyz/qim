/**
 * 快速回复 API：对接通用用户设置接口 /api/v1/user-settings/quick_replies
 * 数据存 user_settings 表（setting_key='quick_replies'），value 为 JSON 字符串数组。
 * 用户未自定义时（has_value=false 或空数组）前端返回默认短语。
 * 去重/空串/上限校验在前端完成。
 */

import { request, type ApiResponse } from '../composables/useRequest'

const SETTING_KEY = 'quick_replies'

/** 单用户快速回复条数上限（防止滥用） */
export const MAX_QUICK_REPLIES = 100

/** 用户未自定义时的兜底短语（按使用频率粗排） */
export const DEFAULT_QUICK_REPLIES: string[] = [
  '收到', '好的', '明白', '了解', '没问题',
  '稍后回复', '正在处理', '马上看',
  '谢谢', '感谢', '辛苦了',
  '抱歉', '不好意思', '久等了',
  '请确认', '请查收', '帮忙看一下',
  '好的就这样', '晚点再聊', '先这样',
]

interface UserSettingValue<T> {
  value: T
  has_value: boolean
}

/** 获取当前用户的快速回复短语（未自定义时返回默认短语） */
export async function fetchQuickReplies(): Promise<string[]> {
  const res = await request<ApiResponse<UserSettingValue<string[]>>>(
    `/api/v1/user-settings/${SETTING_KEY}`,
    { method: 'GET' }
  )
  const { value, has_value } = res.data ?? { value: null, has_value: false }
  // 未设置 / 非数组 / 空数组：返回默认值（复制一份，避免调用方误改常量）
  if (!has_value || !Array.isArray(value) || value.length === 0) {
    return [...DEFAULT_QUICK_REPLIES]
  }
  return value
}

/** 保存当前用户的快速回复短语（前端校验 + 整存整取） */
export async function saveQuickReplies(items: string[]): Promise<void> {
  // 去空串、去重（保留首次出现顺序）
  const cleaned: string[] = []
  const seen = new Set<string>()
  for (const r of items) {
    if (!r) continue
    if (seen.has(r)) continue
    seen.add(r)
    cleaned.push(r)
  }
  if (cleaned.length > MAX_QUICK_REPLIES) {
    throw new Error(`快速回复条数不能超过 ${MAX_QUICK_REPLIES} 条`)
  }
  await request<ApiResponse<null>>(
    `/api/v1/user-settings/${SETTING_KEY}`,
    { method: 'PUT', body: JSON.stringify({ value: cleaned }) }
  )
}
