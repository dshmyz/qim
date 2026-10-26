<template>
  <div class="ai-graph">
    <div class="graph-toolbar">
      <div v-if="stats.totalNodes" class="graph-stats">
        <span>节点 {{ shownNodes }}<template v-if="stats.truncated"> / 共 {{ stats.totalNodes }}</template></span>
        <span>关系 {{ shownEdges }}<template v-if="stats.truncated"> / 共 {{ stats.totalEdges }}</template></span>
        <span v-if="stats.truncated" class="graph-hint">已按预算截断</span>
      </div>
      <div class="graph-actions">
        <button class="btn btn-sm" :disabled="rebuilding" @click="rebuildGraph">
          <i class="fas" :class="rebuilding ? 'fa-spinner fa-spin' : 'fa-hammer'"></i>
          {{ rebuilding ? '重建中' : '重建图谱' }}
        </button>
        <button class="btn btn-sm" :disabled="loading" @click="loadGraph">
          <i class="fas" :class="loading ? 'fa-spinner fa-spin' : 'fa-sync-alt'"></i> 刷新
        </button>
      </div>
    </div>

    <div class="graph-body">
      <div class="graph-stage">
        <div v-if="errorMsg" class="graph-error">{{ errorMsg }}</div>
        <div v-if="!errorMsg && !loading && stats.totalNodes === 0" class="graph-empty">
          <i class="fas fa-project-diagram"></i>
          <p>暂无知识图谱。为群知识库绑定并处理文档后，图谱会自动生成。</p>
        </div>
        <!-- v-else-if 而非 v-else：出错时不能同时渲染一块空白画布 -->
        <div v-else-if="!errorMsg" ref="graphRef" class="graph-canvas">
          <!-- 加载中保留画布挂载，避免每次刷新销毁/重建 ECharts 实例 -->
          <EChartsGraph
            v-if="graphNodes.length"
            :nodes="graphNodes"
            :edges="graphEdges"
            :type-colors="typeColors"
            :show-legend="false"
            :max-labeled-nodes="40"
            :max-edges="graphEdgeBudget(graphNodes.length)"
            @node-click="onNodeClick"
            @blank-click="clearDetail"
          />
          <div v-if="loading" class="graph-overlay">
            <div class="loading-spinner"></div>
          </div>
        </div>
      </div>

      <!-- 选中节点详情卡片（固定右侧，与分身影像图谱一致；实体节点展示反查来源） -->
      <transition name="slide">
        <div v-if="selectedNode" class="node-detail">
          <div class="node-detail-head">
            <span class="node-detail-title">{{ selectedNode.label }}</span>
            <span class="node-detail-type badge" :class="selectedNode.type">{{ typeLabel(selectedNode.type) }}</span>
            <button class="node-detail-close" @click="clearDetail"><i class="fas fa-times"></i></button>
          </div>
          <div class="node-detail-body">
            <template v-if="selectedNode.data?.related?.length">
              <div class="node-detail-meta">出现在以下文档：</div>
              <ul class="source-list">
                <li v-for="(r, i) in selectedNode.data.related" :key="i">
                  <i class="fas fa-file-alt"></i> {{ r }}
                </li>
              </ul>
            </template>
            <template v-else>
              <div class="node-detail-meta" v-if="selectedNode.data?.content">
                内容
              </div>
              <div class="node-detail-content" v-if="selectedNode.data?.content">
                {{ selectedNode.data.content }}
              </div>
              <div v-if="!selectedNode.data?.content && !selectedNode.data?.related?.length" class="dim">
                该节点暂无更多信息
              </div>
            </template>
          </div>
        </div>
      </transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import EChartsGraph, { type GraphNode as EGNode, type GraphEdge as EGEdge } from '../../shared/EChartsGraph.vue'
import { graphEdgeBudget } from '@/utils/graph-limits'

interface Props {
  groupId: number
  serverUrl: string
}

const props = defineProps<Props>()

const graphRef = ref<HTMLElement | null>(null)
const loading = ref(false)
const rebuilding = ref(false)
const errorMsg = ref('')
const selectedNode = ref<EGNode | null>(null)

