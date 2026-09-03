<template>
  <div class="ai-tools-page">
    <el-card shadow="never">
      <div class="toolbar">
        <div class="toolbar-left">
          <h2 class="page-title">AI 工具管理</h2>
          <p class="page-desc">管理 AI 可调用的工具，启用或禁用特定工具</p>
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
        description="暂无 AI 工具"
      >
        <el-button type="primary" @click="handleRefresh">刷新</el-button>
      </el-empty>
    </el-card>

    <!-- 工具面配置：各 AI 入口的白名单，admin 覆盖后改完即生效 -->
    <el-card shadow="never" style="margin-top: 24px">
      <div class="toolbar" style="margin-bottom: 16px">
        <div class="toolbar-left">
          <h2 class="page-title">工具面配置</h2>
          <p class="page-desc">配置各 AI 入口可调用的工具白名单，保存后即时生效；「恢复默认」清除覆盖配置</p>
        </div>
      </div>

      <el-table :data="scopes" v-loading="scopesLoading" style="width: 100%">
        <el-table-column prop="name" label="入口" width="140">
          <template #default="{ row }">
            <div class="tool-name">
              <span>{{ row.name }}</span>
              <el-tag v-if="row.overridden" size="small" type="warning" style="margin-left: 6px">已覆盖</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="desc" label="说明" min-width="240" show-overflow-tooltip />
        <el-table-column label="生效工具" min-width="380">
          <template #default="{ row }">
            <el-select v-model="row.draft" multiple filterable allow-create default-first-option
              placeholder="选择工具" size="small" style="width: 100%">
              <el-option v-for="t in registeredTools" :key="t" :label="t" :value="t" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" align="center">
          <template #default="{ row }">
            <el-button size="small" type="primary" :loading="row.saving" @click="saveScope(row)">保存</el-button>
            <el-button size="small" :disabled="!row.overridden" @click="resetScope(row)">恢复默认</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 推荐提示词：下发给客户端侧边栏指令条 / bot 会话空态示例 -->
    <el-card shadow="never" style="margin-top: 24px">
      <div class="toolbar" style="margin-bottom: 12px">
        <div class="toolbar-left">
          <h2 class="page-title">推荐提示词</h2>
          <p class="page-desc">下发给客户端的快捷指令（侧边栏指令条 / bot 会话示例）。每行一条，最多 10 条、单条 ≤100 字；清空保存即恢复客户端内置默认</p>
        </div>
      </div>
      <el-input
        v-model="promptsText"
        type="textarea"
        :rows="5"
        placeholder="每行一条，例如：&#10;我有哪些待办任务？&#10;帮我总结今天的日程"
      />
      <div style="margin-top: 10px">
        <el-button type="primary" size="small" :loading="promptsSaving" @click="savePrompts">保存并下发</el-button>
        <el-button size="small" :disabled="!promptsConfigured" @click="resetPrompts">恢复内置默认</el-button>
        <span v-if="promptsConfigured" style="margin-left: 10px; font-size: 12px; color: var(--color-text-secondary)">已自定义</span>
      </div>
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
import { getAITools, updateAIToolConfig, getAIToolScopes, updateAIToolScope, getSuggestedPromptsAdmin, updateSuggestedPromptsAdmin } from '@/api/aiTools'
import type { AITool, AIToolScope } from '@/api/aiTools'

const tools = ref<(AITool & { loading?: boolean })[]>([])
const loading = ref(false)
const detailDialogVisible = ref(false)
const selectedTool = ref<AITool | null>(null)

type ScopeRow = AIToolScope & { draft: string[]; saving?: boolean }
const scopes = ref<ScopeRow[]>([])
const registeredTools = ref<string[]>([])
const scopesLoading = ref(false)

// ── 推荐提示词 ──
const promptsText = ref('')
const promptsSaving = ref(false)
const promptsConfigured = ref(false)

const loadPrompts = async () => {
  try {
    const res = await getSuggestedPromptsAdmin()
    const list = res.data.data?.prompts ?? []
    promptsText.value = list.join('\n')
    promptsConfigured.value = true
  } catch {
    promptsConfigured.value = false
    promptsText.value = ''
  }
}

const savePrompts = async () => {
  const list = promptsText.value.split('\n').map(l => l.trim()).filter(Boolean)
  try {
    promptsSaving.value = true
    await updateSuggestedPromptsAdmin(list)
    promptsConfigured.value = list.length > 0
    promptsText.value = list.join('\n')
    ElMessage.success('推荐提示词已下发，客户端即时生效')
  } catch (error) {
    ElMessage.error('保存失败：' + (error as Error).message)
  } finally {
    promptsSaving.value = false
  }
}

const resetPrompts = async () => {
  try {
    promptsSaving.value = true
    await updateSuggestedPromptsAdmin([])
    promptsText.value = ''
    promptsConfigured.value = false
    ElMessage.success('已恢复客户端内置默认')
  } catch (error) {
    ElMessage.error('恢复失败：' + (error as Error).message)
  } finally {
    promptsSaving.value = false
  }
}

const enabledCount = computed(() => tools.value.filter(t => t.enabled).length)
const disabledCount = computed(() => tools.value.filter(t => !t.enabled).length)

const fetchTools = async () => {
  try {
    loading.value = true
    const res = await getAITools()
    tools.value = res.data.data.tools.map(tool => ({ ...tool, loading: false }))
  } catch (error) {
    ElMessage.error('获取工具列表失败')
  } finally {
    loading.value = false
  }
}

const fetchScopes = async () => {
  try {
    scopesLoading.value = true
    const res = await getAIToolScopes()
    const payload = res.data.data as { scopes: AIToolScope[]; registered_tools: string[] }
    registeredTools.value = payload.registered_tools
    scopes.value = payload.scopes.map(s => ({ ...s, draft: [...s.tools], saving: false }))
  } catch (error) {
    ElMessage.error('获取工具面配置失败')
  } finally {
    scopesLoading.value = false
  }
}

const saveScope = async (row: ScopeRow) => {
  if (row.draft.length === 0) {
    ElMessage.warning('工具列表不能为空（如需恢复默认请点「恢复默认」）')
    return
  }
  try {
    row.saving = true
    const res = await updateAIToolScope(row.scope, { tools: row.draft })
    const data = res.data.data as { tools: string[] }
    row.tools = data.tools
    row.draft = [...data.tools]
    row.overridden = true
    ElMessage.success(`已保存，${row.name} 的工具面即时生效`)
  } catch (error) {
    ElMessage.error('保存失败：' + (error as Error).message)
  } finally {
    row.saving = false
  }
}

const resetScope = async (row: ScopeRow) => {
  try {
    row.saving = true
    const res = await updateAIToolScope(row.scope, { reset: true })
    const data = res.data.data as { tools: string[] }
    row.tools = data.tools
    row.draft = [...data.tools]
    row.overridden = false
    ElMessage.success(`已恢复 ${row.name} 的默认工具面`)
  } catch (error) {
    ElMessage.error('恢复默认失败：' + (error as Error).message)
  } finally {
    row.saving = false
  }
}

const handleRefresh = () => {
  fetchTools()
  fetchScopes()
  ElMessage.success('已刷新')
}

const handleToggleTool = async (tool: AITool & { loading?: boolean }) => {
  try {
    tool.loading = true
    await updateAIToolConfig(tool.name, tool.enabled)
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
  fetchScopes()
  loadPrompts()
})
</script>

<style scoped>
.ai-tools-page {
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
