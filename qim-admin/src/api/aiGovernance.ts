import { request } from '@/utils/request'

// 治理看板聚合结果（与后端 service.GovernanceMetrics 对应）
export interface SceneGovernance {
  scene: string
  total: number
  completed: number
  interrupted: number
  not_triggered: number
  completion_rate: number
}

export interface NamedCount {
  name: string
  count: number
}

export interface DailySceneCount {
  day: string
  scene: string
  count: number
}

export interface FeedbackGovernance {
  total: number
  up: number
  down: number
  down_reasons: NamedCount[]
}

export interface GovernanceMetrics {
  window_days: number
  generated_at: string
  scenes: SceneGovernance[]
  interrupt_reasons: NamedCount[]
  daily_trend: DailySceneCount[]
  feedback: FeedbackGovernance
  signals: NamedCount[]
}

export function getGovernanceMetrics(days: number): Promise<GovernanceMetrics> {
  return request<GovernanceMetrics>({
    url: `/v1/admin/ai-governance?days=${days}`,
    method: 'get',
  }).then(res => res.data.data)
}

// ---- 自我进化执行器 ----
export interface ModelHealth {
  id: number
  provider: string
  window_calls: number
  completed: number
  interrupted: number
  avg_duration_ms: number
  score: number
  state: 'healthy' | 'degraded'
  reason: string
  updated_at: string
}

export interface EvolutionAction {
  id: number
  kind: string
  target: string
  before: string
  after: string
  basis: string
  state: 'proposed' | 'active' | 'reverted'
  created_at: string
}

export function evaluateEvolution(): Promise<{ actions: EvolutionAction[] }> {
  return request<{ actions: EvolutionAction[] }>({
    url: '/v1/admin/ai-evolution/evaluate',
    method: 'post',
  }).then(res => res.data.data)
}

export function getEvolutionEnabled(): Promise<boolean> {
  return request<{ enabled: boolean }>({ url: '/v1/admin/ai-evolution/enabled' })
    .then(res => res.data.data.enabled)
}

export function setEvolutionEnabled(enabled: boolean): Promise<void> {
  return request({
    url: '/v1/admin/ai-evolution/enabled',
    method: 'put',
    data: { enabled },
  }).then(() => undefined)
}

export function getModelHealth(): Promise<ModelHealth[]> {
  return request<{ health: ModelHealth[] }>({ url: '/v1/admin/ai-evolution/health' })
    .then(res => res.data.data.health)
}

export function getEvolutionActions(): Promise<EvolutionAction[]> {
  return request<{ actions: EvolutionAction[] }>({ url: '/v1/admin/ai-evolution/actions' })
    .then(res => res.data.data.actions)
}

export function revertEvolutionAction(id: number): Promise<void> {
  return request({ url: `/v1/admin/ai-evolution/actions/${id}/revert`, method: 'post' })
    .then(() => undefined)
}

// ---- P4：提示补丁提案式进化 ----
export interface PromptPatch {
  id: number
  category: string
  scene: string
  rule: string
  rationale: string
  case_count: number
  replay_total: number
  replay_passed: number
  state: 'proposed' | 'validated' | 'active' | 'rejected' | 'reverted'
  created_at: string
  updated_at: string
}

export function generatePatches(days = 30): Promise<PromptPatch[]> {
  return request<{ patches: PromptPatch[] }>({
    url: `/v1/admin/ai-patches/generate?days=${days}`,
    method: 'post',
  }).then(res => res.data.data.patches)
}

export function listPatches(state?: string): Promise<PromptPatch[]> {
  return request<{ patches: PromptPatch[] }>({
    url: `/v1/admin/ai-patches${state ? `?state=${state}` : ''}`,
  }).then(res => res.data.data.patches)
}

export function replayPatch(id: number): Promise<PromptPatch> {
  return request<PromptPatch>({ url: `/v1/admin/ai-patches/${id}/replay`, method: 'post' })
    .then(res => res.data.data)
}

export function approvePatch(id: number): Promise<void> {
  return request({ url: `/v1/admin/ai-patches/${id}/approve`, method: 'post' })
    .then(() => undefined)
}

export function deactivatePatch(id: number): Promise<void> {
  return request({ url: `/v1/admin/ai-patches/${id}/deactivate`, method: 'post' })
    .then(() => undefined)
}

