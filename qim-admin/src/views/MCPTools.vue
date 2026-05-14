<template>
  <div class="mcp-tools-page">
    <el-card shadow="never">
      <div class="toolbar">
        <div class="toolbar-left">
          <h2 class="page-title">MCP 工具管理</h2>
          <p class="page-desc">管理 AI 分身可调用的 MCP 工具，启用或禁用特定工具</p>
        </div>
        <div class="toolbar-right">
          <el-button @click="handleRefresh">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </div>

      <!-- 工具统计 -->
      <el-row :gutter="16" style="margin-bottom: 24px">
        <el-col :span="8">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon purple">
                <el-icon :size="32"><Tools /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">总工具数</div>
                <div class="stat-value">{{ tools.length }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="8">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon green">
                <el-icon :size="32"><Check /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">已启用</div>
                <div class="stat-value">{{ enabledCount }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="8">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon orange">
                <el-icon :size="32"><CircleClose /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">已禁用</div>
                <div class="stat-value">{{ disabledCount }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 工具列表 -->
      <el-table :data="tools" v-loading="loading" style="width: 100%">
        <el-table-column prop="name" label="工具名称" min-width="180">
          <template #default="{ row }">
            <div class="tool-name">
              <el-icon class="tool-icon"><Setting /></el-icon>
              <span>{{ formatToolName(row.name) }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="description" label="描述" min-width="300" show-overflow-tooltip />

        <el-table-column label="参数" min-width="200">
          <template #default="{ row }">
            <div class="params-container">
              <el-tag
                v-for="param in getRequiredParams(row.parameters)"
                :key="param"
                size="small"
                type="primary"
                class="param-tag"
              >
                {{ param }}
              </el-tag>
              <span v-if="getRequiredParams(row.parameters).length === 0" class="no-params">
                无必需参数
              </span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="启用状态" width="120" align="center">
          <template #default="{ row }">
            <el-switch
              v-model="row.enabled"
              @change="handleToggleTool(row)"
              :loading="row.loading"
              active-text="启用"
              inactive-text="禁用"
              inline-prompt
            />
          </template>
        </el-table-column>
      </el-table>

      <!-- 空状态 -->
      <el-empty
        v-if="!loading && tools.length === 0"
        description="暂无 MCP 工具"
      >
        <el-button type="primary" @click="handleRefresh">刷新</el-button>
      </el-empty>
    </el-card>

    <!-- 工具详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="selectedTool ? formatToolName(selectedTool.name) : '工具详情'"
      width="600px"
    >
      <div v-if="selectedTool" class="tool-detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="工具名称">
            {{ formatToolName(selectedTool.name) }}
          </el-descriptions-item>
          <el-descriptions-item label="描述">
            {{ selectedTool.description }}
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="selectedTool.enabled ? 'success' : 'danger'" size="small">
              {{ selectedTool.enabled ? '已启用' : '已禁用' }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>

        <div class="params-section">
          <h4>参数列表</h4>
          <el-table :data="formatParameters(selectedTool.parameters)" size="small">
            <el-table-column prop="name" label="参数名" />
            <el-table-column prop="type" label="类型" width="100" />
            <el-table-column prop="required" label="必填" width="80" align="center">
              <template #default="{ row }">
                <el-tag :type="row.required ? 'danger' : 'info'" size="small">
                  {{ row.required ? '是' : '否' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="description" label="说明" />
          </el-table>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Refresh, Tools, Check, CircleClose, Setting } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getMCPTools, updateMCPToolConfig } from '@/api/mcpTools'
import type { MCPTool } from '@/api/mcpTools'

const tools = ref<(MCPTool & { loading?: boolean })[]>([])
const loading = ref(false)
const detailDialogVisible = ref(false)
const selectedTool = ref<MCPTool | null>(null)

const enabledCount = computed(() => tools.value.filter(t => t.enabled).length)
const disabledCount = computed(() => tools.value.filter(t => !t.enabled).length)

const fetchTools = async () => {
  try {
    loading.value = true
    const res = await getMCPTools()
    tools.value = res.data.data.tools.map(tool => ({ ...tool, loading: false }))
  } catch (error) {
    ElMessage.error('获取工具列表失败')
  } finally {
    loading.value = false
  }
}

const handleRefresh = () => {
  fetchTools()
  ElMessage.success('已刷新')
}

const handleToggleTool = async (tool: MCPTool & { loading?: boolean }) => {
  try {
    tool.loading = true
    await updateMCPToolConfig(tool.name, tool.enabled)
    ElMessage.success(`${formatToolName(tool.name)} 已${tool.enabled ? '启用' : '禁用'}`)
  } catch (error) {
    tool.enabled = !tool.enabled
    ElMessage.error('操作失败，请重试')
  } finally {
    tool.loading = false
  }
}

const formatToolName = (name: string) => {
  const nameMap: Record<string, string> = {
    server_monitor: '服务器监控',
    log_analyzer: '日志分析',
    process_manager: '进程管理',
    network_tools: '网络工具',
    user_management: '用户管理',
    group_management: '群组管理',
    system_notification: '系统通知',
    search_knowledge: '知识搜索',
    create_todo: '创建待办',
    search_notes: '笔记搜索',
  }
  return nameMap[name] || name
}

const getRequiredParams = (parameters: Record<string, any>) => {
  if (!parameters || !parameters.properties) return []
  const required = parameters.required || []
  return Object.keys(parameters.properties).filter(
    key => required.includes(key) || parameters.properties[key].required === true
  )
}

const formatParameters = (parameters: Record<string, any>) => {
  if (!parameters || !parameters.properties) return []
  const required = parameters.required || []
  return Object.entries(parameters.properties).map(([key, value]: [string, any]) => ({
    name: key,
    type: value.type || 'string',
    required: required.includes(key) || value.required === true,
    description: value.description || '',
  }))
}

onMounted(() => {
  fetchTools()
})
</script>

<style scoped>
.mcp-tools-page {
  padding: 0;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
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

.stat-card {
  margin-bottom: 0;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.04);
  color: var(--color-text-secondary);
}

.stat-icon.purple {
  background: rgba(168, 85, 247, 0.1);
  color: #a855f7;
}

.stat-icon.green {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.stat-icon.orange {
  background: rgba(249, 115, 22, 0.1);
  color: #f97316;
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
  font-size: 22px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.tool-name {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
}

.tool-icon {
  color: var(--el-color-primary);
}

.params-container {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.param-tag {
  margin: 0;
}

.no-params {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.tool-detail {
  padding: 8px 0;
}

.params-section {
  margin-top: 20px;
}

.params-section h4 {
  margin: 0 0 12px;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}
</style>
