<template>
  <div class="avatar-graph">
    <div class="graph-toolbar">
      <div class="source-tabs">
        <button
          v-for="s in sources"
          :key="s.key"
          class="source-tab"
          :class="{ active: source === s.key }"
          @click="switchSource(s.key)"
        >
          <i :class="s.icon"></i> {{ s.label }}
        </button>
      </div>
      <div v-if="stats.totalNodes" class="graph-stats">
        <span>节点 {{ stats.totalNodes }}</span>
        <span v-if="stats.totalEdges">关系 {{ stats.totalEdges }}</span>
      </div>
      <button class="btn btn-sm" @click="loadGraph">
        <i class="fas fa-sync-alt"></i> 刷新
      </button>
    </div>

    <div class="graph-body">
      <div class="graph-stage">
        <div v-if="loading" class="graph-empty">
          <div class="loading-spinner"></div>
        </div>
        <div v-else-if="stats.totalNodes === 0" class="graph-empty">
          <i class="fas fa-project-diagram"></i>
          <p>{{ emptyText }}</p>
        </div>
        <div v-else class="graph-canvas">
          <EChartsGraph
            :nodes="graphNodes"
            :edges="graphEdges"
            :type-colors="activeTypeColors"
            :show-legend="source === 'memory'"
            :size-by-count="source === 'memory'"
            @node-click="onNodeClick"
            @blank-click="clearDetail"
          />
        </div>
      </div>

      <!-- 选中节点详情卡片（固定右侧，与群聊图谱一致用 slide 过渡） -->
      <transition name="slide">
        <div v-if="selectedNode" class="node-detail">
          <div class="node-detail-head">
            <span class="node-detail-title">{{ selectedNode.label }}</span>
            <span class="node-detail-type badge" :class="[selectedNode.type, source]">{{ typeLabel(selectedNode.type) }}</span>
            <button class="node-detail-close" @click="clearDetail"><i class="fas fa-times"></i></button>
          </div>
          <div class="node-detail-meta" v-if="selectedNode.count">
            出现于 {{ selectedNode.count }} 处 · 相关名词可点击回看
          </div>
          <div class="node-detail-body">
            <template v-if="source === 'memory'">
              <div v-if="relatedMemories.length" class="mem-list">
                <div
                  v-for="(m, i) in relatedMemories"
                  :key="m.id"
                  class="mem-item"
                  :class="{ active: i === activeMemoryIdx }"
                  @click.stop="activeMemoryIdx = i"
                >
                  <i class="fas fa-brain"></i>
                  {{ m.content }}
                </div>
              </div>
              <div v-else class="dim">该名词暂无关联记忆</div>
            </template>
            <template v-else>
              <div class="note-content">{{ selectedNode.data?.content || '无内容预览' }}</div>
            </template>
          </div>
        </div>
      </transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { request } from '../../composables/useRequest'
import EChartsGraph, { type GraphNode as EGNode, type GraphEdge as EGEdge } from '../shared/EChartsGraph.vue'

interface GraphNode {
  id: string
  label?: string
  name?: string
  type: string
  count: number
  x?: number
  y?: number
  data?: Record<string, any>
}

interface GraphEdge {
  source: string
  target: string
  weight: number
}

interface GraphData {
  nodes: GraphNode[]
  edges: GraphEdge[]
  memories?: { id: string; content: string; terms?: string[] }[]
  total_nodes?: number
  total_edges?: number
}

const sources = [
  { key: 'memory', label: '记忆', icon: 'fas fa-brain' },
  { key: 'note', label: '笔记', icon: 'fas fa-sticky-note' },
] as const

const source = ref<'memory' | 'note'>('memory')

const loading = ref(false)
const selectedNode = ref<GraphNode | null>(null)
const relatedMemories = ref<{ id: string; content: string }[]>([])
// 当前图数据携带的原始记忆列表（带 terms），供点节点回查关联记忆
const relatedMemoriesRaw = ref<NonNullable<GraphData['memories']>>([])
const activeMemoryIdx = ref(0)

const stats = ref({ totalNodes: 0, totalEdges: 0 })

// 交给我们统一的知识图谱渲染组件 EChartsGraph 的归一化数据（节点/边）
const graphNodes = ref<EGNode[]>([])
const graphEdges = ref<EGEdge[]>([])

