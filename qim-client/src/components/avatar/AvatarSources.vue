<template>
  <div v-if="sources && sources.length > 0" class="avatar-sources">
    <span class="avatar-sources-label">
      <svg viewBox="0 0 24 24" width="11" height="11" fill="currentColor" aria-hidden="true">
        <path d="M11 2a10 10 0 100 20 10 10 0 000-20zm0 3a7 7 0 110 14 7 7 0 010-14zm-1 4v4l3 2 .8-1.2-2.3-1.5V9H10z"/>
      </svg>
      依据
    </span>
    <span v-for="(src, i) in sources" :key="i" class="avatar-source-tag" :title="src.snippet || ''">
      {{ label(src.type) }}
      <template v-if="src.title">《{{ src.title }}》</template>
    </span>
  </div>
</template>

<script setup lang="ts">
import type { AvatarSource } from '../../types'

defineProps<{
  sources?: AvatarSource[]
}>()

// 来源类型文案：没有标题（如记忆）时退化为类型名本身
function label(type: AvatarSource['type']): string {
  switch (type) {
    case 'note': return '笔记'
    case 'group': return '群知识'
    case 'memory': return '记忆'
    default: return '资料'
  }
}
</script>

<style scoped>
.avatar-sources {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px 8px;
  margin-top: 4px;
  font-size: 11px;
  line-height: 1.4;
  color: var(--text-secondary, #666);
  opacity: 0.75;
}

.avatar-sources-label {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: inherit;
}

.avatar-source-tag {
  display: inline-flex;
  align-items: center;
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 0 6px;
  border-radius: 8px;
  background: rgba(59, 130, 246, 0.08);
  color: #4f7cff;
}

[data-theme="elegant-dark"] .avatar-source-tag {
  background: rgba(59, 130, 246, 0.16);
  color: #7aa2ff;
}
</style>
