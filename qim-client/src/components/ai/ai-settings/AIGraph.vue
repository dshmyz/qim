<template>
  <div class="ai-graph">
    <div class="graph-toolbar">
      <div v-if="stats.totalNodes" class="graph-stats">
        <span>节点 {{ stats.totalNodes }}</span>
        <span>关系 {{ stats.totalEdges }}</span>
      </div>
      <button class="btn btn-sm" @click="loadGraph">
        <i class="fas fa-sync-alt"></i> 刷新
      </button>
    </div>

    <div v-if="loading" class="graph-empty">
      <div class="loading-spinner"></div>
    </div>
    <div v-else-if="stats.totalNodes === 0" class="graph-empty">
      <i class="fas fa-project-diagram"></i>
      <p>暂无知识图谱。为群知识库绑定并处理文档后，图谱会自动生成。</p>
    </div>
    <div v-else ref="graphRef" class="graph-canvas">
      <EChartsGraph
        :nodes="graphNodes"
        :edges="graphEdges"
        :type-colors="typeColors"
        :show-legend="false"
        @node-click="onNodeClick"
        @blank-click="clearTooltip"
      />
    </div>

    <!-- 节点详情（浮动 tooltip） -->
    <div v-if="selectedNode" class="node-tooltip" :style="tooltipStyle">
      <div class="node-tooltip-title">{{ selectedNode.label }}</div>
      <div class="node-tooltip-content">{{ selectedNode.data?.content }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import EChartsGraph, { type GraphNode as EGNode, type GraphEdge as EGEdge } from '../../shared/EChartsGraph.vue'

interface Props {
  groupId: number
  serverUrl: string
}

const props = defineProps<Props>()

const graphRef = ref<HTMLElement | null>(null)
const loading = ref(false)
const selectedNode = ref<EGNode | null>(null)
const tooltipStyle = ref<Record<string, string>>({})

const stats = ref({ totalNodes: 0, totalEdges: 0 })

// 归一化后交给统一渲染组件 EChartsGraph 的节点/边；群图谱只有知识节点、无边
const graphNodes = ref<EGNode[]>([])
const graphEdges = ref<EGEdge[]>([])
// 群知识节点：统一青色渐变（由 EChartsGraph 默认配色兜底，这里显式声明更清晰）
const typeColors: Record<string, [string, string]> = {
  knowledge: ['#5eead4', '#14b8a6'], // 青 · 知识节点
}

async function loadGraph() {
  loading.value = true
  selectedNode.value = null
  tooltipStyle.value = {}
  let graph: { nodes: EGNode[]; edges: EGEdge[]; total_nodes?: number; total_edges?: number } | undefined
  try {
    const res = await fetch(
      `${props.serverUrl}/api/v1/groups/${props.groupId}/knowledge-graph?max_nodes=80`,
      { headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` } }
    )
    const data = await res.json()
    if (data.code !== 0) {
      renderEmpty()
      return
    }
    const raw = data.data
    graph = raw || { nodes: [], edges: [] }
    stats.value = { totalNodes: raw?.total_nodes || 0, totalEdges: raw?.total_edges || 0 }
  } catch (e) {
    console.error('加载群知识图谱失败', e)
    renderEmpty()
    return
  } finally {
    loading.value = false
  }
  if (graph && graph.nodes && graph.nodes.length) {
    graphNodes.value = graph.nodes
    graphEdges.value = graph.edges || []
  } else {
    renderEmpty()
  }
}

function renderEmpty() {
  stats.value = { totalNodes: 0, totalEdges: 0 }
  graphNodes.value = []
  graphEdges.value = []
}

function onNodeClick(node: EGNode) {
  selectedNode.value = node
  // 浮动 tooltip 锚定在图右上角，展示该知识节点内容（不遮挡节点本身）
  tooltipStyle.value = { right: '12px', top: '12px' }
}

function clearTooltip() {
  selectedNode.value = null
  tooltipStyle.value = {}
}

onMounted(loadGraph)
</script>

<style scoped>
.ai-graph { position: relative; min-height: 320px; }
.graph-toolbar { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; border-bottom: 1px solid var(--border-color, #eee); }
.graph-stats { display: flex; gap: 16px; font-size: 13px; color: var(--text-secondary, #666); }
.btn { padding: 4px 12px; font-size: 13px; background: var(--card-bg, #f5f5f5); color: var(--text-color, #333); border: 1px solid var(--border-color, #ddd); border-radius: 6px; cursor: pointer; }
.btn:hover { background: var(--primary-color-alpha, rgba(99, 102, 241, 0.08)); }
.graph-canvas { height: 420px; position: relative; }
.graph-empty { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 320px; color: var(--text-secondary, #999); text-align: center; padding: 24px; }
.graph-empty i { font-size: 36px; margin-bottom: 8px; color: var(--text-secondary, #bbb); }
.graph-empty p { font-size: 13px; }
.loading-spinner { width: 32px; height: 32px; border: 3px solid #eee; border-top: 3px solid var(--primary-color); border-radius: 50%; animation: graphspin 0.8s linear infinite; }
@keyframes graphspin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }
.node-tooltip { position: absolute; z-index: 20; max-width: 260px; background: #fff; border: 1px solid var(--border-color, #ddd); border-radius: 8px; box-shadow: 0 4px 16px rgba(0,0,0,0.15); padding: 10px 12px; pointer-events: none; }
.node-tooltip-title { font-size: 13px; font-weight: 600; color: var(--text-color, #333); margin-bottom: 4px; }
.node-tooltip-content { font-size: 12px; color: var(--text-secondary, #666); line-height: 1.5; word-break: break-all; max-height: 120px; overflow: hidden; }
</style>