// 记忆来源：实体橙 / 主题紫（带图例，节点按 count 缩放）
const memoryTypeColors: Record<string, [string, string]> = {
  entity: ['#ffd08a', '#f97316'],
  theme:  ['#c4b5fd', '#8b5cf6'],
}
// 笔记来源：笔记片段青 / 实体绿（与群聊图谱 AIGraph 配色一致）
const noteTypeColors: Record<string, [string, string]> = {
  note:     ['#5eead4', '#14b8a6'],
  entity:   ['#86efac', '#16a34a'],
}

const activeTypeColors = computed(() =>
  source.value === 'memory' ? memoryTypeColors : noteTypeColors
)

const emptyText = computed(() =>
  source.value === 'memory'
    ? '暂无记忆图谱。分身记住的重要信息，会从记忆里自动生成实体关系图谱。'
    : '暂无笔记。为分身添加笔记后，这里会显示笔记片段簇。'
)

function switchSource(key: 'memory' | 'note') {
  if (source.value === key) return
  source.value = key
  selectedNode.value = null
  relatedMemories.value = []
  loadGraph()
}

async function loadGraph() {
  loading.value = true
  selectedNode.value = null
  relatedMemories.value = []
  let graph: GraphData | undefined
  try {
    const data = await request<{ code: number; data?: GraphData }>(
      `/api/v1/avatar/knowledge-graph?source=${source.value}&max_nodes=80`,
      { method: 'GET' }
    )
    if (data.code !== 0) {
      renderEmpty()
      return
    }
    const raw = data.data
    graph = (raw ? normalizeGraph(raw) : { nodes: [], edges: [] })
    stats.value = { totalNodes: graph.total_nodes || 0, totalEdges: graph.total_edges || 0 }
    relatedMemoriesRaw.value = graph.memories || []
  } catch (e) {
    console.error('加载分身知识图谱失败', e)
    renderEmpty()
    return
  } finally {
    loading.value = false
  }
  if (graph.nodes && graph.nodes.length) {
    graphNodes.value = graph.nodes as unknown as EGNode[]
    graphEdges.value = graph.edges || []
  } else {
    renderEmpty()
  }
}

function renderEmpty() {
  stats.value = { totalNodes: 0, totalEdges: 0 }
  graphNodes.value = []
  graphEdges.value = []
  relatedMemoriesRaw.value = []
}

// 归一到前端统一 GraphData：memory 来源节点字段名是 name，note 来源是 label，
// 统一成 label 供渲染/点击回查复用，并给 count 兜底默认值。
function normalizeGraph(raw: GraphData): GraphData {
  return {
    nodes: (raw.nodes || []).map(n => ({
      ...n,
      label: n.label ?? n.name ?? '',
      count: n.count ?? 1,
    })),
    edges: raw.edges || [],
    memories: raw.memories,
    total_nodes: raw.total_nodes,
    total_edges: raw.total_edges,
  }
}

function onNodeClick(node: EGNode) {
  const hit = node as unknown as GraphNode
  selectedNode.value = hit
  activeMemoryIdx.value = 0
  if (relatedMemoriesRaw.value.length) {
    const name = String(hit.label ?? '')
    relatedMemories.value = relatedMemoriesRaw.value
      .filter(m => m.terms && m.terms.includes(name))
      .map(m => ({ id: m.id, content: m.content }))
  } else {
    relatedMemories.value = []
  }
}

function clearDetail() {
  selectedNode.value = null
  relatedMemories.value = []
}

function typeLabel(type: string): string {
  if (source.value === 'note') {
    if (type === 'knowledge') return '知识'
    if (type === 'entity') return '实体'
  }
  if (type === 'entity') return '实体'
  if (type === 'theme') return '主题'
  if (type === 'note') return '笔记片段'
  if (type === 'knowledge') return '知识'
  return type
}

onMounted(loadGraph)
</script>

