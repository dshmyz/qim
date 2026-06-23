<template>
  <div v-if="count > 0" class="at-mention-banner" @click="$emit('navigate')">
    <AtMentionBadge :is-at-mention="true" />
    <span class="banner-text">你有 <strong>{{ count }}</strong> 条未读的 @ 消息</span>
    <span class="banner-action">点击查看 <i class="fas fa-chevron-down"></i></span>
  </div>
</template>

<script setup lang="ts">
import AtMentionBadge from './AtMentionBadge.vue'

defineProps<{
  count: number
}>()

defineEmits<{
  navigate: []
}>()
</script>

<style scoped>
.at-mention-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: var(--primary-light, rgba(59, 130, 246, 0.08));
  border-bottom: 1px solid rgba(59, 130, 246, 0.18);
  cursor: pointer;
  transition: all 0.2s;
}

.at-mention-banner:hover {
  background: rgba(59, 130, 246, 0.12);
}

.banner-text {
  flex: 1;
  font-size: 13px;
  color: var(--text-secondary);
}

.banner-text strong {
  color: var(--primary-color, #2563eb);
}

.banner-action {
  font-size: 12px;
  color: var(--primary-color, #2563eb);
  opacity: 0.8;
}
</style>
