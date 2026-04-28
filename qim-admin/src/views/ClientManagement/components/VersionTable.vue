<template>
  <el-table :data="versions" v-loading="loading" stripe>
    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column prop="version" label="版本号" width="120" />
    <el-table-column label="平台" width="100">
      <template #default="{ row }">
        <el-tag :type="getPlatformType(row.platform)" size="small">
          {{ getPlatformLabel(row.platform) }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="updateNotes" label="更新日志" min-width="200" show-overflow-tooltip />
    <el-table-column label="强制更新" width="100" align="center">
      <template #default="{ row }">
        <el-tag :type="row.forceUpdate ? 'danger' : 'info'" size="small">
          {{ row.forceUpdate ? '是' : '否' }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column label="灰度发布" width="100" align="center">
      <template #default="{ row }">
        <template v-if="row.rolloutPercentage >= 100">
          <el-tag type="success" size="small">全量</el-tag>
        </template>
        <template v-else-if="row.rolloutPercentage > 0">
          <el-tag type="warning" size="small">{{ row.rolloutPercentage }}%</el-tag>
        </template>
        <template v-else>
          <el-tag type="info" size="small">关闭</el-tag>
        </template>
      </template>
    </el-table-column>
    <el-table-column label="状态" width="100" align="center">
      <template #default="{ row }">
        <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
          {{ row.status === 'active' ? '启用' : '停用' }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="releaseDate" label="发布时间" width="120" />
    <el-table-column label="操作" width="160" fixed="right">
      <template #default="{ row }">
        <el-button size="small" @click="$emit('edit', row)">编辑</el-button>
        <el-popconfirm title="确定删除该版本吗？" @confirm="$emit('delete', row.id)">
          <template #reference>
            <el-button size="small" type="danger">删除</el-button>
          </template>
        </el-popconfirm>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import type { ClientVersion } from '@/types/client'

interface Props {
  versions: ClientVersion[]
  loading?: boolean
}

withDefaults(defineProps<Props>(), {
  loading: false,
})

defineEmits<{
  'edit': [version: ClientVersion]
  'delete': [id: number]
}>()

function getPlatformType(platform: string): 'primary' | 'success' | 'warning' {
  const typeMap: Record<string, 'primary' | 'success' | 'warning'> = {
    windows: 'primary',
    macos: 'success',
    linux: 'warning',
  }
  return typeMap[platform] || 'primary'
}

function getPlatformLabel(platform: string): string {
  const labelMap: Record<string, string> = {
    windows: 'Windows',
    macos: 'macOS',
    linux: 'Linux',
  }
  return labelMap[platform] || platform
}
</script>
