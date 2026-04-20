<template>
  <div class="system-message">
    <span class="system-message-content">{{ content }}</span>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  content: string
}>()
</script>

<style scoped>
.system-message {
  display: flex;
  justify-content: center;
  margin: 12px 0;
}

.system-message-content {
  padding: 4px 12px;
  background: rgba(150, 150, 150, 0.15);
  color: #999999;
  font-size: 12px;
  border-radius: 12px;
  line-height: 1.4;
}
</style>
