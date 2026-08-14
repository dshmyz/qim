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

    <div class="graph-body">
      <div class="graph-stage">
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
            @blank-click="clearDetail"
          />
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

const stats = ref({ totalNodes: 0, totalEdges: 0 })

// 归一化后交给统一渲染组件 EChartsGraph 的节点/边；群图谱包含知识文档与实体节点及关系边
const graphNodes = ref<EGNode[]>([])
const graphEdges = ref<EGEdge[]>([])
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

async function loadGraph() {
  loading.value = true
  clearDetail()
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
