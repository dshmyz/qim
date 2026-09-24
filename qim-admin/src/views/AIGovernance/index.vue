<template>
  <div class="ai-governance-page">
    <!-- 顶部：标题 + 窗口切换 -->
    <div class="page-header">
      <div class="header-title">
        <h2>AI 治理看板</h2>
        <span class="header-sub">回复完成率、中断原因与用户反馈分布</span>
      </div>
      <div class="header-actions">
        <el-radio-group v-model="windowDays" size="default" @change="fetchMetrics">
          <el-radio-button :value="7">近 7 天</el-radio-button>
          <el-radio-button :value="14">近 14 天</el-radio-button>
          <el-radio-button :value="30">近 30 天</el-radio-button>
        </el-radio-group>
        <el-button :icon="Refresh" circle @click="fetchMetrics" />
      </div>
    </div>

    <!-- 场景卡片 -->
    <el-row :gutter="16" v-loading="loading">
      <el-col v-for="scene in sceneCards" :key="scene.scene" :xs="12" :sm="12" :md="6">
        <el-card shadow="hover" class="scene-card">
          <div class="scene-head">
            <span class="scene-name">{{ scene.label }}</span>
            <el-icon :size="18" :color="scene.color"><component :is="scene.icon" /></el-icon>
          </div>
          <div class="scene-rate" :style="{ color: rateColor(scene.completion_rate) }">
            {{ (scene.completion_rate * 100).toFixed(0) }}<span class="rate-pct">%</span>
          </div>
          <div class="scene-rate-label">完成率</div>
          <div class="scene-split">
            <span class="split-item completed">完成 {{ scene.completed }}</span>
            <span class="split-item interrupted">中断 {{ scene.interrupted }}</span>
            <span class="split-item skipped">拦截 {{ scene.not_triggered }}</span>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 趋势 + 中断原因 -->
    <el-row :gutter="16" style="margin-top: 16px">
      <el-col :xs="24" :md="15">
        <el-card shadow="never">
          <template #header><span>📈 每日完成回复数</span></template>
          <div ref="trendChartRef" style="height: 300px"></div>
        </el-card>
      </el-col>
      <el-col :xs="24" :md="9">
        <el-card shadow="never" class="h-full">
          <template #header><span>⚠️ 中断原因分布</span></template>
          <PieChart v-if="interruptPie.length" :data="interruptPie" height="300px" />
          <el-empty v-else description="窗口内无中断" :image-size="80" />
        </el-card>
      </el-col>
    </el-row>

    <!-- 反馈 + 学习信号 -->
    <el-row :gutter="16" style="margin-top: 16px">
      <el-col :xs="24" :md="14">
        <el-card shadow="never">
          <template #header><span>👍👎 用户显式反馈</span></template>
          <div class="feedback-overview">
            <div class="feedback-stat up">
              <el-icon :size="22"><Pointer /></el-icon>
              <span class="fb-count">{{ metrics?.feedback.up ?? 0 }}</span>
              <span class="fb-label">好评</span>
            </div>
            <div class="feedback-stat down">
              <el-icon :size="22"><Pointer /></el-icon>
              <span class="fb-count">{{ metrics?.feedback.down ?? 0 }}</span>
              <span class="fb-label">差评</span>
            </div>
          </div>
          <el-divider v-if="metrics?.feedback.down_reasons.length" class="my-2" />
          <div v-if="metrics?.feedback.down_reasons.length" class="down-reasons">
            <div v-for="r in metrics.feedback.down_reasons" :key="r.name" class="reason-row">
              <span class="reason-name">{{ feedbackReasonLabels[r.name] ?? r.name }}</span>
              <div class="reason-bar-wrap">
                <div class="reason-bar" :style="{ width: reasonBarWidth(r.count) }"></div>
              </div>
              <span class="reason-count">{{ r.count }}</span>
            </div>
          </div>
          <el-empty v-else description="窗口内无差评原因数据" :image-size="60" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="10">
        <el-card shadow="never">
          <template #header><span>🧬 学习信号采集</span></template>
          <div v-if="metrics?.signals.length" class="signal-list">
            <div v-for="s in metrics.signals" :key="s.name" class="signal-row">
              <el-icon :size="20" color="#6366f1"><EditPen /></el-icon>
              <span class="signal-name">{{ signalLabels[s.name] ?? s.name }}</span>
              <el-tag type="primary" effect="light" round>{{ s.count }} 条</el-tag>
            </div>
          </div>
          <el-empty v-else description="暂无学习信号" :image-size="70" />
          <el-alert
            class="signal-hint"
            type="info"
            :closable="false"
            title="学习信号是后续 prompt 补丁提案与自我进化的原料"
          />
        </el-card>
      </el-col>
    </el-row>

    <!-- ===== 自我进化执行器 ===== -->
    <el-card shadow="never" style="margin-top: 16px">
      <template #header>
        <div class="evolution-header">
          <span>🧬 自我进化执行器</span>
          <div class="evolution-controls">
            <span class="evolution-mode">
              {{ evolutionEnabled ? '自动模式：系统可在窄域内降权/恢复模型' : '关闭：仅产出建议（dry-run）' }}
            </span>
            <el-switch
              v-model="evolutionEnabled"
              inline-prompt
              active-text="自动"
              inactive-text="关闭"
              @change="onToggleEvolution"
            />
            <el-button type="primary" plain :loading="evaluating" @click="onEvaluate">
              立即评估
            </el-button>
            <el-button :icon="Refresh" circle @click="loadEvolution" />
          </div>
        </div>
      </template>

      <el-row :gutter="16">
        <!-- 模型健康 -->
        <el-col :xs="24" :md="11">
          <h4 class="evolution-col-title">模型健康门控</h4>
          <el-table :data="healthList" size="small" v-if="healthList.length">
            <el-table-column label="Provider" prop="provider" min-width="110" />
            <el-table-column label="完成率" width="78">
              <template #default="{ row }">
                <span :style="{ color: rateColor(row.score) }">{{ (row.score * 100).toFixed(0) }}%</span>
              </template>
            </el-table-column>
            <el-table-column label="样本" prop="window_calls" width="58" />
            <el-table-column label="状态" width="86">
              <template #default="{ row }">
                <el-tag :type="row.state === 'degraded' ? 'danger' : 'success'" size="small" effect="light">
                  {{ row.state === 'degraded' ? '已降权' : '健康' }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-else description="暂无评估数据，点击「立即评估」" :image-size="60" />
        </el-col>

        <!-- 动作审计 -->
        <el-col :xs="24" :md="13">
          <h4 class="evolution-col-title">动作审计（最新在前）</h4>
          <el-table :data="actionList" size="small" max-height="320" v-if="actionList.length">
            <el-table-column label="动作" width="120">
              <template #default="{ row }">
                <el-tag size="small" effect="plain" :type="actionTagType(row.kind)">
                  {{ actionKindLabels[row.kind] ?? row.kind }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="对象" prop="target" width="92" />
            <el-table-column label="依据" prop="basis" min-width="170" show-overflow-tooltip />
            <el-table-column label="状态" width="84">
              <template #default="{ row }">
                <el-tag size="small" :type="actionStateType(row.state)">
                  {{ actionStateLabels[row.state] ?? row.state }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="78">
              <template #default="{ row }">
                <el-button
                  v-if="row.state === 'active'"
                  size="small" link type="danger" @click="onRevert(row)"
                >回滚</el-button>
                <span v-else class="revert-disabled">—</span>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-else description="尚无动作记录" :image-size="60" />
        </el-col>
      </el-row>

      <el-alert
        class="evolution-safety"
        type="warning"
        :closable="false"
        title="安全约束：所有动作留痕可回滚；样本不足（<20）不评估；完成率<70% 降权、≥85% 恢复；关闭时系统只建议不执行。工具权限与凭据永不自动化。"
      />
    </el-card>

    <!-- ===== P4：提示补丁提案 ===== -->
    <el-card shadow="never" style="margin-top: 16px">
      <template #header>
        <div class="evolution-header">
          <span>📚 从失败中学习：提示补丁</span>
          <div class="evolution-controls">
            <el-button type="primary" plain :loading="generating" @click="onGenerate">
              分析失败并生成补丁
            </el-button>
            <el-button :icon="Refresh" circle @click="loadPatches" />
          </div>
        </div>
      </template>

      <el-table :data="patchList" size="default" v-if="patchList.length">
        <el-table-column label="失败类别" width="110">
          <template #default="{ row }">
            {{ feedbackReasonLabels[row.category] ?? row.category }}
          </template>
        </el-table-column>
        <el-table-column label="补丁规则" prop="rule" min-width="240" show-overflow-tooltip />
        <el-table-column label="案例" prop="case_count" width="60" align="center" />
        <el-table-column label="回放" width="90" align="center">
          <template #default="{ row }">
            <span v-if="row.replay_total">
              {{ row.replay_passed }}/{{ row.replay_total }}
            </span>
            <span v-else>—</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="92">
          <template #default="{ row }">
            <el-tag size="small" :type="patchStateType(row.state)">{{ patchStateLabels[row.state] ?? row.state }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" link type="primary"
              v-if="row.state === 'proposed' || row.state === 'rejected'" @click="onReplay(row)"
            >黄金回放</el-button>
            <el-button size="small" link type="success"
              v-if="row.state === 'validated'" @click="onApprove(row)"
            >批准生效</el-button>
            <el-button size="small" link type="danger"
              v-if="row.state === 'active'" @click="onDeactivate(row)"
            >停用</el-button>
            <span v-else-if="row.state === 'reverted'" class="revert-disabled">已停用</span>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-else description="尚无补丁。质量门拒绝的回复会沉淀为失败案例，累计 ≥3 条同类案例后可生成补丁。" :image-size="70" />

      <el-alert
        class="evolution-safety"
        type="success"
        :closable="false"
        title="流程：失败案例聚类 → LLM 提炼规则 → 黄金回放（用补丁重答历史失败案例，通过率≥60% 才可批准）→ 人工批准 → 规则注入 system prompt。补丁可随时停用，全程人工把关。"
      />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import * as echarts from 'echarts'
import { Refresh, ChatDotRound, User, UserFilled, Avatar, Pointer, EditPen } from '@element-plus/icons-vue'
import {
  getGovernanceMetrics, type GovernanceMetrics, type SceneGovernance,
  getModelHealth, getEvolutionActions, getEvolutionEnabled, setEvolutionEnabled,
  evaluateEvolution, revertEvolutionAction,
  generatePatches, listPatches, replayPatch, approvePatch, deactivatePatch,
  type ModelHealth, type EvolutionAction, type PromptPatch,
} from '@/api/aiGovernance'
import PieChart from '@/components/charts/PieChart.vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const metrics = ref<GovernanceMetrics | null>(null)
const loading = ref(false)
const windowDays = ref(7)

// ---- 进化执行器状态 ----
const evolutionEnabled = ref(false)
const evaluating = ref(false)
const healthList = ref<ModelHealth[]>([])
const actionList = ref<EvolutionAction[]>([])

// ---- P4 补丁 ----
const patchList = ref<PromptPatch[]>([])
const generating = ref(false)
const patchStateLabels: Record<string, string> = {
  proposed: '待回放',
  validated: '待批准',
  active: '生效中',
  rejected: '回放未过',
  reverted: '已停用',
}
function patchStateType(state: string): 'info' | 'warning' | 'success' | 'danger' {
  switch (state) {
    case 'active': return 'success'
    case 'validated': return 'warning'
    case 'rejected': return 'danger'
    default: return 'info'
  }
}

const actionKindLabels: Record<string, string> = {
  model_degrade: '降权',
  model_restore: '恢复',
}
const actionStateLabels: Record<string, string> = {
  proposed: '建议',
  active: '已生效',
  reverted: '已回滚',
}
function actionTagType(kind: string): 'danger' | 'success' {
  return kind === 'model_degrade' ? 'danger' : 'success'
}
function actionStateType(state: string): 'info' | 'success' | 'danger' {
  if (state === 'active') return 'success'
  if (state === 'reverted') return 'info'
  return 'danger' // proposed
}

const sceneMeta: Record<string, { label: string; icon: any; color: string }> = {
  bot_dm: { label: 'Bot 单聊', icon: ChatDotRound, color: '#6366f1' },
  group_mention: { label: '群 @AI', icon: User, color: '#0ea5e9' },
  group_auto: { label: '群自动回复', icon: UserFilled, color: '#f59e0b' },
  avatar: { label: '数字分身', icon: Avatar, color: '#10b981' },
}

const sceneCards = computed<(SceneGovernance & { label: string; icon: any; color: string })[]>(() => {
  return (metrics.value?.scenes ?? []).map(s => ({
    ...s,
    label: sceneMeta[s.scene]?.label ?? s.scene,
    icon: sceneMeta[s.scene]?.icon ?? ChatDotRound,
    color: sceneMeta[s.scene]?.color ?? '#6366f1',
  }))
})

const interruptReasonLabels: Record<string, string> = {
  length: '长度截断',
  no_done: '上游静默断连',
  error: '生成错误',
  timeout: '超时',
  quality: '质量门拦截',
}
const interruptPie = computed(() =>
  (metrics.value?.interrupt_reasons ?? []).map(r => ({
    name: interruptReasonLabels[r.name] ?? r.name,
    value: r.count,
  }))
)

const feedbackReasonLabels: Record<string, string> = {
  off_topic: '答非所问',
  hallucinated: '编造信息',
  wrong_trigger: '不该回复',
  incomplete: '没说完',
  factual_error: '事实错误',
  bad_tone: '语气不当',
  other: '其他',
}

const signalLabels: Record<string, string> = {
  draft_edit: '草稿编辑对照',
}

const maxDownReason = computed(() =>
  Math.max(1, ...(metrics.value?.feedback.down_reasons ?? []).map(r => r.count))
)
function reasonBarWidth(v: number): string {
  return `${(v / maxDownReason.value) * 100}%`
}

function rateColor(rate: number): string {
  if (rate >= 0.9) return '#10b981'
  if (rate >= 0.7) return '#f59e0b'
  return '#ef4444'
}

// ---- 趋势图 ----
const trendChartRef = ref<HTMLDivElement>()
let trendChart: echarts.ECharts | null = null

function daysInWindow(): string[] {
  const out: string[] = []
  for (let i = windowDays.value - 1; i >= 0; i--) {
    const d = new Date()
    d.setDate(d.getDate() - i)
    out.push(`${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`)
  }
  return out
}

function renderTrend() {
  if (!trendChartRef.value) return
  if (!trendChart) trendChart = echarts.init(trendChartRef.value)

  const days = daysInWindow()
  const scenes = new Set((metrics.value?.daily_trend ?? []).map(t => t.scene))
  const series = [...scenes].map(scene => {
    const byDay = new Map(
      (metrics.value?.daily_trend ?? []).filter(t => t.scene === scene).map(t => {
        const day = t.day.slice(5) // YYYY-MM-DD -> MM-DD
        return [day, t.count]
      })
    )
    return {
      name: sceneMeta[scene]?.label ?? scene,
      type: 'line' as const,
      smooth: true,
      data: days.map(d => byDay.get(d) ?? 0),
      itemStyle: { color: sceneMeta[scene]?.color ?? '#94a3b8' },
      areaStyle: { opacity: 0.08 },
    }
  })

  trendChart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { top: 0, data: series.map(s => s.name) },
    grid: { left: 36, right: 20, top: 40, bottom: 30 },
    xAxis: { type: 'category', boundaryGap: false, data: days },
    yAxis: { type: 'value', minInterval: 1 },
    series,
  }, true)
}

async function fetchMetrics() {
  loading.value = true
  try {
    metrics.value = await getGovernanceMetrics(windowDays.value)
    await nextTick()
    renderTrend()
  } finally {
    loading.value = false
  }
}

function handleResize() { trendChart?.resize() }

// ---- 进化执行器交互 ----
async function loadEvolution() {
  const [enabled, health, actions] = await Promise.all([
    getEvolutionEnabled(), getModelHealth(), getEvolutionActions(),
  ])
  evolutionEnabled.value = enabled
  healthList.value = health
  actionList.value = actions
}

async function onToggleEvolution(value: boolean | string | number) {
  const enabled = Boolean(value)
  // 开启自动模式前二次确认，避免误操作放权
  if (enabled) {
    try {
      await ElMessageBox.confirm(
        '开启后系统可在窄域内自动降权/恢复模型。所有动作留痕、可一键回滚；工具权限与凭据不会被自动改动。确认开启？',
        '开启自动进化',
        { type: 'warning', confirmButtonText: '确认开启', cancelButtonText: '取消' }
      )
    } catch {
      evolutionEnabled.value = false // 用户取消：复位开关且不调用接口
      return
    }
  }
  try {
    await setEvolutionEnabled(enabled)
    ElMessage.success(enabled ? '已开启自动模式' : '已切换为仅建议（dry-run）')
    await onEvaluate()
  } catch {
    evolutionEnabled.value = !enabled
    ElMessage.error('切换失败')
  }
}

async function onEvaluate() {
  evaluating.value = true
  try {
    await evaluateEvolution()
    await loadEvolution()
    ElMessage.success('评估完成')
  } catch {
    ElMessage.error('评估失败')
  } finally {
    evaluating.value = false
  }
}

async function onRevert(row: EvolutionAction) {
  try {
    await ElMessageBox.confirm(
      `确认回滚针对「${row.target}」的${actionKindLabels[row.kind] ?? row.kind}动作？`,
      '回滚确认',
      { type: 'warning', confirmButtonText: '回滚', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  await revertEvolutionAction(row.id)
  await loadEvolution()
  ElMessage.success('已回滚')
}

// ---- P4 补丁操作 ----
async function loadPatches() {
  patchList.value = await listPatches()
}

async function onGenerate() {
  generating.value = true
  try {
    const created = await generatePatches(30)
    await loadPatches()
    ElMessage.success(created.length ? `已生成 ${created.length} 条补丁提案` : '暂无达到阈值的失败类别')
  } catch {
    ElMessage.error('生成失败')
  } finally {
    generating.value = false
  }
}

async function onReplay(row: PromptPatch) {
  try {
    const result = await replayPatch(row.id)
    await loadPatches()
    if (result.state === 'validated') {
      ElMessage.success(`回放通过（${result.replay_passed}/${result.replay_total}），可批准生效`)
    } else {
      ElMessage.warning(`回放未通过（${result.replay_passed}/${result.replay_total}）`)
    }
  } catch {
    ElMessage.error('回放失败')
  }
}

async function onApprove(row: PromptPatch) {
  try {
    await ElMessageBox.confirm(
      `确认批准并让这条规则生效？\n\n「${row.rule}」\n\n生效后将注入 AI 的 system prompt，可随时停用。`,
      '批准补丁',
      { type: 'warning', confirmButtonText: '批准生效', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  await approvePatch(row.id)
  await loadPatches()
  ElMessage.success('补丁已生效')
}

async function onDeactivate(row: PromptPatch) {
  try {
    await ElMessageBox.confirm('确认停用这条补丁规则？', '停用确认',
      { type: 'warning', confirmButtonText: '停用', cancelButtonText: '取消' })
  } catch {
    return
  }
  await deactivatePatch(row.id)
  await loadPatches()
  ElMessage.success('补丁已停用')
}

onMounted(() => {
  fetchMetrics()
  loadEvolution().catch(() => {})
  loadPatches().catch(() => {})
})
window.addEventListener('resize', handleResize)
onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  trendChart?.dispose()
})
</script>

<style scoped>
.ai-governance-page {
  padding: 4px 4px 24px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 18px;
}

.header-title h2 {
  margin: 0 0 2px;
  font-size: 20px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.header-sub {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

/* ---- 场景卡 ---- */
.scene-card :deep(.el-card__body) {
  padding: 18px;
}

.scene-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.scene-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-regular);
}

.scene-rate {
  margin-top: 10px;
  font-size: 38px;
  font-weight: 700;
  line-height: 1.1;
}

.rate-pct {
  font-size: 18px;
  font-weight: 600;
}

.scene-rate-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.scene-split {
  display: flex;
  gap: 10px;
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.split-item {
  font-size: 12px;
}

.split-item.completed { color: #10b981; }
.split-item.interrupted { color: #ef4444; }
.split-item.skipped { color: var(--el-text-color-secondary); }

/* ---- 反馈 ---- */
.feedback-overview {
  display: flex;
  gap: 32px;
}

.feedback-stat {
  display: flex;
  align-items: center;
  gap: 8px;
}

.feedback-stat.up { color: #10b981; }
.feedback-stat.down { color: #ef4444; }

.fb-count {
  font-size: 26px;
  font-weight: 700;
}

.fb-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.down-reasons {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.reason-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.reason-name {
  width: 72px;
  flex-shrink: 0;
  font-size: 13px;
  color: var(--el-text-color-regular);
}

.reason-bar-wrap {
  flex: 1;
  height: 8px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  overflow: hidden;
}

.reason-bar {
  height: 100%;
  border-radius: 4px;
  background: linear-gradient(90deg, #818cf8, #6366f1);
  transition: width 0.4s ease;
}

.reason-count {
  width: 28px;
  text-align: right;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

/* ---- 学习信号 ---- */
.signal-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-bottom: 14px;
}

.signal-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.signal-name {
  flex: 1;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.signal-hint {
  margin-top: 4px;
}

.h-full :deep(.el-card__body) {
  display: flex;
  flex-direction: column;
}

.my-2 {
  margin: 14px 0;
}

/* ---- 进化执行器 ---- */
.evolution-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.evolution-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.evolution-mode {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.evolution-col-title {
  margin: 4px 0 10px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.revert-disabled {
  color: var(--el-text-color-disabled);
}

.evolution-safety {
  margin-top: 16px;
}
</style>
