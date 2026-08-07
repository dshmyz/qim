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
        <el-form-item label="选择群">
          <el-select
            v-model="selectedGroupId"
            placeholder="选择群（自动填充集合名）"
            filterable
            clearable
            style="width: 220px"
            @change="onGroupSelect"
          >
            <el-option
              v-for="g in groups"
              :key="g.id"
              :label="g.name"
              :value="g.id"
            />
          </el-select>
        </el-form-item>
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
        <div class="graph-canvas">
          <EChartsGraph
            :nodes="graphNodes"
            :edges="graphEdges"
            :type-colors="typeColors"
            :show-legend="true"
            :show-edge-label="true"
            @node-click="onNodeClick"
            @blank-click="clearDetail"
          />
        </div>
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
          <el-descriptions-item label="来源" v-if="selectedNode.data?.related?.length">
            <div class="node-source-list">
              <div v-for="(r, i) in selectedNode.data.related" :key="i" class="node-source-item">
                <el-icon><Document /></el-icon>
                <span>{{ r }}</span>
              </div>
            </div>
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
import { ref, onMounted } from 'vue'
import { Refresh, Connection, Link, Document } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'
import { getGroups } from '@/api/groups'
import type { Group } from '@/types'
import EChartsGraph, { type GraphNode as EGNode, type GraphEdge as EGEdge } from '@/components/charts/EChartsGraph.vue'

interface GraphData {
  nodes: EGNode[]
  edges: EGEdge[]
  total_nodes: number
  total_edges: number
}

const loading = ref(false)
const detailDialogVisible = ref(false)
const selectedNode = ref<EGNode | null>(null)

const form = ref({
  collection: 'group_1',
  query: '',
  maxNodes: 50,
})

// 群下拉：选中群后自动填充集合名 group_{id}，仍可手动微调
const groups = ref<Group[]>([])
const selectedGroupId = ref<number | undefined>(undefined)
const fetchGroups = async () => {
  try {
    const res = await getGroups({ page: 1, pageSize: 100 })
    groups.value = res.data?.data?.list || []
  } catch (e) {
    console.error('加载群列表失败', e)
  }
}
const onGroupSelect = (id: number | undefined) => {
  if (id != null) {
    form.value.collection = `group_${id}`
  }
}

const stats = ref({
  totalNodes: 0,
  totalEdges: 0,
  knowledgeCount: 0,
})

// 归一化后交给统一渲染组件 EChartsGraph 的节点/边
const graphNodes = ref<EGNode[]>([])
const graphEdges = ref<EGEdge[]>([])
// 管理后台配色：查询橙 / 知识紫 / 实体绿
const typeColors: Record<string, [string, string]> = {
  query: ['#ffd08a', '#f97316'],      // 橙 · 查询
  knowledge: ['#c4b5fd', '#8b5cf6'],  // 紫 · 知识
  entity: ['#86efac', '#16a34a'],     // 绿 · 实体
}

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
      knowledgeCount: data.nodes.filter((n: EGNode) => n.type === 'knowledge').length,
    }

    graphNodes.value = data.nodes
    graphEdges.value = data.edges || []
  } catch (error) {
    ElMessage.error('获取知识图谱数据失败')
  } finally {
    loading.value = false
  }
}

// EChartsGraph 冒泡的节点点击 → 弹出详情弹窗
const onNodeClick = (node: EGNode) => {
  selectedNode.value = node
  detailDialogVisible.value = true
}

const clearDetail = () => {
  selectedNode.value = null
  detailDialogVisible.value = false
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
  fetchGroups()
  fetchGraphData()
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
.node-source-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.node-source-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--el-color-primary);
  line-height: 1.5;
  padding: 3px 0;
  border-bottom: 1px dashed var(--el-border-color-lighter);
}
.node-source-item .el-icon {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
}
</style>
