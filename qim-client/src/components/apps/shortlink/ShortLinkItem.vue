<template>
  <div class="short-link-item" @mouseenter="isHovered = true" @mouseleave="isHovered = false">
    <!-- 选择框 -->
    <div v-if="selectMode" class="link-checkbox">
      <input
        type="checkbox"
        :checked="isSelected"
        @change="handleSelectChange"
      />
    </div>

    <div class="link-info">
      <div class="original-url">{{ link.original_url }}</div>
      <div class="link-meta">
        <a :href="link.short_url" target="_blank" class="short-url">{{ link.short_url }}</a>
        <span class="created-time">创建于 {{ formatDate(link.created_at) }}</span>
      </div>
    </div>

    <div class="visit-count">
      <div class="count-value">{{ link.visit_count }}</div>
      <div class="count-label">访问</div>
    </div>

    <div class="link-actions">
      <button class="action-btn copy-btn" @click="$emit('copy', link.short_url)">
        <i class="fas fa-copy"></i> 复制
      </button>
      <button class="action-btn delete-btn" @click="$emit('delete', link.id)">
        <i class="fas fa-trash"></i> 删除
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

export interface ShortLink {
  id: number
  original_url: string
  short_url: string
  visit_count: number
  created_at: string
}

const props = defineProps<{
  link: ShortLink
  selectMode?: boolean
  isSelected?: boolean
}>()

const emit = defineEmits<{
  copy: [url: string]
  delete: [id: number]
  'select-change': [id: number, selected: boolean]
}>()

const isHovered = ref(false)

const formatDate = (dateString: string) => {
  const date = new Date(dateString)
  return date.toLocaleDateString('zh-CN')
}

const handleSelectChange = (e: Event) => {
  const target = e.target as HTMLInputElement
  emit('select-change', props.link.id, target.checked)
}
</script>

<style scoped>
.short-link-item {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color, #f3f4f6);
  display: flex;
  align-items: center;
  gap: 16px;
  transition: background 0.2s;
  cursor: pointer;
}

.short-link-item:hover {
  background: var(--hover-bg, #f9fafb);
}

.link-checkbox {
  display: flex;
  align-items: center;
  padding-right: 8px;
}

.link-checkbox input {
  width: 16px;
  height: 16px;
  cursor: pointer;
}

.link-info {
  flex: 1;
  min-width: 0;
}

.original-url {
  font-size: 14px;
  color: var(--text-primary, #1f2937);
  margin-bottom: 4px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.link-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
}

.short-url {
  color: var(--accent-color, #667eea);
  text-decoration: none;
  font-weight: 500;
}

.short-url:hover {
  text-decoration: underline;
}

.created-time {
  color: var(--text-secondary, #9ca3af);
}

.visit-count {
  text-align: center;
  min-width: 80px;
}

.count-value {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary, #1f2937);
}

.count-label {
  font-size: 11px;
  color: var(--text-secondary, #6b7280);
}

.link-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  padding: 6px 12px;
  border: none;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 4px;
}

.copy-btn {
  background: var(--accent-bg, rgba(102, 126, 234, 0.1));
  color: var(--accent-color, #667eea);
}

.copy-btn:hover {
  background: var(--accent-bg-hover, rgba(102, 126, 234, 0.2));
  transform: translateY(-1px);
}

.delete-btn {
  background: var(--danger-bg, rgba(239, 68, 68, 0.1));
  color: var(--danger-color, #ef4444);
}

.delete-btn:hover {
  background: var(--danger-bg-hover, rgba(239, 68, 68, 0.2));
  transform: translateY(-1px);
}
</style>
