import { request } from '../composables/useRequest'
import type { RenderRule } from '../stores/renderRules'

// 后端测试结果项
export interface TestRuleResult {
  matched: string  // 匹配到的文本
  url: string      // 渲染后的 URL
  label: string    // 渲染后的标签
}

// 管理后台：获取全部规则（含禁用的）
export async function fetchAllRenderRules(): Promise<{ rules: RenderRule[]; version: number }> {
  const res = await request<{ rules: RenderRule[]; version: number }>(
    '/api/v1/admin/render-rules',
    { method: 'GET' }
  )
  return { rules: res.rules ?? [], version: res.version ?? 0 }
}

// 管理后台：批量覆盖保存规则
export async function saveRenderRules(rules: RenderRule[]): Promise<void> {
  await request<{ code: number; message: string }>(
    '/api/v1/admin/render-rules',
    { method: 'PUT', body: JSON.stringify({ rules }) }
  )
}

// 管理后台：测试单条规则在样例文本上的匹配效果
export async function testRenderRule(
  rule: RenderRule,
  sampleText: string
): Promise<TestRuleResult[]> {
  const res = await request<{ results: TestRuleResult[] }>(
    '/api/v1/admin/render-rules/test',
    { method: 'POST', body: JSON.stringify({ rule, sample_text: sampleText }) }
  )
  return res.results ?? []
}
