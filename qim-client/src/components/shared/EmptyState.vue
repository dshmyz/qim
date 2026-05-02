<!--
  EmptyState.vue - 通用空状态组件

  功能：
  - 显示空状态提示
  - 支持自定义图标、标题和描述
  - 支持可选的操作按钮

  使用示例：
  <EmptyState
    icon="fa-bullhorn"
    title="暂无数据"
    description="还没有任何内容"
  />
  <EmptyState
    icon="fa-bullhorn"
    title="暂无订阅频道"
    description="去频道广场订阅感兴趣的频道吧！"
    action-text="浏览频道广场"
    @action="handleAction"
  />
-->
<template>
  <div class="empty-state" role="status" aria-live="polite">
    <i :class="['empty-icon', 'fas', icon]"></i>
    <h4 class="empty-title">{{ title }}</h4>
    <p class="empty-description">{{ description }}</p>
    <button
      v-if="actionText"
      class="empty-action-btn"
      @click="$emit('action')"
      :aria-label="actionText"
    >
      {{ actionText }}
    </button>
  </div>
</template>

<script setup lang="ts">
interface Props {
  icon: string
  title: string
  description: string
  actionText?: string
}

defineProps<Props>()

defineEmits<{
  action: []
}>()
</script>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-12) var(--spacing-4);
  text-align: center;
}

.empty-icon {
  font-size: 48px;
  color: var(--primary-color);
  margin-bottom: var(--spacing-4);
  opacity: 0.5;
}

.empty-title {
  margin: 0 0 var(--spacing-2) 0;
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
}

.empty-description {
  margin: 0 0 var(--spacing-4) 0;
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.empty-action-btn {
  padding: var(--spacing-2) var(--spacing-4);
  border: 1px solid var(--primary-color);
  border-radius: var(--radius-md);
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  transition: all var(--transition-fast);
}

.empty-action-btn:hover {
  background: var(--primary-dark);
  border-color: var(--primary-dark);
  transform: translateY(-1px);
}

.empty-action-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}
</style>
