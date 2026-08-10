<template>
  <div class="providers-page">
    <el-tabs v-model="activeTab" class="config-tabs">
      <!-- AI 供应商 -->
      <el-tab-pane name="providers">
        <template #label>
          <span class="tab-label"><el-icon><Cpu /></el-icon>AI 供应商</span>
        </template>
        <el-card shadow="never">
          <!-- 工具栏 -->
          <div class="toolbar">
            <div class="toolbar-left">
              <h2 class="page-title">AI 模型提供商管理</h2>
              <p class="page-desc">管理 AI 模型提供商配置，包括 API 端点、密钥、可用模型等</p>
            </div>
            <div class="toolbar-right">
              <el-button type="primary" @click="handleCreate">
                <el-icon><Plus /></el-icon>
                添加提供商
              </el-button>
              <el-button @click="handleRefresh">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>
            </div>
          </div>

          <!-- 提供商列表 -->
          <ProviderTable
            :providers="aiStore.providers"
            :loading="aiStore.loading"
            :testing-id="aiStore.testingId"
            @test="handleTest"
            @edit="handleEdit"
            @delete="handleDelete"
            @toggle="handleToggle"
          />

          <!-- 空状态 -->
          <el-empty
            v-if="!aiStore.loading && aiStore.providers.length === 0"
            description="暂无 AI 模型提供商，请添加"
          >
            <el-button type="primary" @click="handleCreate">添加提供商</el-button>
          </el-empty>
        </el-card>

        <!-- 创建/编辑对话框 -->
        <ProviderFormDialog
          v-model:visible="dialogVisible"
          :is-edit="isEdit"
          :provider-data="currentProvider"
          :submit-loading="submitLoading"
          @confirm="handleDialogConfirm"
        />
      </el-tab-pane>

      <!-- 模型路由 -->
      <el-tab-pane name="router">
        <template #label>
          <span class="tab-label"><el-icon><Share /></el-icon>模型路由</span>
        </template>
        <Router />
      </el-tab-pane>

      <!-- 外部 MCP -->
      <el-tab-pane name="mcp">
        <template #label>
          <span class="tab-label"><el-icon><Connection /></el-icon>外部 MCP</span>
        </template>
        <ExternalMCPConfig />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Plus, Refresh, Cpu, Share, Connection } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useAIStore } from '@/stores/ai'
import type { AIProvider } from '@/types/ai'
import ProviderTable from './components/ProviderTable.vue'
import ProviderFormDialog from './components/ProviderFormDialog.vue'
import Router from './Router.vue'
import ExternalMCPConfig from '../components/ExternalMCPConfig.vue'

const activeTab = ref('providers')
const aiStore = useAIStore()

// 对话框状态
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentProvider = ref<Partial<AIProvider> | null>(null)
const submitLoading = ref(false)

// 创建提供商
const handleCreate = () => {
  isEdit.value = false
  currentProvider.value = null
  dialogVisible.value = true
}

// 编辑提供商
const handleEdit = (row: AIProvider) => {
  isEdit.value = true
  currentProvider.value = row
  dialogVisible.value = true
}

// 删除提供商
const handleDelete = async (id: number) => {
  try {
    await aiStore.removeProvider(id)
    ElMessage.success('删除成功')
  } catch {
    // 错误已在请求拦截器中处理
  }
}

// 切换启用状态
const handleToggle = async (row: AIProvider) => {
  const action = row.enabled ? '启用' : '停用'
  try {
    await aiStore.toggleProvider(row.id)
    ElMessage.success(`${action}成功`)
  } catch {
    // API 失败时回滚 switch 状态
    row.enabled = !row.enabled
  }
}

// 测试连接
const handleTest = async (row: AIProvider) => {
  try {
    const result = await aiStore.testConnection(row.id)
    if (result.success) {
      ElMessage.success(`连接成功，耗时 ${result.responseTime || 0}ms`)
    } else {
      ElMessage.warning(`连接失败：${result.message}`)
    }
  } catch {
    // 错误已在请求拦截器中处理
  }
}

// 对话框确认
const handleDialogConfirm = async (data: Record<string, unknown>) => {
  submitLoading.value = true
  try {
    if (isEdit.value) {
      await aiStore.editProvider(data.id as number, data as Partial<AIProvider>)
      ElMessage.success('编辑成功')
    } else {
      await aiStore.addProvider(data as Omit<AIProvider, 'id' | 'status' | 'lastTestAt' | 'createdAt'>)
      ElMessage.success('添加成功')
    }
    dialogVisible.value = false
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    submitLoading.value = false
  }
}

// 刷新
const handleRefresh = () => {
  aiStore.loadProviders()
}

onMounted(() => {
  aiStore.loadProviders()
})
</script>

<style scoped>
.providers-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  padding-left: 24px;
}

.config-tabs {
  --el-tabs-header-margin-bottom: 0;
}

.config-tabs :deep(.el-tabs__header) {
  margin: 0;
  background: var(--el-bg-color, #fff);
  border-radius: 8px 8px 0 0;
  padding: 0 4px 0 24px;
}

.config-tabs :deep(.el-tabs__nav-wrap::after) {
  height: 1px;
  background: var(--el-border-color-lighter, #e4e7ed);
}

.config-tabs :deep(.el-tabs__item) {
  font-size: 14px;
  font-weight: 500;
  height: 48px;
  line-height: 48px;
  padding: 0 20px;
}

.tab-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.tab-label .el-icon {
  font-size: 15px;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding-bottom: var(--space-5);
  flex-wrap: wrap;
  gap: var(--space-4);
}

.toolbar-left {
  flex: 1;
  min-width: 200px;
}

.toolbar-right {
  display: flex;
  gap: var(--space-3);
  flex-shrink: 0;
}

.page-title {
  margin: 0 0 var(--space-1);
  font-size: 18px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.page-desc {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.el-empty {
  padding: var(--space-10) 0;
}
</style>