const stats = ref({ totalNodes: 0, totalEdges: 0, truncated: false })

// 归一化后交给统一渲染组件 EChartsGraph 的节点/边；群图谱包含知识文档与实体节点及关系边
const graphNodes = ref<EGNode[]>([])
const graphEdges = ref<EGEdge[]>([])
// 实际渲染数量（与截断前的真实总数区分展示）；边只统计两端都存在且在预算内的部分
const shownNodes = computed(() => graphNodes.value.length)
const shownEdges = computed(() => {
  const ids = new Set(graphNodes.value.map(n => String(n.id)))
  const valid = graphEdges.value.filter(e => ids.has(String(e.source)) && ids.has(String(e.target)))
  return Math.min(valid.length, graphEdgeBudget(graphNodes.value.length))
})
// 群知识图谱：知识文档青 / 实体绿（与 admin 一致），关系边由 EChartsGraph 默认渲染
const typeColors: Record<string, [string, string]> = {
  knowledge: ['#5eead4', '#14b8a6'],      // 青 · 知识文档节点
  entity: ['#86efac', '#16a34a'],          // 绿 · 实体节点
}

const typeLabel = (type: string) => {
  const map: Record<string, string> = {
    knowledge: '知识',
    entity: '实体',
  }
  return map[type] || type
}

const authHeaders = () => ({ 'Authorization': `Bearer ${localStorage.getItem('token')}` })
const REQUEST_TIMEOUT_MS = 15000
// 重建要逐文档重写节点/关系（上限 500 篇），比读图慢得多，给更长的超时但同样要能中止。
const REBUILD_TIMEOUT_MS = 60000

// 图谱构建在服务端是同步的，大群可能较慢：超时主动中止，避免界面无限转圈。
async function fetchGraph(): Promise<void> {
  const ctrl = new AbortController()
  const timer = setTimeout(() => ctrl.abort(), REQUEST_TIMEOUT_MS)
  try {
    const res = await fetch(
      `${props.serverUrl}/api/v1/groups/${props.groupId}/knowledge-graph?max_nodes=80`,
      { headers: authHeaders(), signal: ctrl.signal }
    )
    const data = await res.json()
    if (data.code !== 0) {
      throw new Error(data.message || '加载知识图谱失败')
    }
    const raw = data.data
    stats.value = {
      totalNodes: raw?.total_nodes || 0,
      totalEdges: raw?.total_edges || 0,
      truncated: !!raw?.truncated,
    }
    graphNodes.value = raw?.nodes || []
    graphEdges.value = raw?.edges || []
  } finally {
    clearTimeout(timer)
  }
}

async function loadGraph() {
  if (loading.value) return
  loading.value = true
  errorMsg.value = ''
  clearDetail()
  try {
    await fetchGraph()
  } catch (e) {
    console.error('加载群知识图谱失败', e)
    errorMsg.value = '加载知识图谱失败或超时，请稍后重试'
    renderEmpty()
  } finally {
    loading.value = false
  }
}

// 重建图谱：清理旧版全连接边与孤儿节点。服务端仅允许群主/管理员调用。
async function rebuildGraph() {
  if (rebuilding.value) return
  if (!window.confirm('将按现有文档重建本群知识图谱，并清理旧版冗余关系。确认继续？')) return
  rebuilding.value = true
  errorMsg.value = ''
  const ctrl = new AbortController()
  const timer = setTimeout(() => ctrl.abort(), REBUILD_TIMEOUT_MS)
  try {
    const res = await fetch(
      `${props.serverUrl}/api/v1/groups/${props.groupId}/knowledge-graph/rebuild`,
      { method: 'POST', headers: authHeaders(), signal: ctrl.signal }
    )
    const data = await res.json()
    if (data.code !== 0) {
      throw new Error(data.message || '重建失败')
    }
    await loadGraph()
  } catch (e) {
    console.error('重建群知识图谱失败', e)
    errorMsg.value = '重建图谱失败或超时，请稍后重试'
  } finally {
    clearTimeout(timer)
    rebuilding.value = false
  }
}

function renderEmpty() {
  stats.value = { totalNodes: 0, totalEdges: 0, truncated: false }
  graphNodes.value = []
  graphEdges.value = []
}

