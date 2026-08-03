import type { ApiResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

// 渲染规则数据结构（与后端 service.RenderRule 一致）
export interface RenderRule {
  id: string
  name: string
  enabled: boolean
  priority: number
  scope: {
    groups: string[]
    exclude_groups: string[]
    conversation_types: string[]
  }
  match: {
    pattern: string
    flags: string
    capture_groups: Record<string, number>
  }
  render: {
    type: 'link' | 'link_card' | 'text_chip'
    url_template: string
    label_template: string
    title_template?: string
    icon?: string
    target?: '_blank' | '_self'
    class?: string
  }
}

export interface RenderRulesResponse {
  rules: RenderRule[]
  version: number
}

export interface TestRuleResult {
  matched: string
  url: string
  label: string
}

// 获取全部规则（含禁用的）
export const getRenderRules = (): Promise<AxiosResponse<ApiResponse<RenderRulesResponse>>> => {
  return request({
    url: '/v1/admin/render-rules',
    method: 'get',
  })
}

// 批量保存规则（覆盖式）
export const saveRenderRules = (rules: RenderRule[]): Promise<AxiosResponse<ApiResponse<null>>> => {
  return request({
    url: '/v1/admin/render-rules',
    method: 'put',
    data: { rules },
  })
}

// 测试单条规则在样例文本上的匹配效果
export const testRenderRule = (
  rule: RenderRule,
  sampleText: string
): Promise<AxiosResponse<ApiResponse<{ results: TestRuleResult[] }>>> => {
  return request({
    url: '/v1/admin/render-rules/test',
    method: 'post',
    data: { rule, sample_text: sampleText },
  })
}
