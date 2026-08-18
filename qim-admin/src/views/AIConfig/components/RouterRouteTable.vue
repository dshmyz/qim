<template>
  <el-table :data="rows" v-loading="loading" style="width: 100%">
    <el-table-column label="任务类型" width="200" fixed>
      <template #default="{ row }">
        <div class="task-cell">
          <span class="task-label">{{ row.label }}</span>
          <span class="task-value">{{ row.value }}</span>
          <span class="task-desc">{{ row.desc }}</span>
        </div>
      </template>
    </el-table-column>

    <el-table-column label="Provider" width="220">
      <template #default="{ row }">
        <el-select
          v-model="row.route.provider"
          placeholder="选择供应商"
          clearable
          filterable
          style="width: 100%"
          :disabled="providerList.length === 0"
        >
          <el-option
            v-for="p in providerList"
            :key="p.name"
            :label="p.name"
            :value="p.name"
          />
        </el-select>
      </template>
    </el-table-column>

    <el-table-column label="模型" min-width="200">
      <template #default="{ row }">
        <el-select
          v-model="row.route.model"
          placeholder="输入或选择模型"
          filterable
          allow-create
          default-first-option
          clearable
          style="width: 100%"
          :no-data-text="noModelText(row)"
        >
          <el-option
            v-for="m in modelsFor(row.route.provider)"
            :key="m"
            :label="m"
            :value="m"
          />
        </el-select>
      </template>
    </el-table-column>

    <el-table-column label="Fallback（备用供应商）" min-width="220">
      <template #default="{ row }">
        <el-select
          v-model="row.route.fallback"
          multiple
          filterable
          allow-create
          default-first-option
          placeholder="可多选备用供应商，留空表示无 fallback"
          style="width: 100%"
        >
          <el-option
            v-for="p in providerList"
            :key="p.name"
            :label="p.name"
            :value="p.name"
          />
        </el-select>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TaskType, AIRoute } from '@/types/ai'
import { TASK_TYPE_OPTIONS } from '@/types/ai'

interface RouteRow {
  value: TaskType
  label: string
  desc: string
  route: AIRoute
}

const props = defineProps<{
  modelValue: Record<TaskType, AIRoute>
  loading?: boolean
  providers?: Array<{ name: string; models: string[] }>
}>()

const providerList = computed(() => props.providers || [])

const emit = defineEmits<{
  (e: 'update:modelValue', value: Record<TaskType, AIRoute>): void
}>()

// 数据行与 modelValue 双向同步：编辑直接改 row.route（引用共享）
const rows = computed<RouteRow[]>(() =>
  TASK_TYPE_OPTIONS.map((opt) => ({
    value: opt.value,
    label: opt.label,
    desc: opt.desc,
    route: props.modelValue[opt.value] || { provider: '', model: '', fallback: [] },
  }))
)

// 当前选中 provider 的已登记模型（用于模型下拉候选）
function modelsFor(providerName: string): string[] {
  const p = (props.providers || []).find((x) => x.name === providerName)
  return p ? p.models : []
}

function noModelText(row: RouteRow): string {
  if (!row.route.provider) return '先选择 Provider'
  const ms = modelsFor(row.route.provider)
  if (!ms.length) return '该供应商未登记模型，可自由输入'
  return '无匹配模型，可自由输入'
}

// 通过事件对外暴露「mx」同步方法（由父组件在 v-model 变更时调用）
defineExpose({
  sync(target: Record<TaskType, AIRoute>) {
    emit('update:modelValue', target)
  },
})
</script>

<style scoped>
.task-cell {
  display: flex;
  flex-direction: column;
  line-height: 1.4;
}
.task-label {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.task-value {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-family: 'SFMono-Regular', Consolas, monospace;
}
.task-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