function onNodeClick(node: EGNode) {
  selectedNode.value = node
}

function clearDetail() {
  selectedNode.value = null
}

onMounted(loadGraph)
</script>

<style scoped>
.ai-graph { position: relative; min-height: 320px; }
.graph-toolbar { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; border-bottom: 1px solid var(--border-color, #eee); }
.graph-stats { display: flex; gap: 16px; font-size: var(--font-size-xs); color: var(--text-secondary, #666); }
.graph-actions { display: flex; gap: 8px; }
.graph-hint { color: var(--warning-color, #d97706); }
.graph-error { padding: 8px 16px; font-size: var(--font-size-xs); color: var(--danger-color, #dc2626); }
.graph-overlay { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; background: var(--card-bg, rgba(255,255,255,0.72)); z-index: 10; }
.btn:disabled { opacity: 0.6; cursor: default; }
.btn { padding: 4px 12px; font-size: var(--font-size-xs); background: var(--card-bg, #f5f5f5); color: var(--text-color, #333); border: 1px solid var(--border-color, #ddd); border-radius: 6px; cursor: pointer; }
.btn:hover { background: var(--primary-color-alpha, rgba(99, 102, 241, 0.08)); }
.graph-body { display: flex; align-items: stretch; }
.graph-stage { flex: 1; min-width: 0; }
.graph-canvas { height: 420px; position: relative; }
.graph-empty { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 320px; color: var(--text-secondary, #999); text-align: center; padding: 24px; }
.graph-empty i { font-size: 36px; margin-bottom: 8px; color: var(--text-secondary, #bbb); }
.graph-empty p { font-size: var(--font-size-xs); }
.loading-spinner { width: 32px; height: 32px; border: 3px solid #eee; border-top: 3px solid var(--primary-color); border-radius: 50%; animation: graphspin 0.8s linear infinite; }
@keyframes graphspin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }

/* 右侧固定详情卡片（对齐分身影像图谱）；显式提升到拓扑画布之上，避免被图谱层级压住/数据重叠 */
.node-detail { position: relative; z-index: 20; width: 240px; flex-shrink: 0; border-left: 1px solid var(--border-color, #eee); background: var(--card-bg, #fff); display: flex; flex-direction: column; max-height: 420px; }
.node-detail-head { display: flex; align-items: center; gap: 8px; padding: 12px 14px; border-bottom: 1px solid var(--border-color, #f0f0f0); }
.node-detail-title { font-size: var(--font-size-sm); font-weight: 600; color: var(--text-color, #333); flex: 1; word-break: break-all; }
.node-detail-close { border: none; background: transparent; color: var(--text-secondary, #999); cursor: pointer; padding: 2px 4px; font-size: var(--font-size-xxs); }
.node-detail-close:hover { color: var(--text-color, #333); }
.node-detail-meta { font-size: var(--font-size-xxs); color: var(--text-secondary, #666); padding: 10px 14px 2px; }
.node-detail-body { flex: 1; overflow-y: auto; padding: 0 14px 14px; }
.node-detail-content { font-size: var(--font-size-xxs); color: var(--text-secondary, #666); line-height: 1.5; word-break: break-all; margin-top: 4px; }
.badge { font-size: var(--font-size-xxxs); padding: 1px 8px; border-radius: 10px; }
.badge.knowledge { background: rgba(20, 184, 166, 0.12); color: #0d9488; }
.badge.entity { background: rgba(22, 163, 74, 0.12); color: #15803d; }
.source-list { list-style: none; margin: 6px 0 0; padding: 0; }
.source-list li { display: flex; align-items: flex-start; gap: 6px; font-size: var(--font-size-xxs); color: var(--primary-color, #6366f1); line-height: 1.5; padding: 4px 0; margin-bottom: 4px; border-bottom: 1px dashed var(--border-color, #eee); }
.source-list li i { margin-top: 2px; color: var(--text-secondary, #999); }
.dim { font-size: var(--font-size-xxs); color: var(--text-secondary, #999); padding: 10px 14px; }
.slide-enter-active, .slide-leave-active { transition: max-width 0.18s ease, opacity 0.18s ease; }
</style>
