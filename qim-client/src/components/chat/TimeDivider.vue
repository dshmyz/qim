<script setup lang="ts">
interface Props {
  timestamp: number
  formatTime: (timestamp: number) => string
}

defineProps<Props>()
</script>

<template>
  <div class="time-divider">
    <span class="divider-line" />
    <span class="divider-badge">{{ formatTime(timestamp) }}</span>
    <span class="divider-line" />
  </div>
</template>

<style scoped>
.time-divider {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 16px 0;
}

.divider-line {
  flex: 1;
  height: 1px;
  background-color: var(--color-gray-200);
}

.divider-badge {
  flex-shrink: 0;
  padding: 4px 12px;
  font-size: 12px;
  line-height: 1;
  color: var(--color-gray-500);
  background-color: var(--color-gray-200);
  border-radius: 12px;
  white-space: nowrap;
  user-select: none;
}
</style>
