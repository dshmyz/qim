<template>
  <el-table :data="providers" v-loading="loading" stripe>
    <el-table-column prop="id" label="ID" width="70" />

    <el-table-column label="提供商名称" min-width="160">
      <template #default="{ row }">
        <div class="provider-cell">
          <el-tag :type="providerTypeTag(row.type)" size="small" class="provider-type-tag">
            {{ providerTypeLabel(row.type) }}
          </el-tag>
          <span class="provider-name">{{ row.name }}</span>
        </div>
      </template>
    </el-table-column>

    <el-table-column prop="apiEndpoint" label="API 端点" min-width="220" show-overflow-tooltip />

    <el-table-column label="支持模型" min-width="200">
      <template #default="{ row }">
        <div class="models-cell">
          <el-tag
            v-for="model in row.models"
            :key="model"
            size="small"
            class="model-tag"
          >
            {{ model }}
          </el-tag>
          <span v-if="row.models.length === 0" class="no-models">未配置</span>
        </div>
      </template>
    </el-table-column>

    <el-table-column label="状态" width="120" align="center">
      <template #default="{ row }">
        <el-tooltip :content="statusTooltip(row)">
          <el-tag :type="statusTagType(row.status)" size="small">
            {{ statusLabel(row.status) }}
          </el-tag>
        </el-tooltip>
      </template>
    </el-table-column>

    <el-table-column label="启用" width="80" align="center">
      <template #default="{ row }">
        <el-switch
          v-model="row.enabled"
          size="small"
          @change="handleToggle(row)"
        />
      </template>
    </el-table-column>

    <el-table-column label="优先级" width="80" align="center">
      <template #default="{ row }">
        {{ row.priority }}
      </template>
    </el-table-column>

    <el-table-column label="最后测试时间" width="180">
      <template #default="{ row }">
        <span class="time-text">{{ formatTime(row.lastTestAt) }}</span>
      </template>
    </el-table-column>

    <el-table-column label="操作" width="240" fixed="right">
      <template #default="{ row }">
        <el-button size="small" @click="$emit('test', row)">
          测试连接
        </el-button>
        <el-button size="small" @click="$emit('edit', row)">编辑</el-button>
        <el-popconfirm
          title="确定删除该提供商吗？"
          @confirm="$emit('delete', row.id)"
        >
          <template #reference>
            <el-button size="small" type="danger">删除</el-button>
          </template>
        </el-popconfirm>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import { PROVIDER_TYPE_LABELS, type AIProvider, type ProviderType, type ProviderStatus } from '@/types/ai'

interface Props {
  providers: AIProvider[]
  loading: boolean
  testingId: number | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  test: [row: AIProvider]
  edit: [row: AIProvider]
  delete: [id: number]
  toggle: [row: AIProvider]
}>()

function providerTypeLabel(type: ProviderType): string {
  return PROVIDER_TYPE_LABELS[type] || type
}

function providerTypeTag(type: ProviderType): 'success' | 'warning' | 'info' | 'primary' | 'danger' {
  const map: Record<ProviderType, 'success' | 'warning' | 'info' | 'primary' | 'danger'> = {
    openai: 'success',
    anthropic: 'primary',
    ollama: 'warning',
    azure: 'danger',
    custom: 'info',
  }
  return map[type] || 'info'
}

function statusLabel(status: ProviderStatus): string {
  const map: Record<ProviderStatus, string> = {
    connected: '已连接',
    error: '连接失败',
    testing: '测试中',
    unknown: '未测试',
  }
  return map[status] || '未知'
}

function statusTagType(status: ProviderStatus): 'success' | 'danger' | 'warning' | 'info' {
  const map: Record<ProviderStatus, 'success' | 'danger' | 'warning' | 'info'> = {
    connected: 'success',
    error: 'danger',
    testing: 'warning',
    unknown: 'info',
  }
  return map[status] || 'info'
}

function statusTooltip(row: AIProvider): string {
  if (!row.enabled) {
    return '已停用'
  }
  if (row.status === 'connected') {
    return '连接正常'
  }
  if (row.status === 'error') {
    return '连接失败，请检查配置'
  }
  if (row.status === 'testing') {
    return '正在测试连接...'
  }
  return '尚未测试连接'
}

function formatTime(time?: string): string {
  if (!time) return '未测试'
  const date = new Date(time)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes} 分钟前`
  if (hours < 24) return `${hours} 小时前`
  if (days < 7) return `${days} 天前`
  return date.toLocaleDateString('zh-CN')
}

function handleToggle(row: AIProvider) {
  emit('toggle', row)
}

const emit = defineEmits<{
  test: [row: AIProvider]
  edit: [row: AIProvider]
  delete: [id: number]
  toggle: [row: AIProvider]
}>()
</script>

<style scoped>
.provider-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.provider-type-tag {
  min-width: 70px;
  text-align: center;
}

.provider-name {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.models-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
}

.model-tag {
  margin: 0;
}

.no-models {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.time-text {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
