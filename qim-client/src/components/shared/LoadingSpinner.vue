<!--
  LoadingSpinner.vue - 通用加载状态组件

  功能：
  - 显示加载动画
  - 支持自定义提示文本
  - 支持自定义尺寸

  使用示例：
  <LoadingSpinner />
  <LoadingSpinner text="加载中..." size="large" />
-->
<template>
  <div class="loading-state" role="status" aria-live="polite">
    <div :class="['loading-spinner', sizeClass]"></div>
    <span v-if="text" class="loading-text">{{ text }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  text?: string
  size?: 'small' | 'medium' | 'large'
}

const props = withDefaults(defineProps<Props>(), {
  text: '加载中...',
  size: 'medium'
})

const sizeClass = computed(() => `spinner-${props.size}`)
</script>

<style scoped>
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-12) 0;
  color: var(--text-secondary);
}

.loading-spinner {
  border: 3px solid var(--border-color);
  border-top: 3px solid var(--primary-color);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: var(--spacing-4);
}

.spinner-small {
  width: 24px;
  height: 24px;
}

.spinner-medium {
  width: 40px;
  height: 40px;
}

.spinner-large {
  width: 56px;
  height: 56px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.loading-text {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}
</style>
