<template>
  <div class="router-page">
    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <h2 class="page-title">模型路由</h2>
        <p class="page-desc">
          配置「任务类型 → Provider / 模型」映射，保存后立即生效，无需重启。
          <template v-if="usingDb">
            <el-tag type="warning" size="small" class="db-tag">当前使用数据库覆盖</el-tag>
          </template>
          <template v-else>
            <el-tag size="small" class="db-tag">当前使用 config.yaml 默认路由</el-tag>
          </template>
        </p>
      </div>
      <div class="toolbar-right">
        <el-button @click="handleLoad" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button :disabled="!usingDb" @click="handleClear">
          <el-icon><RefreshLeft /></el-icon>
          恢复 config.yaml 默认
        </el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">
          <el-icon><Check /></el-icon>
          保存路由
        </el-button>
      </div>
    </div>

    <!-- 默认任务 -->
    <el-card shadow="never" class="block-card">
      <template #header>
        <div class="card-header">
          <span>默认任务（Default Task）</span>
          <el-tooltip content="未在路由表中配置的任务，将回退到默认任务的 Provider / 模型" placement="top">
            <el-icon class="hint"><QuestionFilled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-select v-model="defaultTask" style="width: 260px">
        <el-option
          v-for="opt in TASK_TYPE_OPTIONS"
          :key="opt.value"
          :label="opt.label"
          :value="opt.value"
        />
      </el-select>
    </el-card>

    <!-- 路由表 -->
    <el-card shadow="never" class="block-card">
      <template #header>
        <div class="card-header">
          <span>任务路由表</span>
          <span class="card-hint">Provider 来自「AI 供应商」页；模型可在已登记模型中选择或自由输入</span>
        </div>
      </template>
      <RouterRouteTable v-model="routes" :loading="loading" :providers="providerCandidates" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { Refresh, RefreshLeft, Check, QuestionFilled } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getAIRouter, saveAIRouter, clearAIRouter } from '@/api/ai'
import type { TaskType, AIRoute, AIRouterConfig } from '@/types/ai'
import { TASK_TYPE_OPTIONS } from '@/types/ai'
import RouterRouteTable from './components/RouterRouteTable.vue'

const loading = ref(false)
const saving = ref(false)
const usingDb = ref(false)
const defaultTask = ref<TaskType>('chat')

// 路由表：key 为任务类型，值为 {provider, model, fallback}
const routes = ref<Record<TaskType, AIRoute>>(createEmptyRoutes())
// Provider 下拉候选（来自 GET /admin/ai/router 返回的 providers）
const providerCandidates = ref<Array<{ id: number; name: string; models: string[] }>>([])

function createEmptyRoutes(): Record<TaskType, AIRoute> {
  const out = {} as Record<TaskType, AIRoute>
  for (const opt of TASK_TYPE_OPTIONS) {
    out[opt.value] = { provider: '', model: '', fallback: [] }
  }
  return out
}

// 加载当前生效路由
async function handleLoad() {
  loading.value = true
  try {
    const { data } = await getAIRouter()
    const resp = data.data
    usingDb.value = resp.usingDb
    defaultTask.value = resp.defaultTask || 'chat'
    // 合并后端返回的路由，未配置的任务保持空占位
    const merged = createEmptyRoutes()
    for (const opt of TASK_TYPE_OPTIONS) {
      const kv = resp.routes[opt.value]
      if (kv && (kv.provider || kv.model)) {
        merged[opt.value] = {
          provider: kv.provider || '',
          model: kv.model || '',
          fallback: kv.fallback || [],
        }
      }
    }
    routes.value = merged
    providerCandidates.value = resp.providers || []
  } catch {
    // 错误已在请求拦截器中提示
  } finally {
    loading.value = false
  }
}

// 保存路由（只提交用户配置过 provider/model 的条目）
async function handleSave() {
  const clean = {} as Record<TaskType, AIRoute>
  let hasRoute = false
  for (const opt of TASK_TYPE_OPTIONS) {
    const r = routes.value[opt.value]
    if (r && (r.provider || r.model)) {
      clean[opt.value] = {
        provider: r.provider || '',
        model: r.model || '',
        fallback: r.fallback?.filter(Boolean) || [],
      }
      hasRoute = true
    }
  }
  if (!hasRoute) {
    ElMessage.warning('请至少为一个任务类型配置 Provider / 模型')
    return
  }

  const payload: AIRouterConfig = { defaultTask: defaultTask.value, routes: clean }
  saving.value = true
  try {
    const { data } = await saveAIRouter(payload)
    usingDb.value = data.data.usingDb
    ElMessage.success('路由已保存并立即生效')
    await handleLoad()
  } catch {
    // 校验不通过（如引用了未配置的供应商）时拦截器已提示
  } finally {
    saving.value = false
  }
}

// 恢复 config.yaml 默认
async function handleClear() {
  try {
    await ElMessageBox.confirm('确定清除数据库路由覆盖，恢复为 config.yaml 默认路由？', '恢复默认', {
      type: 'warning',
      confirmButtonText: '恢复默认',
      cancelButtonText: '取消',
    })
  } catch {
    return // 用户取消
  }
  try {
    const { data } = await clearAIRouter()
    usingDb.value = data.data.usingDb
    ElMessage.success('已恢复 config.yaml 默认路由')
    await handleLoad()
  } catch {
    // 错误已在拦截器中提示
  }
}

onMounted(handleLoad)
</script>

<style scoped>
.router-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding-bottom: var(--space-5);
  padding-left: 24px;
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
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.db-tag {
  margin-left: 4px;
}

.block-card {
  margin-bottom: var(--space-2);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: 600;
}

.card-hint {
  font-weight: 400;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.hint {
  color: var(--el-text-color-secondary);
  cursor: help;
}
</style>
