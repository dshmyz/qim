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
  background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
  border-bottom: 1px solid #f59e0b;
  cursor: pointer;
  transition: all 0.2s;
}

.at-mention-banner:hover {
  background: linear-gradient(135deg, #fde68a 0%, #fcd34d 100%);
}

.banner-text {
  flex: 1;
  font-size: 13px;
  color: #92400e;
}

.banner-text strong {
  color: #b45309;
}

.banner-action {
  font-size: 12px;
  color: #b45309;
  opacity: 0.8;
}
</style>
