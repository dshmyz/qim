<template>
  <el-tag :type="tagType" size="small" round>
    {{ label }}
  </el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface StatusMap {
  [key: string]: { label: string; type: 'success' | 'warning' | 'danger' | 'info' }
}

interface Props {
  status: string
  map?: StatusMap
}

const props = withDefaults(defineProps<Props>(), {
  map: () => ({
    active: { label: '正常', type: 'success' },
    inactive: { label: '停用', type: 'info' },
    banned: { label: '封禁', type: 'danger' },
    published: { label: '已发布', type: 'success' },
    draft: { label: '草稿', type: 'info' },
  })
})

const tagType = computed(() => props.map[props.status]?.type || 'info')
const label = computed(() => props.map[props.status]?.label || props.status)
</script>