<style scoped>
.avatar-graph { position: relative; min-height: 320px; }
.graph-toolbar { display: flex; justify-content: space-between; align-items: center; gap: 12px; padding: 12px 16px; border-bottom: 1px solid var(--border-color, #eee); flex-wrap: wrap; }
.source-tabs { display: flex; gap: 4px; }
.source-tab { padding: 4px 14px; font-size: var(--font-size-xs); background: transparent; color: var(--text-secondary, #666); border: 1px solid var(--border-color, #ddd); border-radius: 999px; cursor: pointer; }
.source-tab.active { background: var(--primary-color-alpha, rgba(99, 102, 241, 0.12)); color: var(--primary-color, #6366f1); border-color: var(--primary-color, #6366f1); }
.graph-stats { display: flex; gap: 16px; font-size: var(--font-size-xs); color: var(--text-secondary, #666); margin-left: auto; }
.btn { padding: 4px 12px; font-size: var(--font-size-xs); background: var(--card-bg, #f5f5f5); color: var(--text-color, #333); border: 1px solid var(--border-color, #ddd); border-radius: 6px; cursor: pointer; }
.btn:hover { background: var(--primary-color-alpha, rgba(99, 102, 241, 0.08)); }

.graph-body { display: flex; align-items: stretch; }
.graph-stage { flex: 1; min-width: 0; }
.graph-canvas { height: 420px; min-height: 420px; position: relative; }
.graph-canvas :deep(canvas) { display: block; }

.graph-empty { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 320px; color: var(--text-secondary, #999); text-align: center; padding: 24px; }
.graph-empty i { font-size: 36px; margin-bottom: 8px; color: var(--text-secondary, #bbb); }
.graph-empty p { font-size: var(--font-size-xs); max-width: 320px; }

.loading-spinner { width: 32px; height: 32px; border: 3px solid #eee; border-top: 3px solid var(--primary-color); border-radius: 50%; animation: graphspin 0.8s linear infinite; }
@keyframes graphspin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }

/* 右侧详情卡片（与群聊图谱一致：slide 过渡 + z-index 抬高防遮挡） */
.node-detail { position: relative; z-index: 20; width: 240px; flex-shrink: 0; border-left: 1px solid var(--border-color, #eee); background: var(--card-bg, #fff); display: flex; flex-direction: column; max-height: 420px; }
.node-detail-head { display: flex; align-items: center; gap: 8px; padding: 12px 14px; border-bottom: 1px solid var(--border-color, #f0f0f0); }
.node-detail-title { font-size: var(--font-size-sm); font-weight: 600; color: var(--text-color, #333); flex: 1; word-break: break-all; }
.node-detail-close { border: none; background: transparent; color: var(--text-secondary, #999); cursor: pointer; padding: 2px 4px; font-size: var(--font-size-xxs); }
.node-detail-close:hover { color: var(--text-color, #333); }
.badge { font-size: var(--font-size-xxxs); padding: 1px 8px; border-radius: 10px; }
/* 记忆模式 badge：实心色底（保持原有风格） */
.badge.entity.memory { background: #f97316; color: #fff; }
.badge.theme.memory  { background: #8b5cf6; color: #fff; }
.badge.note.memory   { background: #14b8a6; color: #fff; }
/* 笔记模式 badge：半透明底色（与群聊图谱 AIGraph 一致） */
.badge.knowledge    { background: rgba(20, 184, 166, 0.12); color: #0d9488; }
.badge.entity.note  { background: rgba(22, 163, 74, 0.12); color: #15803d; }
/* 通用兜底：非 memory 模式下 entity 走半透明风格 */
.badge.entity       { background: rgba(22, 163, 74, 0.12); color: #15803d; }
.node-detail-meta { font-size: var(--font-size-xxxs); color: var(--text-secondary, #999); padding: 8px 14px 0; }
.node-detail-body { flex: 1; overflow-y: auto; padding: 10px 14px 14px; }
.mem-list { display: flex; flex-direction: column; gap: 6px; }
.mem-item { display: flex; gap: 6px; padding: 6px 8px; border-radius: 6px; cursor: pointer; font-size: var(--font-size-xxs); line-height: 1.5; color: var(--text-color, #333); background: var(--hover-color, #f6f7f9); }
.mem-item i { color: var(--primary-color, #6366f1); margin-top: 2px; }
.mem-item:hover { background: var(--primary-color-alpha, rgba(99, 102, 241, 0.08)); }
.mem-item.active { background: var(--primary-color-alpha, rgba(99, 102, 241, 0.15)); outline: 1px solid var(--primary-color, #6366f1); }
.note-content { font-size: var(--font-size-xxs); line-height: 1.6; color: var(--text-color, #333); word-break: break-all; }
.dim { color: var(--text-secondary, #999); font-size: var(--font-size-xxs); }

.slide-enter-active, .slide-leave-active { transition: max-width 0.18s ease, opacity 0.18s ease; }
.slide-enter-from, .slide-leave-to { max-width: 0; opacity: 0; }
</style>
