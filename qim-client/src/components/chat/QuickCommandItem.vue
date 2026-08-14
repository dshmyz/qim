<template>
  <!-- 快速回复候选项：闪电图标 + 短语文本 -->
  <span class="quick-cmd-item">
    <span class="quick-cmd-item__icon">
      <i class="fas fa-bolt"></i>
    </span>
    <span class="quick-cmd-item__text">{{ text }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface QuickReplyItem {
  id: string
  text: string
}

const props = defineProps<{
  item: QuickReplyItem
  active: boolean
}>()

const text = computed(() => props.item.text)
</script>

<style scoped>
.quick-cmd-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.quick-cmd-item__icon {
  font-size: var(--font-size-xxs);
  color: var(--primary-color, #3385ff);
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  width: 14px;
}
.quick-cmd-item__text {
  flex: 1;
  font-size: var(--font-size-xs);
  color: var(--text-color, #303133);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
