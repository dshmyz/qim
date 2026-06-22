<template>
  <div class="knowledge-graph-page">
    <el-card shadow="never">
      <div class="toolbar">
        <div class="toolbar-left">
          <h2 class="page-title">知识图谱可视化</h2>
          <p class="page-desc">展示知识库中的实体关系和知识关联</p>
        </div>
        <div class="toolbar-right">
          <el-button @click="handleRefresh">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </div>

      <!-- 查询表单 -->
      <el-form :inline="true" class="search-form" @submit.prevent="handleSearch">
        <el-form-item label="集合名称">
          <el-input v-model="form.collection" placeholder="如: group_1" style="width: 200px" />
        </el-form-item>
        <el-form-item label="搜索查询">
          <el-input v-model="form.query" placeholder="输入关键词搜索" style="width: 200px" />
        </el-form-item>
        <el-form-item label="最大节点数">
          <el-input-number v-model="form.maxNodes" :min="10" :max="200" style="width: 140px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">查询</el-button>
        </el-form-item>
      </el-form>

      <!-- 图谱容器 -->
      <div class="graph-container" v-loading="loading">
        <div ref="graphRef" class="graph-canvas"></div>
      </div>

      <!-- 统计信息 -->
      <el-row :gutter="16" style="margin-top: 16px">
        <el-col :span="8">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon purple">
                <el-icon :size="28"><Connection /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">节点数</div>
                <div class="stat-value">{{ stats.totalNodes }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="8">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon blue">
                <el-icon :size="28"><Link /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">关系数</div>
                <div class="stat-value">{{ stats.totalEdges }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="8">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon green">
                <el-icon :size="28"><Document /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">知识条目</div>
                <div class="stat-value">{{ stats.knowledgeCount }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </el-card>

    <!-- 节点详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="selectedNode?.label || '节点详情'"
      width="500px"
    >
      <div v-if="selectedNode" class="node-detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="节点 ID">
            {{ selectedNode.id }}
          </el-descriptions-item>
          <el-descriptions-item label="类型">
            <el-tag :type="getNodeTypeColor(selectedNode.type)" size="small">
              {{ getNodeTypeName(selectedNode.type) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="内容" v-if="selectedNode.data?.content">
            {{ selectedNode.data.content }}
          </el-descriptions-item>
          <el-descriptions-item label="分数" v-if="selectedNode.data?.score">
            {{ (selectedNode.data.score * 100).toFixed(1) }}%
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Refresh, Connection, Link, Document } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

interface GraphNode {
  id: string
  label: string
  type: string
  x?: number
  y?: number
  data?: Record<string, any>
}

interface GraphEdge {
  source: string
  target: string
  label: string
  type: string
}

interface GraphData {
  nodes: GraphNode[]
  edges: GraphEdge[]
  total_nodes: number
  total_edges: number
}

type LayoutNode = GraphNode & { x: number; y: number }

const graphRef = ref<HTMLElement | null>(null)
const loading = ref(false)
const detailDialogVisible = ref(false)
const selectedNode = ref<GraphNode | null>(null)

const form = ref({
  collection: 'group_1',
  query: '',
  maxNodes: 50,
})

const stats = ref({
  totalNodes: 0,
  totalEdges: 0,
  knowledgeCount: 0,
})

let canvas: HTMLCanvasElement | null = null
let animationFrameId: number | null = null
let resizeHandler: (() => void) | null = null

const fetchGraphData = async () => {
  try {
    loading.value = true
    const res = await request.get('/v1/admin/knowledge-graph', {
      params: {
        collection: form.value.collection,
        query: form.value.query,
        max_nodes: form.value.maxNodes,
      },
    })
    
    const data: GraphData = res.data.data
    stats.value = {
      totalNodes: data.total_nodes,
      totalEdges: data.total_edges,
      knowledgeCount: data.nodes.filter((n: GraphNode) => n.type === 'knowledge').length,
    }

    renderGraph(data)
  } catch (error) {
    ElMessage.error('获取知识图谱数据失败')
  } finally {
    loading.value = false
  }
}

const renderGraph = (data: GraphData) => {
  if (!graphRef.value) return

  const graphCanvas = document.createElement('canvas')
  canvas = graphCanvas
  graphCanvas.style.width = '100%'
  graphCanvas.style.height = '100%'
  graphRef.value.innerHTML = ''
  graphRef.value.appendChild(graphCanvas)

  const ctx = graphCanvas.getContext('2d')
  if (!ctx) return

  const resizeCanvas = () => {
    const rect = graphRef.value?.getBoundingClientRect()
    if (!rect) return
    graphCanvas.width = rect.width
    graphCanvas.height = rect.height
  }
  resizeCanvas()
  resizeHandler = resizeCanvas
  window.addEventListener('resize', resizeCanvas)

  // 布局节点
  const nodes: LayoutNode[] = data.nodes.map(node => ({
    ...node,
    x: node.x ?? Math.random() * graphCanvas.width * 0.8 + graphCanvas.width * 0.1,
    y: node.y ?? Math.random() * graphCanvas.height * 0.8 + graphCanvas.height * 0.1,
  }))
  const edges = [...data.edges]

  if (nodes.length === 0) return

  // 使用简单力导向布局
  const width = graphCanvas.width
  const height = graphCanvas.height
  
  // 初始位置
  nodes.forEach((node, i) => {
    if (!node.x || !node.y) {
      node.x = Math.random() * width * 0.8 + width * 0.1
      node.y = Math.random() * height * 0.8 + height * 0.1
    }
  })

  let iteration = 0
  const maxIterations = 100

  const layout = () => {
    if (iteration >= maxIterations) {
      draw()
      return
    }

    // 斥力
    for (let i = 0; i < nodes.length; i++) {
      const nodeA = nodes[i]
      if (!nodeA) continue
      for (let j = i + 1; j < nodes.length; j++) {
        const nodeB = nodes[j]
        if (!nodeB) continue
        const dx = nodeB.x - nodeA.x
        const dy = nodeB.y - nodeA.y
        const dist = Math.sqrt(dx * dx + dy * dy) || 1
        const force = 5000 / (dist * dist)
        const fx = (dx / dist) * force
        const fy = (dy / dist) * force
        nodeA.x -= fx
        nodeA.y -= fy
        nodeB.x += fx
        nodeB.y += fy
      }
    }

    // 引力
    edges.forEach(edge => {
      const source = nodes.find(n => n.id === edge.source)
      const target = nodes.find(n => n.id === edge.target)
      if (!source || !target) return

      const dx = target.x - source.x
      const dy = target.y - source.y
      const dist = Math.sqrt(dx * dx + dy * dy) || 1
      const force = dist * 0.01
      const fx = (dx / dist) * force
      const fy = (dy / dist) * force
      source.x += fx
      source.y += fy
      target.x -= fx
      target.y -= fy
    })

    // 中心引力
    nodes.forEach(node => {
      const dx = width / 2 - node.x
      const dy = height / 2 - node.y
      node.x += dx * 0.01
      node.y += dy * 0.01
    })

    iteration++
    layout()
  }

  const draw = () => {
    ctx.clearRect(0, 0, width, height)

    // 绘制边
    edges.forEach(edge => {
      const source = nodes.find(n => n.id === edge.source)
      const target = nodes.find(n => n.id === edge.target)
      if (!source || !target) return

      ctx.beginPath()
      ctx.moveTo(source.x, source.y)
      ctx.lineTo(target.x, target.y)
      ctx.strokeStyle = 'rgba(100, 116, 139, 0.4)'
      ctx.lineWidth = 2
      ctx.stroke()

      // 边标签
      const midX = (source.x + target.x) / 2
      const midY = (source.y + target.y) / 2
      ctx.fillStyle = 'rgba(100, 116, 139, 0.6)'
      ctx.font = '10px sans-serif'
      ctx.fillText(edge.label, midX, midY - 5)
    })

    // 绘制节点
    nodes.forEach(node => {
      const isQuery = node.type === 'query'
      const radius = isQuery ? 30 : 20
      
      ctx.beginPath()
      ctx.arc(node.x, node.y, radius, 0, Math.PI * 2)
      ctx.fillStyle = isQuery ? 'rgba(249, 115, 22, 0.8)' : 'rgba(168, 85, 247, 0.8)'
      ctx.fill()
      ctx.strokeStyle = isQuery ? '#f97316' : '#a855f7'
      ctx.lineWidth = 2
      ctx.stroke()

      // 节点标签
      ctx.fillStyle = '#1e293b'
      ctx.font = '12px sans-serif'
      ctx.textAlign = 'center'
      ctx.fillText(node.label.substring(0, 10), node.x, node.y + 4)
    })
  }

  layout()

  // 点击事件
  graphCanvas.addEventListener('click', (e: MouseEvent) => {
    const rect = graphCanvas.getBoundingClientRect()
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top

    const clickedNode = nodes.find(node => {
      const dx = node.x - x
      const dy = node.y - y
      return Math.sqrt(dx * dx + dy * dy) < 25
    })

    if (clickedNode) {
      selectedNode.value = clickedNode
      detailDialogVisible.value = true
    }
  })
}

const handleSearch = () => {
  fetchGraphData()
}

const handleRefresh = () => {
  fetchGraphData()
  ElMessage.success('已刷新')
}

const getNodeTypeColor = (type: string) => {
  const colors: Record<string, string> = {
    knowledge: 'primary',
    query: 'warning',
    entity: 'success',
  }
  return colors[type] || 'info'
}

const getNodeTypeName = (type: string) => {
  const names: Record<string, string> = {
    knowledge: '知识节点',
    query: '查询节点',
    entity: '实体节点',
  }
  return names[type] || type
}

onMounted(() => {
  fetchGraphData()
})

onUnmounted(() => {
  if (animationFrameId) {
    cancelAnimationFrame(animationFrameId)
  }
  if (resizeHandler) {
    window.removeEventListener('resize', resizeHandler)
    resizeHandler = null
  }
})
</script>

<style scoped>
.knowledge-graph-page {
  padding: 0;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}

.toolbar-left {
  flex: 1;
}

.page-title {
  margin: 0 0 8px;
  font-size: 20px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.page-desc {
  margin: 0;
  font-size: 14px;
  color: var(--color-text-secondary);
}

.toolbar-right {
  display: flex;
  gap: 8px;
}

.search-form {
  margin-bottom: 16px;
  padding: 16px;
  background: var(--color-surface-secondary);
  border-radius: 8px;
}

.graph-container {
  height: 600px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  overflow: hidden;
}

.graph-canvas {
  width: 100%;
  height: 100%;
}

.stat-card {
  margin-bottom: 0;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 12px;
}

.stat-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
}

.stat-icon.purple {
  background: rgba(168, 85, 247, 0.1);
  color: #a855f7;
}

.stat-icon.blue {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.stat-icon.green {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.stat-info {
  flex: 1;
}

.stat-label {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 4px;
}

.stat-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.node-detail {
  padding: 8px 0;
}
</style>
